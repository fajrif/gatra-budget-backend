package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gatra-budget-backend/models"
	"gatra-budget-backend/routes"

	"github.com/gofiber/fiber/v2"
)

// reportPaths are every route under /api/reports, all of which must be
// admin-only.
var reportPaths = []string{
	"/api/reports/projects-summary",
	"/api/reports/management-fee",
}

// authedGet performs a GET with a Bearer token against the real middleware.
func authedGet(t *testing.T, a *fiber.App, path, token string) *http.Response {
	t.Helper()

	req := httptest.NewRequest("GET", path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.Test(req)
	if err != nil {
		t.Fatalf("GET %s failed: %v", path, err)
	}
	return resp
}

// TestReportRoutesAdminOnlyOverRealJWT drives the full routes.Setup wiring with
// real logins and real signed tokens: a plain user must get 403 on every
// /api/reports route, an admin must get 200. This is the server-side gate that
// keeps remaining budget and margin from leaking to non-admins.
func TestReportRoutesAdminOnlyOverRealJWT(t *testing.T) {
	tx := withTx(t)

	newTestUser(t, tx, "reports-admin@gatrahitawasana.com", "admin")
	newTestUser(t, tx, "reports-user@gatrahitawasana.com", "user")

	a := app()
	routes.Setup(a)

	userToken := loginAndGetToken(t, a, "reports-user@gatrahitawasana.com")
	adminToken := loginAndGetToken(t, a, "reports-admin@gatrahitawasana.com")

	for _, path := range reportPaths {
		t.Run("user is forbidden: "+path, func(t *testing.T) {
			resp := authedGet(t, a, path, userToken)
			if resp.StatusCode != http.StatusForbidden {
				t.Fatalf("expected 403 for non-admin, got %d: %s", resp.StatusCode, readRawBody(t, resp))
			}
		})

		t.Run("admin is allowed: "+path, func(t *testing.T) {
			resp := authedGet(t, a, path, adminToken)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("expected 200 for admin, got %d: %s", resp.StatusCode, readRawBody(t, resp))
			}
		})
	}

	t.Run("project summary is forbidden for non-admin", func(t *testing.T) {
		project, _ := newBudgetItemTestProject(t, tx)
		resp := authedGet(t, a, "/api/reports/projects/"+project.ID.String()+"/summary", userToken)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for non-admin, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}
	})

	t.Run("plain project payload never carries remaining budget or margin", func(t *testing.T) {
		project, _ := newBudgetItemTestProject(t, tx)
		resp := authedGet(t, a, "/api/projects/"+project.ID.String(), userToken)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for GET project, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		raw := readRawBody(t, resp)
		for _, forbidden := range []string{"remaining_budget", "profit_margin", "absorption_pct"} {
			if strings.Contains(raw, forbidden) {
				t.Fatalf("plain project payload leaked %q: %s", forbidden, raw)
			}
		}
	})
}

// TestReportTotalsAgainstRealDB checks the arithmetic against real Postgres
// aggregation: SUM by project, the 80% absorption threshold, and the
// management fee computed from the full project total.
func TestReportTotalsAgainstRealDB(t *testing.T) {
	tx := withTx(t)
	newTestUser(t, tx, "reports-totals@gatrahitawasana.com", "admin")

	project, costType := newBudgetItemTestProject(t, tx)
	if err := tx.Model(&project).Updates(map[string]any{
		"nilai_pekerjaan":             1_000_000_000,
		"tarif_management_fee_persen": 10,
	}).Error; err != nil {
		t.Fatalf("failed to set contract value and fee rate: %v", err)
	}

	jan, feb := "2024-01", "2024-02"
	for _, row := range []struct {
		month  string
		amount float64
	}{
		{jan, 400_000_000},
		{feb, 450_000_000},
	} {
		transaction := models.Transaction{
			ProjectID:        project.ID,
			CostTypeID:       costType.ID,
			TanggalTransaksi: models.Date{Time: mustParseDate(t, row.month+"-15")},
			BulanRealisasi:   &row.month,
			JumlahRealisasi:  row.amount,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			t.Fatalf("failed to create transaction: %v", err)
		}
	}

	a := app()
	routes.Setup(a)
	token := loginAndGetToken(t, a, "reports-totals@gatrahitawasana.com")

	t.Run("project summary: remaining budget, absorption alert, margin", func(t *testing.T) {
		resp := authedGet(t, a, "/api/reports/projects/"+project.ID.String()+"/summary", token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		summary := readBody(t, resp)["project_summary"].(map[string]any)
		assertFloat(t, summary, "total_realisasi", 850_000_000)
		assertFloat(t, summary, "remaining_budget", 150_000_000)
		assertFloat(t, summary, "absorption_pct", 85)
		if summary["absorption_alert"] != true {
			t.Fatalf("expected absorption_alert at 85%%, got %v", summary["absorption_alert"])
		}
		assertFloat(t, summary, "profit_margin", 85_000_000)
	})

	t.Run("management fee report: per-month rows and project-level total", func(t *testing.T) {
		resp := authedGet(t, a, "/api/reports/management-fee", token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		body := readBody(t, resp)
		rows := body["rows"].([]any)

		var ours []map[string]any
		for _, r := range rows {
			row := r.(map[string]any)
			if row["project_id"] == project.ID.String() {
				ours = append(ours, row)
			}
		}
		if len(ours) != 2 {
			t.Fatalf("expected 2 monthly rows for this project, got %d", len(ours))
		}
		assertFloat(t, ours[0], "management_fee", 40_000_000) // 10% of Jan 400M
		assertFloat(t, ours[1], "management_fee", 45_000_000) // 10% of Feb 450M
	})
}

// loginAndGetToken logs in through the real login route for a user created in
// the test transaction, and returns the signed token.
func loginAndGetToken(t *testing.T, a *fiber.App, email string) string {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"email": email, "password": "Integration123!"})
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.Test(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login as %s failed with %d: %s", email, resp.StatusCode, readRawBody(t, resp))
	}

	token, _ := readBody(t, resp)["token"].(string)
	if token == "" {
		t.Fatalf("no token in login response for %s", email)
	}
	return token
}

// assertFloat fails unless the named JSON field equals want.
func assertFloat(t *testing.T, m map[string]any, key string, want float64) {
	t.Helper()

	got, ok := m[key].(float64)
	if !ok {
		t.Fatalf("expected %s to be a number, got %v", key, m[key])
	}
	if got != want {
		t.Fatalf("%s: expected %v, got %v", key, want, got)
	}
}
