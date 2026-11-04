package integration

import (
	"net/http"
	"strings"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

func groupsApp() *fiber.App {
	a := app()
	a.Get("/project-groups", handlers.GetAllProjectGroups)
	a.Post("/project-groups", handlers.CreateProjectGroup)
	a.Put("/project-groups/:id", handlers.UpdateProjectGroup)
	a.Delete("/project-groups/:id", handlers.DeleteProjectGroup)
	a.Get("/projects", handlers.GetAllProjects)
	a.Post("/invoices", handlers.CreateInvoice)
	a.Put("/invoices/:id", handlers.UpdateInvoice)
	a.Get("/invoices", handlers.GetAllInvoices)
	return a
}

func TestProjectGroupTotalsAndDeleteGuard(t *testing.T) {
	tx := withTx(t)
	a := groupsApp()

	resp := doJSON(t, a, "POST", "/project-groups", `{"nama":"Pengadaan TAD Sumbar","deskripsi":"Kontrak dipecah"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	groupID := readBody(t, resp)["id"].(string)

	// Names are unique regardless of case.
	resp = doJSON(t, a, "POST", "/project-groups", `{"nama":"  pengadaan tad sumbar "}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate name, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	resp = doJSON(t, a, "POST", "/project-groups", `{"nama":"   "}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank name, got %d", resp.StatusCode)
	}

	// Two child projects: 1B (from newTestProject) + 500M.
	p1, _ := newTestProject(t, tx)
	p2, _ := newTestProject(t, tx)
	p2.NilaiPekerjaan = 500_000_000
	tx.Save(&p2)
	tx.Model(&models.Project{}).Where("id IN ?", []any{p1.ID, p2.ID}).Update("project_group_id", groupID)

	resp = doJSON(t, a, "GET", "/project-groups?search=tad+sumbar", "")
	groups := readBody(t, resp)["project_groups"].([]any)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	g := groups[0].(map[string]any)
	assertFloat(t, g, "project_count", 2)
	assertFloat(t, g, "total_nilai", 1_500_000_000)

	// A group that still has child projects cannot be deleted.
	resp = doJSON(t, a, "DELETE", "/project-groups/"+groupID, "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 deleting a group with projects, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	var still int64
	tx.Model(&models.ProjectGroup{}).Where("id = ?", groupID).Count(&still)
	if still != 1 {
		t.Fatal("the group must survive a rejected delete")
	}

	// The projects list can be filtered by group and carries the group.
	resp = doJSON(t, a, "GET", "/projects?project_group_id="+groupID, "")
	projects := readBody(t, resp)["projects"].([]any)
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects in the group, got %d", len(projects))
	}
	if projects[0].(map[string]any)["project_group"] == nil {
		t.Fatal("expected the project_group to be preloaded")
	}

	// Once emptied, the group can go.
	tx.Model(&models.Project{}).Where("project_group_id = ?", groupID).Update("project_group_id", nil)
	resp = doJSON(t, a, "DELETE", "/project-groups/"+groupID, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting an empty group, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
}

func TestProjectGroupForeignKeyBlocksDelete(t *testing.T) {
	tx := withTx(t)
	group := models.ProjectGroup{Nama: "FK Guard"}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	project, _ := newTestProject(t, tx)
	tx.Model(&project).Update("project_group_id", group.ID)

	tx.Exec("SAVEPOINT fk")
	if err := tx.Delete(&models.ProjectGroup{}, "id = ?", group.ID).Error; err == nil {
		t.Fatal("expected the foreign key to reject deleting a group with projects")
	}
	tx.Exec("ROLLBACK TO SAVEPOINT fk")
}

func TestInvoiceBillingPeriodAgainstRealDB(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx) // contract 2026-01 .. 2026-12
	a := groupsApp()

	payload := func(no, metode, periods string) string {
		return `{"no_invoice":"` + no + `","project_id":"` + project.ID.String() + `","nama_invoice":"Tagihan",
			"tgl_tagihan":"2026-03-10","status":"Submit","metode_pembayaran":"` + metode + `",
			"periode_tagihan":` + periods + `,"nilai":100000000}`
	}

	t.Run("Kontrak bills one month that can differ from tgl_tagihan", func(t *testing.T) {
		resp := doJSON(t, a, "POST", "/invoices", payload("INV-P-1", "Kontrak", `["2026-02"]`))
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}
		body := readBody(t, resp)
		if body["metode_pembayaran"] != "Kontrak" || strings.Join(toStrings(body["periode_tagihan"]), ",") != "2026-02" {
			t.Fatalf("unexpected invoice: %v", body)
		}

		// The array round-trips through Postgres.
		var saved models.Invoice
		tx.Where("no_invoice = ?", "INV-P-1").First(&saved)
		if len(saved.PeriodeTagihan) != 1 || saved.PeriodeTagihan[0] != "2026-02" {
			t.Fatalf("expected [2026-02] from the database, got %v", saved.PeriodeTagihan)
		}
	})

	t.Run("SPR bills several months, stored sorted", func(t *testing.T) {
		resp := doJSON(t, a, "POST", "/invoices", payload("INV-P-2", "SPR", `["2026-04","2026-03"]`))
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}
		var saved models.Invoice
		tx.Where("no_invoice = ?", "INV-P-2").First(&saved)
		if strings.Join(saved.PeriodeTagihan, ",") != "2026-03,2026-04" {
			t.Fatalf("expected sorted [2026-03 2026-04], got %v", saved.PeriodeTagihan)
		}
	})

	t.Run("rejects the rules the form enforces", func(t *testing.T) {
		cases := map[string]string{
			"Kontrak with two months":      payload("INV-P-3", "Kontrak", `["2026-02","2026-03"]`),
			"month outside the contract":   payload("INV-P-4", "SPR", `["2025-12"]`),
			"month after the contract end": payload("INV-P-5", "SPR", `["2027-01"]`),
			"duplicate month":              payload("INV-P-6", "SPR", `["2026-03","2026-03"]`),
			"empty periods":                payload("INV-P-7", "SPR", `[]`),
			"unknown method":               payload("INV-P-8", "Cash", `["2026-03"]`),
		}
		for name, body := range cases {
			resp := doJSON(t, a, "POST", "/invoices", body)
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("%s: expected 400, got %d: %s", name, resp.StatusCode, readRawBody(t, resp))
			}
		}
	})

	t.Run("list filters by periode and metode_pembayaran", func(t *testing.T) {
		count := func(query string) int {
			resp := doJSON(t, a, "GET", "/invoices?project_id="+project.ID.String()+query, "")
			return len(readBody(t, resp)["invoices"].([]any))
		}
		if got := count("&periode=2026-03"); got != 1 {
			t.Errorf("periode=2026-03: expected 1 (the SPR invoice), got %d", got)
		}
		if got := count("&periode=2026-02"); got != 1 {
			t.Errorf("periode=2026-02: expected 1 (the Kontrak invoice), got %d", got)
		}
		if got := count("&metode_pembayaran=SPR"); got != 1 {
			t.Errorf("metode_pembayaran=SPR: expected 1, got %d", got)
		}
	})
}

func TestImportInvoiceBillingPeriodAndProjectGroup(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	group := models.ProjectGroup{Nama: "Grup Import"}
	if err := tx.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	a := importApp()

	row := func(n int, no string) map[string]any {
		return map[string]any{
			"row": n, "no_sp2k": project.NoSP2K, "no_invoice": no, "nama_invoice": "Termin",
			"tgl_tagihan": "2026-10-01", "status": "submit", "nilai": "100000000",
			"metode_pembayaran": "spr", "periode_tagihan": "2026-03, 2026-04",
		}
	}

	t.Run("reports period errors per row and saves nothing", func(t *testing.T) {
		kontrakTwo := row(3, "INV-IP-2")
		kontrakTwo["metode_pembayaran"] = "Kontrak"
		outside := row(4, "INV-IP-3")
		outside["periode_tagihan"] = "2027-05"
		missing := row(5, "INV-IP-4")
		missing["periode_tagihan"] = ""
		badMethod := row(6, "INV-IP-5")
		badMethod["metode_pembayaran"] = "Tunai"

		before := countRows(t, &models.Invoice{})
		status, body := postImport(t, a, "/invoices/import", []map[string]any{row(2, "INV-IP-1"), kontrakTwo, outside, missing, badMethod})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %v", status, body)
		}
		errs := importErrorsOf(t, body)
		for _, key := range []string{"3:periode_tagihan", "4:periode_tagihan", "5:periode_tagihan", "6:metode_pembayaran"} {
			if _, ok := errs[key]; !ok {
				t.Errorf("expected an error for %s, got %v", key, errs)
			}
		}
		if after := countRows(t, &models.Invoice{}); after != before {
			t.Fatalf("a failed import must save nothing: %d -> %d", before, after)
		}
	})

	t.Run("accepts a multi-month SPR cell", func(t *testing.T) {
		status, body := postImport(t, a, "/invoices/import", []map[string]any{row(2, "INV-IP-OK")})
		if status != http.StatusOK || body["created"] != float64(1) {
			t.Fatalf("expected 1 created, got %d: %v", status, body)
		}
		var saved models.Invoice
		tx.Where("no_invoice = ?", "INV-IP-OK").First(&saved)
		if saved.MetodePembayaran != "SPR" || strings.Join(saved.PeriodeTagihan, ",") != "2026-03,2026-04" {
			t.Fatalf("unexpected invoice: %+v", saved)
		}
	})

	t.Run("project import resolves the group by name", func(t *testing.T) {
		good := validProjectRow(2, "SP-GRP-1")
		good["project_group"] = " grup import "
		unknown := validProjectRow(3, "SP-GRP-2")
		unknown["project_group"] = "Tidak Ada"

		status, body := postImport(t, a, "/projects/import", []map[string]any{good, unknown})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %v", status, body)
		}
		if _, ok := importErrorsOf(t, body)["3:project_group"]; !ok {
			t.Fatalf("expected an error for the unknown group, got %v", importErrorsOf(t, body))
		}

		status, body = postImport(t, a, "/projects/import", []map[string]any{good})
		if status != http.StatusOK {
			t.Fatalf("expected 200, got %d: %v", status, body)
		}
		var saved models.Project
		tx.Where("no_sp2k = ?", "SP-GRP-1").First(&saved)
		if saved.ProjectGroupID == nil || *saved.ProjectGroupID != group.ID {
			t.Fatalf("expected the project in group %s, got %v", group.ID, saved.ProjectGroupID)
		}
	})
}

func toStrings(v any) []string {
	list, _ := v.([]any)
	out := make([]string, len(list))
	for i, x := range list {
		out[i], _ = x.(string)
	}
	return out
}
