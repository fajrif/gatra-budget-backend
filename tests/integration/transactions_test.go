package integration

import (
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"
)

func TestTransactionMinimumAmountCheckConstraint(t *testing.T) {
	tx := withTx(t)
	project, costType := newTestProject(t, tx)

	transaction := models.Transaction{
		ProjectID:        project.ID,
		CostTypeID:       costType.ID,
		TanggalTransaksi: models.Date{Time: mustParseDate(t, "2024-02-01")},
		JumlahRealisasi:  50_000, // below the 100,000 CHECK constraint
	}

	err := tx.Create(&transaction).Error
	if err == nil {
		t.Fatal("expected the database CHECK constraint to reject jumlah_realisasi <= 100000")
	}
}

// TestTransactionCreateComputesBulanRealisasiAgainstRealDB exercises
// CreateTransaction end-to-end against the real database.
func TestTransactionCreateComputesBulanRealisasiAgainstRealDB(t *testing.T) {
	tx := withTx(t)
	project, costType := newTestProject(t, tx)

	a := app()
	a.Post("/transactions", handlers.CreateTransaction)

	resp := doJSON(t, a, "POST", "/transactions", `{
		"project_id": "`+project.ID.String()+`",
		"cost_type_id": "`+costType.ID.String()+`",
		"tanggal_transaksi": "2024-02-15",
		"jumlah_realisasi": 2000000
	}`)

	if resp.StatusCode != 201 {
		t.Fatalf("expected 201 creating transaction, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	created := readBody(t, resp)
	if created["bulan_realisasi"] != "2024-02" {
		t.Fatalf("expected bulan_realisasi '2024-02', got %v", created["bulan_realisasi"])
	}
}

func TestTransactionProjectCascadeDeletesTransactions(t *testing.T) {
	tx := withTx(t)
	project, costType := newTestProject(t, tx)

	transaction := models.Transaction{
		ProjectID:        project.ID,
		CostTypeID:       costType.ID,
		TanggalTransaksi: models.Date{Time: mustParseDate(t, "2024-02-01")},
		JumlahRealisasi:  500_000,
	}
	if err := tx.Create(&transaction).Error; err != nil {
		t.Fatalf("failed to create transaction: %v", err)
	}

	if err := tx.Delete(&models.Project{}, "id = ?", project.ID).Error; err != nil {
		t.Fatalf("failed to delete project: %v", err)
	}

	var count int64
	if err := tx.Model(&models.Transaction{}).Where("project_id = ?", project.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count transactions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected ON DELETE CASCADE from project to remove its transactions, found %d", count)
	}
}
