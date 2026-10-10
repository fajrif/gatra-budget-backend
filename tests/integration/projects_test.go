package integration

import (
	"testing"
	"time"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"
)

// TestProjectClientSetNullOnDelete proves the real ON DELETE SET NULL on
// projects.client_id: deleting a client must not cascade-delete its projects,
// only detach them.
func TestProjectClientSetNullOnDelete(t *testing.T) {
	tx := withTx(t)

	client := models.Client{Name: "PT Mitra Jaya"}
	if err := tx.Create(&client).Error; err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	project := models.Project{
		NoSP2K:         "SP2K-001",
		JudulPekerjaan: "Pembangunan Gedung",
		TanggalMulai:   models.Date{Time: mustParseDate(t, "2024-01-01")},
		NilaiPekerjaan: 1_000_000_000,
		ClientID:       &client.ID,
		StatusKontrak:  "Active",
	}
	if err := tx.Create(&project).Error; err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	if err := tx.Delete(&models.Client{}, "id = ?", client.ID).Error; err != nil {
		t.Fatalf("failed to delete client: %v", err)
	}

	var reloaded models.Project
	if err := tx.Where("id = ?", project.ID).First(&reloaded).Error; err != nil {
		t.Fatalf("expected project to survive client deletion: %v", err)
	}
	if reloaded.ClientID != nil {
		t.Fatalf("expected client_id to be SET NULL, got %v", *reloaded.ClientID)
	}
}

// TestCreateProjectComputesJangkaWaktu exercises CreateProject end-to-end
// against the real database, confirming the duration helper and the handler
// wiring agree on a real round trip, not just a mocked one.
func TestCreateProjectComputesJangkaWaktu(t *testing.T) {
	withTx(t)

	a := app()
	a.Post("/projects", handlers.CreateProject)

	resp := doJSON(t, a, "POST", "/projects", `{
		"no_sp2k": "SP2K-002",
		"judul_pekerjaan": "Renovasi Kantor",
		"tanggal_mulai": "2024-01-01",
		"tanggal_selesai": "2024-07-01",
		"nilai_pekerjaan": 500000000
	}`)

	if resp.StatusCode != 201 {
		t.Fatalf("expected 201 creating project, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	created := readBody(t, resp)

	jangkaWaktu, ok := created["jangka_waktu"].(float64)
	if !ok {
		t.Fatalf("expected jangka_waktu in response, got %v", created["jangka_waktu"])
	}
	if jangkaWaktu != 6 {
		t.Fatalf("expected jangka_waktu 6, got %v", jangkaWaktu)
	}
}

func mustParseDate(t *testing.T, s string) (parsed time.Time) {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("failed to parse date %q: %v", s, err)
	}
	return parsed
}
