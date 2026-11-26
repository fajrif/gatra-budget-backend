package integration

import (
	"strings"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"

	"github.com/google/uuid"
)

func newInvoice(t *testing.T, projectID uuid.UUID, no string) models.Invoice {
	t.Helper()
	return models.Invoice{
		NoInvoice:   no,
		ProjectID:   projectID,
		NamaInvoice: "Tagihan",
		TglTagihan:  models.Date{Time: mustParseDate(t, "2026-10-01")},
		Status:      models.InvoiceStatusSubmit,
		Nilai:       100_000_000,

		MetodePembayaran: models.PaymentMethodKontrak,
		PeriodeTagihan:   models.MonthList{"2026-10"},
	}
}

func TestInvoiceNoInvoiceIsUnique(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)

	first := newInvoice(t, project.ID, "INV-U-1")
	if err := tx.Create(&first).Error; err != nil {
		t.Fatalf("failed to create first invoice: %v", err)
	}

	// A failed statement aborts the surrounding transaction, so use a savepoint.
	tx.Exec("SAVEPOINT dup")
	second := newInvoice(t, project.ID, "INV-U-1")
	if err := tx.Create(&second).Error; err == nil {
		t.Fatal("expected the UNIQUE constraint to reject a duplicate no_invoice")
	}
	tx.Exec("ROLLBACK TO SAVEPOINT dup")
}

func TestInvoiceCheckConstraints(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)

	bayar := models.Date{Time: mustParseDate(t, "2026-10-20")}

	cases := []struct {
		name   string
		mutate func(*models.Invoice)
	}{
		{"Kliring without tgl_bayar", func(i *models.Invoice) { i.Status = models.InvoiceStatusKliring }},
		{"Submit with tgl_bayar", func(i *models.Invoice) { i.TglBayar = &bayar }},
		{"Kliring without nilai_penerimaan", func(i *models.Invoice) {
			i.Status = models.InvoiceStatusKliring
			i.TglBayar = &bayar
		}},
		{"Submit with nilai_penerimaan", func(i *models.Invoice) {
			paid := i.Nilai
			i.NilaiPenerimaan = &paid
		}},
		{"negative nilai_penerimaan", func(i *models.Invoice) {
			neg := -1.0
			i.Status = models.InvoiceStatusKliring
			i.TglBayar = &bayar
			i.NilaiPenerimaan = &neg
		}},
		{"fee above nilai", func(i *models.Invoice) { i.ManajemenFee = i.Nilai + 1 }},
		{"unknown status", func(i *models.Invoice) { i.Status = "Paid" }},
		{"unknown metode", func(i *models.Invoice) { i.MetodePembayaran = "Cash" }},
		{"empty periode", func(i *models.Invoice) { i.PeriodeTagihan = models.MonthList{} }},
		{"Kontrak with two months", func(i *models.Invoice) { i.PeriodeTagihan = models.MonthList{"2026-09", "2026-10"} }},
	}
	for n, tc := range cases {
		inv := newInvoice(t, project.ID, "INV-C-"+string(rune('A'+n)))
		tc.mutate(&inv)

		tx.Exec("SAVEPOINT chk")
		if err := tx.Create(&inv).Error; err == nil {
			t.Errorf("%s: expected the database CHECK constraint to reject the row", tc.name)
		}
		tx.Exec("ROLLBACK TO SAVEPOINT chk")
	}
}

func TestInvoiceProjectCascadeDeletesInvoices(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)

	inv := newInvoice(t, project.ID, "INV-D-1")
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatalf("failed to create invoice: %v", err)
	}
	if err := tx.Delete(&models.Project{}, "id = ?", project.ID).Error; err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	var count int64
	if err := tx.Model(&models.Invoice{}).Where("project_id = ?", project.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count invoices: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected ON DELETE CASCADE to remove the project's invoices, found %d", count)
	}
}

// TestInvoiceCreateAndClearAgainstRealDB runs the real handlers through a
// Submit -> Kliring transition: the invoice is billed first, then cleared with
// its received amount and one deduction.
func TestInvoiceCreateAndClearAgainstRealDB(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	pph := newDeductionType(t, tx, "PPh Badan")

	a := app()
	a.Post("/invoices", handlers.CreateInvoice)
	a.Put("/invoices/:id", handlers.UpdateInvoice)
	a.Put("/invoices/:id/kliring", handlers.KliringInvoice)

	create := `{"no_invoice":"INV-H-1","project_id":"` + project.ID.String() + `","nama_invoice":"Termin 1",
		"tgl_tagihan":"2026-10-01","metode_pembayaran":"Kontrak","periode_tagihan":["2026-09"],"nilai":100000000,"manajemen_fee":5000000}`

	resp := doJSON(t, a, "POST", "/invoices", create)
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	created := readBody(t, resp)
	if created["status"] != "Submit" {
		t.Fatalf("a new invoice must be Submit, got %v", created["status"])
	}
	id := created["id"].(string)

	kliring := `{"tgl_bayar":"2026-10-25","nilai_penerimaan":98000000,
		"potongan":[{"deduction_type_id":"` + pph.ID.String() + `","jumlah":2000000}]}`
	resp = doJSON(t, a, "PUT", "/invoices/"+id+"/kliring", kliring)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 clearing the invoice, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	cleared := readBody(t, resp)
	if cleared["status"] != "Kliring" || cleared["tgl_bayar"] != "2026-10-25" || cleared["nilai_penerimaan"] != 98_000_000.0 {
		t.Fatalf("unexpected clearing result: %v", cleared)
	}
	deductions := cleared["deductions"].([]any)
	if len(deductions) != 1 || deductions[0].(map[string]any)["jumlah"] != 2_000_000.0 {
		t.Fatalf("expected one 2.000.000 deduction, got %v", deductions)
	}

	// Editing the billed amount of a cleared invoice is refused.
	changed := strings.Replace(create, `"nilai":100000000`, `"nilai":120000000`, 1)
	resp = doJSON(t, a, "PUT", "/invoices/"+id, changed)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 changing nilai of a Kliring invoice, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	// Re-using the same number on a different invoice is a 409, not a 500.
	resp = doJSON(t, a, "POST", "/invoices", create)
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 for duplicate no_invoice, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
}
