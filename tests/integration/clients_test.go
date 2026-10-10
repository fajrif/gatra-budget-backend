package integration

import (
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"
)

// TestDuplicateClientNameRejected proves the real UNIQUE index on
// clients.name is enforced. The handler's CRUD path is faked in the unit
// tests, so this is the only place the database constraint itself is
// verified.
func TestDuplicateClientNameRejected(t *testing.T) {
	tx := withTx(t)

	client := models.Client{Name: "PT Mitra Jaya"}
	if err := tx.Create(&client).Error; err != nil {
		t.Fatalf("failed to create initial client: %v", err)
	}

	a := app()
	a.Post("/clients", handlers.CreateClient)

	resp := doJSON(t, a, "POST", "/clients", `{"name": "PT Mitra Jaya"}`)

	// A rejected INSERT poisons the rest of this test's transaction (Postgres
	// requires a ROLLBACK/SAVEPOINT before any further statement), so this
	// must be the last thing the test does with tx — see TestDuplicateEmailRejected.
	if resp.StatusCode < 400 {
		t.Fatalf("expected an error status for a duplicate client name, got %d: %s",
			resp.StatusCode, readRawBody(t, resp))
	}
}
