package integration

import (
	"bytes"
	"strings"
	"testing"

	"gatra-budget-backend/models"
	"gatra-budget-backend/seeds"

	"gorm.io/gorm"
)

func count(t *testing.T, tx *gorm.DB, table, where string, args ...any) int64 {
	t.Helper()
	var n int64
	q := tx.Table(table)
	if where != "" {
		q = q.Where(where, args...)
	}
	if err := q.Count(&n).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// A server that already has its master data under other IDs must be reused,
// not duplicated or rejected.
func TestDemoSeederReusesExistingMasters(t *testing.T) {
	tx := withTx(t)
	client := models.Client{Name: "ICON+"}
	eos := models.ContractType{Name: "EOS"}
	gaji := models.CostType{NamaBiaya: "Gaji & Tunjangan"}
	for _, m := range []any{&client, &eos, &gaji} {
		if err := tx.Create(m).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := seeds.DemoSeeder(tx); err != nil {
		t.Fatalf("seed with existing masters: %v", err)
	}
	if n := count(t, tx, "clients", ""); n != 1 {
		t.Errorf("clients: %d, want 1", n)
	}
	if n := count(t, tx, "contract_types", ""); n != 2 {
		t.Errorf("contract types: %d, want 2 (EOS reused, TAD added)", n)
	}
	if n := count(t, tx, "cost_types", "LOWER(nama_biaya) = 'gaji & tunjangan'"); n != 1 {
		t.Errorf("Gaji & Tunjangan: %d, want 1", n)
	}
	if n := count(t, tx, "projects", "client_id = ? AND created_by = ?", client.ID, seeds.DemoCreatedBy); n != 8 {
		t.Errorf("%d demo projects on the existing client, want 8", n)
	}
	if n := count(t, tx, "transactions", "cost_type_id = ?", gaji.ID); n == 0 {
		t.Error("salary transactions should use the existing cost type")
	}
}

func TestDemoSeederIsIdempotentAndConsistent(t *testing.T) {
	tx := withTx(t)

	if err := seeds.DemoSeeder(tx); err != nil {
		t.Fatalf("first run: %v", err)
	}
	tables := []string{"project_groups", "projects", "transactions", "cost_operations", "invoices", "cost_types"}
	first := map[string]int64{}
	for _, tb := range tables {
		first[tb] = count(t, tx, tb, "")
	}
	if err := seeds.DemoSeeder(tx); err != nil {
		t.Fatalf("second run: %v", err)
	}
	for _, tb := range tables {
		if n := count(t, tx, tb, ""); n != first[tb] {
			t.Errorf("%s: %d rows after re-run, want %d (not idempotent)", tb, n, first[tb])
		}
	}

	if first["project_groups"] < 2 {
		t.Errorf("want at least 2 groups, got %d", first["project_groups"])
	}
	if count(t, tx, "projects", "project_group_id IS NULL AND created_by = ?", seeds.DemoCreatedBy) == 0 {
		t.Error("want demo projects without a group")
	}

	// Group 1 has the three split contract periods, its value is their sum.
	var g models.ProjectGroup
	if err := tx.Where("nama = ?", "Pengadaan TAD Sumbar").First(&g).Error; err != nil {
		t.Fatal(err)
	}
	var childCount int64
	var total float64
	tx.Raw("SELECT COUNT(*), COALESCE(SUM(nilai_pekerjaan),0) FROM projects WHERE project_group_id = ?", g.ID).Row().Scan(&childCount, &total)
	if childCount != 3 || total != 2_000_000_000 {
		t.Errorf("group Sumbar: %d projects / %.0f, want 3 / 2000000000", childCount, total)
	}

	// Both years are covered by every kind of data.
	for _, q := range []struct{ table, col string }{
		{"transactions", "tanggal_transaksi"}, {"cost_operations", "tanggal_transaksi"}, {"invoices", "tgl_tagihan"},
	} {
		for _, year := range []string{"2025", "2026"} {
			if count(t, tx, q.table, "to_char("+q.col+",'YYYY') = ?", year) == 0 {
				t.Errorf("%s has no rows in %s", q.table, year)
			}
		}
	}

	// Invoice rules: period inside the contract, Kliring has a payment date
	// after the billing date, fee never above value, SPR spans several months.
	var invoices []models.Invoice
	if err := tx.Preload("Project").Where("created_by = ?", seeds.DemoCreatedBy).Find(&invoices).Error; err != nil {
		t.Fatal(err)
	}
	if len(invoices) == 0 {
		t.Fatal("no demo invoices")
	}
	spr, late := 0, 0
	for _, inv := range invoices {
		if !strings.HasPrefix(inv.NoInvoice, seeds.DemoInvoicePrefix) {
			t.Errorf("invoice %s lacks the demo prefix", inv.NoInvoice)
		}
		start, end := inv.Project.TanggalMulai.Format("2006-01"), inv.Project.TanggalSelesai.Format("2006-01")
		for _, p := range inv.PeriodeTagihan {
			if p < start || p > end {
				t.Errorf("%s: period %s outside contract %s..%s", inv.NoInvoice, p, start, end)
			}
		}
		if inv.MetodePembayaran == models.PaymentMethodSPR && len(inv.PeriodeTagihan) > 1 {
			spr++
		}
		if inv.TglTagihan.Format("2006-01") != inv.PeriodeTagihan[0] {
			late++
		}
		if inv.Status == models.InvoiceStatusKliring && (inv.TglBayar == nil || inv.TglBayar.Before(inv.TglTagihan.Time)) {
			t.Errorf("%s: Kliring needs tgl_bayar after tgl_tagihan", inv.NoInvoice)
		}
		if inv.Status == models.InvoiceStatusSubmit && inv.TglBayar != nil {
			t.Errorf("%s: Submit must not have tgl_bayar", inv.NoInvoice)
		}
		if inv.ManajemenFee <= 0 || inv.ManajemenFee > inv.Nilai {
			t.Errorf("%s: fee %.0f outside (0, %.0f]", inv.NoInvoice, inv.ManajemenFee, inv.Nilai)
		}
	}
	if spr == 0 {
		t.Error("want a SPR invoice covering several months")
	}
	if late == 0 {
		t.Error("want invoices whose period differs from the billing month")
	}
	if count(t, tx, "invoices", "status = ?", models.InvoiceStatusSubmit) == 0 {
		t.Error("want some invoices still in Submit")
	}

	// Expenses respect the validation floor and stay below each contract value.
	if n := count(t, tx, "transactions", "jumlah_realisasi <= 100000"); n != 0 {
		t.Errorf("%d transactions at or below 100000", n)
	}
	rows, err := tx.Raw(`SELECT p.no_sp2k, p.nilai_pekerjaan, COALESCE(SUM(t.jumlah_realisasi),0)
		FROM projects p LEFT JOIN transactions t ON t.project_id = p.id
		WHERE p.created_by = ? GROUP BY p.id`, seeds.DemoCreatedBy).Rows()
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var sp2k string
		var nilai, spent float64
		rows.Scan(&sp2k, &nilai, &spent)
		if spent <= 0 || spent > nilai*0.97 {
			t.Errorf("%s: spent %.0f of %.0f is not a realistic absorption", sp2k, spent, nilai)
		}
	}
}

func TestDemoResetOnlyRemovesDemoRows(t *testing.T) {
	tx := withTx(t)
	if err := seeds.DemoSeeder(tx); err != nil {
		t.Fatal(err)
	}

	// Real data: a project with a transaction, attached to a demo group, and
	// a real invoice on a demo project.
	real, _ := newTestProject(t, tx)
	var g models.ProjectGroup
	tx.Where("nama = ?", "Jasa EOS Sumut").First(&g)
	tx.Model(&models.Project{}).Where("id = ?", real.ID).Update("project_group_id", g.ID)
	realInvoice := newInvoice(t, real.ID, "REAL-001")
	if err := tx.Create(&realInvoice).Error; err != nil {
		t.Fatal(err)
	}

	run := func(answers string) string {
		var out bytes.Buffer
		if err := seeds.DemoReset(tx, strings.NewReader(answers), &out, false); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}

	demoInvoices := func() int64 { return count(t, tx, "invoices", "created_by = ?", seeds.DemoCreatedBy) }
	demoTrx := func() int64 { return count(t, tx, "transactions", "created_by = ?", seeds.DemoCreatedBy) }
	demoOps := func() int64 { return count(t, tx, "cost_operations", "created_by = ?", seeds.DemoCreatedBy) }
	if demoInvoices() == 0 || demoTrx() == 0 || demoOps() == 0 {
		t.Fatal("seed produced no demo rows")
	}

	// Only invoices (y), everything else no / blank / end of input.
	run("y\nn\n\n")
	if demoInvoices() != 0 {
		t.Error("demo invoices should be gone")
	}
	if demoTrx() == 0 || demoOps() == 0 {
		t.Error("transactions / operational expenses must stay when answered no")
	}
	if count(t, tx, "invoices", "no_invoice = ?", realInvoice.NoInvoice) != 1 {
		t.Error("real invoice must survive")
	}

	// Answer yes to everything: demo data goes, real data and its group stay.
	out := run("y\ny\ny\ny\ny\ny\n")
	if demoTrx() != 0 || demoOps() != 0 || count(t, tx, "projects", "created_by = ?", seeds.DemoCreatedBy) != 0 {
		t.Errorf("demo rows remain after yes to all:\n%s", out)
	}
	if count(t, tx, "projects", "id = ?", real.ID) != 1 || count(t, tx, "invoices", "no_invoice = ?", realInvoice.NoInvoice) != 1 {
		t.Error("real project / invoice must survive")
	}
	if count(t, tx, "project_groups", "id = ?", g.ID) != 1 {
		t.Error("group still used by a real project must be kept")
	}
	if count(t, tx, "project_groups", "nama = ?", "Pengadaan TAD Sumbar") != 0 {
		t.Error("unused demo group should be removed")
	}
	if !strings.Contains(out, "dipertahankan") {
		t.Errorf("expected a note about the kept group:\n%s", out)
	}
	if count(t, tx, "operational_cost_types", "") != 6 {
		t.Error("operational cost types are base data and must stay")
	}
}
