package handlers_test

import (
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
	"id", "no_sp2k", "judul_pekerjaan", "status_kontrak", "nilai_pekerjaan",
	"tarif_management_fee_persen", "created_at", "updated_at",
}

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
			AddRow(projectID, "SP2K-001", "Pembangunan Gedung", "Active", 1_000_000_000.0, 10.0, time.Now(), time.Now()))

	// Realized totals are aggregated per project in one grouped query, not
	// one query per project.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT project_id, COALESCE(SUM(jumlah_realisasi), 0) AS total FROM "transactions" WHERE project_id IN`)).
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "total"}).
			AddRow(projectID, 850_000_000.0))

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
	assert.Equal(t, 85_000_000.0, s["profit_margin"], "10% of total_realisasi")
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
				AddRow(id, "SP2K-001", "Pembangunan Gedung", "Active", 500_000_000.0, nil, time.Now(), time.Now()))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT project_id, COALESCE(SUM(jumlah_realisasi), 0) AS total FROM "transactions" WHERE project_id IN`)).
			WillReturnRows(sqlmock.NewRows([]string{"project_id", "total"}).AddRow(id, 100_000_000.0))

		resp, _ := app.Test(httptest.NewRequest("GET", "/reports/projects/"+id+"/summary", nil))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		summary := readJSON(t, resp.Body)["project_summary"].(map[string]any)
		assert.Equal(t, 400_000_000.0, summary["remaining_budget"])
		assert.Equal(t, 20.0, summary["absorption_pct"])
		assert.Equal(t, false, summary["absorption_alert"])
		assert.Equal(t, 0.0, summary["profit_margin"], "no fee rate set, so margin is zero")
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

func TestGetManagementFeeReport(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/reports/management-fee", handlers.GetManagementFeeReport)

	t.Run("groups by project and month and computes fee per project", func(t *testing.T) {
		withFee := uuid.New().String()
		withoutFee := uuid.New().String()

		mock.ExpectQuery(regexp.QuoteMeta(`GROUP BY project_id, bulan_realisasi ORDER BY bulan_realisasi, project_id`)).
			WillReturnRows(sqlmock.NewRows([]string{"project_id", "bulan_realisasi", "total"}).
				AddRow(withFee, "2024-01", 2_000_000.0).
				AddRow(withFee, "2024-02", 3_000_000.0).
				AddRow(withoutFee, "2024-01", 1_000_000.0))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id IN`)).
			WillReturnRows(sqlmock.NewRows(projectColumns).
				AddRow(withFee, "SP2K-001", "Gedung A", "Active", 100_000_000.0, 10.0, time.Now(), time.Now()).
				AddRow(withoutFee, "SP2K-002", "Gedung B", "Active", 100_000_000.0, nil, time.Now(), time.Now()))

		resp, _ := app.Test(httptest.NewRequest("GET", "/reports/management-fee", nil))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := readJSON(t, resp.Body)
		rows := body["rows"].([]any)
		assert.Len(t, rows, 3)

		first := rows[0].(map[string]any)
		assert.Equal(t, "2024-01", first["bulan_realisasi"])
		assert.Equal(t, 200_000.0, first["management_fee"], "10% of 2,000,000")

		totals := body["totals"].(map[string]any)
		assert.Equal(t, 6_000_000.0, totals["total_realisasi"])
		assert.Equal(t, 500_000.0, totals["management_fee"], "10% of 5,000,000, the withFee project's full total")
	})

	t.Run("no transactions returns empty rows without querying projects", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`GROUP BY project_id, bulan_realisasi`)).
			WillReturnRows(sqlmock.NewRows([]string{"project_id", "bulan_realisasi", "total"}))

		resp, _ := app.Test(httptest.NewRequest("GET", "/reports/management-fee", nil))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body := readJSON(t, resp.Body)
		assert.Empty(t, body["rows"])
		assert.Equal(t, 0.0, body["totals"].(map[string]any)["management_fee"])
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
	app.Get("/reports/management-fee", middleware.AdminOnly(), handlers.GetManagementFeeReport)

	for _, path := range []string{
		"/reports/projects-summary",
		"/reports/projects/" + uuid.New().String() + "/summary",
		"/reports/management-fee",
	} {
		resp, _ := app.Test(httptest.NewRequest("GET", path, nil))
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode, path)
	}
}
