package handlers_test

import (
	"database/sql/driver"
	"encoding/json"
	"io"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/middleware"
	"gatra-budget-backend/tests/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// projectColumns are the columns the report queries read from projects.
var projectColumns = []string{
	"id", "no_sp2k", "judul_pekerjaan", "tanggal_mulai", "tanggal_selesai", "nilai_pekerjaan",
	"created_at", "updated_at",
}

// invoiceTotalColumns are the columns of the grouped per-project invoice query.
var invoiceTotalColumns = []string{
	"project_id", "total_invoice", "total_clearing", "total_outstanding", "total_fee",
}

const (
	realisasiQuery = `SELECT project_id, COALESCE(SUM(jumlah_realisasi), 0) AS total FROM "transactions" WHERE project_id IN`
	invoiceQuery   = `FROM "invoices" WHERE project_id IN`
)

func readJSON(t *testing.T, resp io.Reader) map[string]any {
	t.Helper()
	var out map[string]any
	body, err := io.ReadAll(resp)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("failed to decode body %q: %v", string(body), err)
	}
	return out
}

func TestGetProjectsSummaryReport(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/reports/projects-summary", handlers.GetProjectsSummaryReport)

	projectID := uuid.New().String()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" ORDER BY created_at DESC`)).
		WillReturnRows(sqlmock.NewRows(projectColumns).
			AddRow(projectID, "SP2K-001", "Pembangunan Gedung", "2024-01-01", "2024-12-31", 1_000_000_000.0, time.Now(), time.Now()))

	// Realized totals are aggregated per project in one grouped query, not
	// one query per project.
	mock.ExpectQuery(regexp.QuoteMeta(realisasiQuery)).
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "total"}).
			AddRow(projectID, 850_000_000.0))

	// Invoice totals are likewise one grouped query for all projects.
	mock.ExpectQuery(regexp.QuoteMeta(invoiceQuery)).
		WillReturnRows(sqlmock.NewRows(invoiceTotalColumns).
			AddRow(projectID, 300_000_000.0, 200_000_000.0, 100_000_000.0, 30_000_000.0))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "projects"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	resp, _ := app.Test(httptest.NewRequest("GET", "/reports/projects-summary", nil))
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body := readJSON(t, resp.Body)
	summaries, ok := body["projects_summary"].([]any)
	if !assert.True(t, ok, "expected projects_summary array") || !assert.Len(t, summaries, 1) {
		return
	}

	s := summaries[0].(map[string]any)
	assert.Equal(t, 850_000_000.0, s["total_realisasi"])
	assert.Equal(t, 150_000_000.0, s["remaining_budget"], "nilai_pekerjaan - total_realisasi")
	assert.Equal(t, 85.0, s["absorption_pct"])
	assert.Equal(t, true, s["absorption_alert"], "85% is above the 80% threshold")
	assert.Equal(t, 300_000_000.0, s["total_invoice"])
	assert.Equal(t, 200_000_000.0, s["total_clearing"])
	assert.Equal(t, 100_000_000.0, s["total_outstanding"])
	assert.Equal(t, 30_000_000.0, s["profit_margin"], "sum of invoice manajemen_fee")
	assert.Equal(t, s["total_manajemen_fee"], s["profit_margin"])
	assert.Equal(t, "expired", s["contract_status"])
	assert.NotNil(t, body["pagination"])
}

