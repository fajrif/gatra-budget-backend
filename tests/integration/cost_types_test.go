package integration

import (
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"
)

// TestCostTypeCRUDRoundTrip exercises create + read against the real
// database. CostType has no unique constraint of its own (unlike Client and
// ContractType), so what a mock cannot verify here is the real generated SQL
// round-tripping through Postgres.
func TestCostTypeCRUDRoundTrip(t *testing.T) {
	tx := withTx(t)

	a := app()
	a.Post("/cost-types", handlers.CreateCostType)
	a.Get("/cost-types/:id", handlers.GetCostType)

	resp := doJSON(t, a, "POST", "/cost-types", `{"nama_biaya": "Gaji & Tunjangan", "kode": "SAL"}`)
	if resp.StatusCode != 201 {
		t.Fatalf("expected 201 creating cost type, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	created := readBody(t, resp)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("expected an id in the create response, got %v", created)
	}

	var reloaded models.CostType
	if err := tx.Where("id = ?", id).First(&reloaded).Error; err != nil {
		t.Fatalf("failed to reload cost type: %v", err)
	}
	if reloaded.NamaBiaya != "Gaji & Tunjangan" {
		t.Fatalf("expected nama_biaya to persist, got %q", reloaded.NamaBiaya)
	}
	if reloaded.Kode == nil || *reloaded.Kode != "SAL" {
		t.Fatalf("expected kode to persist as SAL, got %v", reloaded.Kode)
	}
}
