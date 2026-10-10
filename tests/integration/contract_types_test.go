package integration

import (
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"
)

// TestDuplicateContractTypeNameRejected proves the real UNIQUE index on
// contract_types.name is enforced.
func TestDuplicateContractTypeNameRejected(t *testing.T) {
	tx := withTx(t)

	contractType := models.ContractType{Name: "PO"}
	if err := tx.Create(&contractType).Error; err != nil {
		t.Fatalf("failed to create initial contract type: %v", err)
	}

	a := app()
	a.Post("/contract-types", handlers.CreateContractType)

	resp := doJSON(t, a, "POST", "/contract-types", `{"name": "PO"}`)

	// A rejected INSERT poisons the rest of this test's transaction (Postgres
	// requires a ROLLBACK/SAVEPOINT before any further statement), so this
	// must be the last thing the test does with tx — see TestDuplicateEmailRejected.
	if resp.StatusCode < 400 {
		t.Fatalf("expected an error status for a duplicate contract type name, got %d: %s",
			resp.StatusCode, readRawBody(t, resp))
	}
}
