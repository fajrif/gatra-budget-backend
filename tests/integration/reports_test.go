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
	"/api/reports/monthly",
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
		project, _ := newTestProject(t, tx)
		resp := authedGet(t, a, "/api/reports/projects/"+project.ID.String()+"/summary", userToken)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for non-admin, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}
	})

	t.Run("plain project payload never carries remaining budget or margin", func(t *testing.T) {
		project, _ := newTestProject(t, tx)
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
// aggregation: SUM by project, the 80% absorption threshold, profit margin as
// the sum of invoice fees, and the monthly report's per-source grouping.
func TestReportTotalsAgainstRealDB(t *testing.T) {
	tx := withTx(t)
	newTestUser(t, tx, "reports-totals@gatrahitawasana.com", "admin")

	project, costType := newTestProject(t, tx)
	// newTestProject sets nilai_pekerjaan to 1,000,000,000.

	for _, row := range []struct {
		month  string
		amount float64
	}{
		{"2026-01", 400_000_000},
		{"2026-02", 450_000_000},
	} {
		month := row.month
		transaction := models.Transaction{
			ProjectID:        project.ID,
			CostTypeID:       costType.ID,
			TanggalTransaksi: models.Date{Time: mustParseDate(t, month+"-15")},
			BulanRealisasi:   &month,
			JumlahRealisasi:  row.amount,
		}
		if err := tx.Create(&transaction).Error; err != nil {
			t.Fatalf("failed to create transaction: %v", err)
		}
	}

	bayar := models.Date{Time: mustParseDate(t, "2026-02-20")}
	for _, inv := range []models.Invoice{
		{NoInvoice: "INV-R-1", NamaInvoice: "Termin 1", TglTagihan: models.Date{Time: mustParseDate(t, "2026-01-10")},
			TglBayar: &bayar, Status: models.InvoiceStatusClearing, Nilai: 500_000_000, ManajemenFee: 40_000_000},
		{NoInvoice: "INV-R-2", NamaInvoice: "Termin 2", TglTagihan: models.Date{Time: mustParseDate(t, "2026-02-10")},
			Status: models.InvoiceStatusSubmit, Nilai: 300_000_000, ManajemenFee: 20_000_000},
	} {
		inv.ProjectID = project.ID
		if err := tx.Create(&inv).Error; err != nil {
			t.Fatalf("failed to create invoice: %v", err)
		}
	}

	a := app()
	routes.Setup(a)
	token := loginAndGetToken(t, a, "reports-totals@gatrahitawasana.com")

	t.Run("project summary: remaining budget, absorption alert, margin from invoice fees", func(t *testing.T) {
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
		assertFloat(t, summary, "total_invoice", 800_000_000)
		assertFloat(t, summary, "total_clearing", 500_000_000)
		assertFloat(t, summary, "total_outstanding", 300_000_000)
		assertFloat(t, summary, "profit_margin", 60_000_000)
	})

	t.Run("monthly report groups each source by its own month", func(t *testing.T) {
		resp := authedGet(t, a, "/api/reports/monthly?year=2026&project_id="+project.ID.String(), token)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readRawBody(t, resp))
		}

		body := readBody(t, resp)
		rows := body["rows"].([]any)
		if len(rows) != 12 {
			t.Fatalf("expected 12 monthly rows, got %d", len(rows))
		}

		jan := rows[0].(map[string]any)
		assertFloat(t, jan, "pengeluaran", 400_000_000)
		assertFloat(t, jan, "tagihan", 500_000_000)
		assertFloat(t, jan, "penerimaan", 0) // billed in January, paid in February

		feb := rows[1].(map[string]any)
		assertFloat(t, feb, "pengeluaran", 450_000_000)
		assertFloat(t, feb, "tagihan", 300_000_000)
		assertFloat(t, feb, "penerimaan", 500_000_000)
		assertFloat(t, feb, "manajemen_fee", 40_000_000) // only the Clearing invoice's fee
		assertFloat(t, feb, "selisih", 50_000_000)

		totals := body["totals"].(map[string]any)
		assertFloat(t, totals, "pengeluaran", 850_000_000)
		assertFloat(t, totals, "penerimaan", 500_000_000)
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
