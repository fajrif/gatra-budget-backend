package integration

import (
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"
)

func TestCostOperationMinimumAmountCheckConstraint(t *testing.T) {
	tx := withTx(t)
	costType := models.OperationalCostType{NamaBiaya: "ATK"}
	if err := tx.Create(&costType).Error; err != nil {
		t.Fatal(err)
	}

	err := tx.Create(&models.CostOperation{
		OperationalCostTypeID: costType.ID,
		TanggalTransaksi:      models.Date{Time: mustParseDate(t, "2024-02-01")},
		Jumlah:                50_000,
	}).Error
	if err == nil {
		t.Fatal("expected the CHECK constraint to reject jumlah <= 100000")
	}
}

func TestOperationalCostTypeNameIsUniqueCaseInsensitive(t *testing.T) {
	tx := withTx(t)
	if err := tx.Create(&models.OperationalCostType{NamaBiaya: "Konsumsi"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`INSERT INTO operational_cost_types (nama_biaya) VALUES ('KONSUMSI')`).Error; err == nil {
		t.Fatal("expected the unique index to reject a case-variant duplicate name")
	}
}

func TestCostOperationCRUDAndTypeDeleteGuard(t *testing.T) {
	tx := withTx(t)
	costType := models.OperationalCostType{NamaBiaya: "Listrik & Air"}
	if err := tx.Create(&costType).Error; err != nil {
		t.Fatal(err)
	}

	a := app()
	a.Post("/cost-operations", handlers.CreateCostOperation)
	a.Delete("/operational-cost-types/:id", handlers.DeleteOperationalCostType)

	resp := doJSON(t, a, "POST", "/cost-operations", `{
		"operational_cost_type_id": "`+costType.ID.String()+`",
		"tanggal_transaksi": "2024-02-15",
		"jumlah": 2000000
	}`)
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	if got := readBody(t, resp)["bulan_realisasi"]; got != "2024-02" {
		t.Fatalf("expected bulan_realisasi 2024-02, got %v", got)
	}

	resp = doJSON(t, a, "DELETE", "/operational-cost-types/"+costType.ID.String(), "")
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 deleting a type still in use, got %d", resp.StatusCode)
	}
}

// TestMonthlyReportIncludesOperationalExpenses checks the combined figures and
// that a project filter leaves operational expenses out.
func TestMonthlyReportIncludesOperationalExpenses(t *testing.T) {
	tx := withTx(t)
	project, costType := newTestProject(t, tx)
	opType := models.OperationalCostType{NamaBiaya: "Sewa Kantor"}
	if err := tx.Create(&opType).Error; err != nil {
		t.Fatal(err)
	}

	date := models.Date{Time: mustParseDate(t, "2031-05-10")}
	bulan := "2031-05"
	if err := tx.Create(&models.Transaction{ProjectID: project.ID, CostTypeID: costType.ID, TanggalTransaksi: date, BulanRealisasi: &bulan, JumlahRealisasi: 1_000_000}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&models.CostOperation{OperationalCostTypeID: opType.ID, TanggalTransaksi: date, BulanRealisasi: &bulan, Jumlah: 250_000}).Error; err != nil {
		t.Fatal(err)
	}

	a := app()
	a.Get("/reports/monthly", handlers.GetMonthlyReport)
	a.Get("/reports/operational-summary", handlers.GetOperationalSummaryReport)

	totals := readBody(t, doJSON(t, a, "GET", "/reports/monthly?year=2031", ""))["totals"].(map[string]any)
	if totals["pengeluaran"] != 1_000_000.0 || totals["operasional"] != 250_000.0 || totals["total_pengeluaran"] != 1_250_000.0 || totals["selisih"] != -1_250_000.0 {
		t.Fatalf("unexpected combined totals: %v", totals)
	}

	filtered := readBody(t, doJSON(t, a, "GET", "/reports/monthly?year=2031&project_id="+project.ID.String(), ""))["totals"].(map[string]any)
	if filtered["operasional"] != 0.0 || filtered["total_pengeluaran"] != 1_000_000.0 {
		t.Fatalf("project filter must exclude operational expenses: %v", filtered)
	}

	summary := readBody(t, doJSON(t, a, "GET", "/reports/operational-summary?year=2031", ""))
	if summary["total"] != 250_000.0 {
		t.Fatalf("unexpected operational summary: %v", summary)
	}
}
