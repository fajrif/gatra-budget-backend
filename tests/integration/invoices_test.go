package integration

import (
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
		{"Clearing without tgl_bayar", func(i *models.Invoice) { i.Status = models.InvoiceStatusClearing }},
		{"Submit with tgl_bayar", func(i *models.Invoice) { i.TglBayar = &bayar }},
		{"fee above nilai", func(i *models.Invoice) { i.ManajemenFee = i.Nilai + 1 }},
		{"unknown status", func(i *models.Invoice) { i.Status = "Paid" }},
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

// TestInvoiceCreateAndClearAgainstRealDB runs the real handler through a
// Submit -> Clearing transition.
func TestInvoiceCreateAndClearAgainstRealDB(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)

	a := app()
	a.Post("/invoices", handlers.CreateInvoice)
	a.Put("/invoices/:id", handlers.UpdateInvoice)

	payload := func(status, bayar string) string {
		return `{"no_invoice":"INV-H-1","project_id":"` + project.ID.String() + `","nama_invoice":"Termin 1",
			"tgl_tagihan":"2026-10-01","status":"` + status + `"` + bayar + `,"nilai":100000000,"manajemen_fee":5000000}`
	}

	resp := doJSON(t, a, "POST", "/invoices", payload("Submit", ""))
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	id := readBody(t, resp)["id"].(string)

	resp = doJSON(t, a, "PUT", "/invoices/"+id, payload("Clearing", `,"tgl_bayar":"2026-10-25"`))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 clearing the invoice, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	updated := readBody(t, resp)
	if updated["status"] != "Clearing" || updated["tgl_bayar"] != "2026-10-25" {
		t.Fatalf("expected Clearing with tgl_bayar, got %v / %v", updated["status"], updated["tgl_bayar"])
	}

	// Re-using the same number on a different invoice is a 409, not a 500.
	resp = doJSON(t, a, "POST", "/invoices", payload("Submit", ""))
	if resp.StatusCode != 409 {
		t.Fatalf("expected 409 for duplicate no_invoice, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
}