func TestGetProjectSummaryReport(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/reports/projects/:id/summary", handlers.GetProjectSummaryReport)

	t.Run("returns remaining budget and margin for one project", func(t *testing.T) {
		id := uuid.New().String()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(id).
			WillReturnRows(sqlmock.NewRows(projectColumns).
				AddRow(id, "SP2K-001", "Pembangunan Gedung", "2024-01-01", "2024-12-31", 500_000_000.0, time.Now(), time.Now()))

		mock.ExpectQuery(regexp.QuoteMeta(realisasiQuery)).
			WillReturnRows(sqlmock.NewRows([]string{"project_id", "total"}).AddRow(id, 100_000_000.0))

		// No invoices: the project is absent from the grouped result, read as zero.
		mock.ExpectQuery(regexp.QuoteMeta(invoiceQuery)).
			WillReturnRows(sqlmock.NewRows(invoiceTotalColumns))

		resp, _ := app.Test(httptest.NewRequest("GET", "/reports/projects/"+id+"/summary", nil))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		summary := readJSON(t, resp.Body)["project_summary"].(map[string]any)
		assert.Equal(t, 400_000_000.0, summary["remaining_budget"])
		assert.Equal(t, 20.0, summary["absorption_pct"])
		assert.Equal(t, false, summary["absorption_alert"])
		assert.Equal(t, 0.0, summary["profit_margin"], "no invoices, so no fee")
	})

	t.Run("unknown project returns 404", func(t *testing.T) {
		id := uuid.New().String()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(id).
			WillReturnError(gorm.ErrRecordNotFound)

		resp, _ := app.Test(httptest.NewRequest("GET", "/reports/projects/"+id+"/summary", nil))
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestGetMonthlyReport(t *testing.T) {
	t.Run("always returns twelve months and sums each source into its own column", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		app := setupApp()
		app.Get("/reports/monthly", handlers.GetMonthlyReport)

		monthRows := func(rows ...[]driver.Value) *sqlmock.Rows {
			r := sqlmock.NewRows([]string{"month", "total"})
			for _, row := range rows {
				r.AddRow(row...)
			}
			return r
		}

		// Queries run in order: expenses, billed, received, received fee.
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "transactions"`)).
			WillReturnRows(monthRows([]driver.Value{"2026-01", 400_000_000.0}, []driver.Value{"2026-02", 450_000_000.0}))
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "invoices"`)).
			WillReturnRows(monthRows([]driver.Value{"2026-01", 600_000_000.0}))
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "invoices"`)).
			WillReturnRows(monthRows([]driver.Value{"2026-02", 500_000_000.0}))
		mock.ExpectQuery(regexp.QuoteMeta(`FROM "invoices"`)).
			WillReturnRows(monthRows([]driver.Value{"2026-02", 50_000_000.0}))

		resp, _ := app.Test(httptest.NewRequest("GET", "/reports/monthly?year=2026", nil))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := readJSON(t, resp.Body)
		rows := body["rows"].([]any)
		if !assert.Len(t, rows, 12) {
			return
		}

		jan := rows[0].(map[string]any)
		assert.Equal(t, "2026-01", jan["month"])
		assert.Equal(t, 400_000_000.0, jan["pengeluaran"])
		assert.Equal(t, 600_000_000.0, jan["tagihan"])
		assert.Equal(t, 0.0, jan["penerimaan"], "Submit invoices are not received yet")
		assert.Equal(t, -400_000_000.0, jan["selisih"])

		feb := rows[1].(map[string]any)
		assert.Equal(t, 500_000_000.0, feb["penerimaan"])
		assert.Equal(t, 50_000_000.0, feb["manajemen_fee"])
		assert.Equal(t, 50_000_000.0, feb["selisih"])

		assert.Equal(t, 0.0, rows[11].(map[string]any)["pengeluaran"], "empty months are zero")

		totals := body["totals"].(map[string]any)
		assert.Equal(t, 850_000_000.0, totals["pengeluaran"])
		assert.Equal(t, 600_000_000.0, totals["tagihan"])
		assert.Equal(t, 500_000_000.0, totals["penerimaan"])
		assert.Equal(t, 50_000_000.0, totals["manajemen_fee"])
		assert.Equal(t, -350_000_000.0, totals["selisih"])
	})

	t.Run("rejects an invalid year", func(t *testing.T) {
		testutil.SetupMockDB(t)
		app := setupApp()
		app.Get("/reports/monthly", handlers.GetMonthlyReport)

		resp, _ := app.Test(httptest.NewRequest("GET", "/reports/monthly?year=abc", nil))
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// TestReportRoutesAreAdminOnly wires each report handler behind the same
// AdminOnly() gate routes.go applies to the whole /api/reports group, and
// confirms a plain user is rejected before any aggregation runs.
func TestReportRoutesAreAdminOnly(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userRole", "user")
		return c.Next()
	})
	app.Get("/reports/projects-summary", middleware.AdminOnly(), handlers.GetProjectsSummaryReport)
	app.Get("/reports/projects/:id/summary", middleware.AdminOnly(), handlers.GetProjectSummaryReport)
	app.Get("/reports/monthly", middleware.AdminOnly(), handlers.GetMonthlyReport)

	for _, path := range []string{
		"/reports/projects-summary",
		"/reports/projects/" + uuid.New().String() + "/summary",
		"/reports/monthly",
	} {
		resp, _ := app.Test(httptest.NewRequest("GET", path, nil))
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode, path)
	}
}
