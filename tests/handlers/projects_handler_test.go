package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/tests/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGetAllProjects(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/projects", handlers.GetAllProjects)

	rows := sqlmock.NewRows([]string{
		"id", "no_sp2k", "judul_pekerjaan", "tanggal_mulai", "tanggal_selesai", "nilai_pekerjaan",
		"created_at", "updated_at",
	}).AddRow(uuid.New(), "SP2K-001", "Pembangunan Gedung", "2024-01-01", "2024-07-01", 1_000_000_000.0, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM "projects"`).WillReturnRows(rows)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "projects"`).WillReturnRows(countRows)

	req := httptest.NewRequest("GET", "/projects", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]any
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &response)
	assert.NotNil(t, response["projects"])
	assert.NotNil(t, response["pagination"])
}

func TestGetProject(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/projects/:id", handlers.GetProject)

	t.Run("Successfully get project by ID", func(t *testing.T) {
		id := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "no_sp2k", "judul_pekerjaan", "tanggal_mulai", "tanggal_selesai", "nilai_pekerjaan",
			"created_at", "updated_at",
		}).AddRow(id, "SP2K-001", "Pembangunan Gedung", "2024-01-01", "2024-07-01", 1_000_000_000.0, time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/projects/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		project, _ := response["project"].(map[string]any)
		if assert.NotNil(t, project) {
			assert.Equal(t, "expired", project["contract_status"], "end date 2024-07-01 is long past")
			assert.Equal(t, false, project["ending_soon"])
			assert.NotContains(t, project, "status_kontrak")
			assert.NotContains(t, project, "management_fee")
		}
	})

	t.Run("Project not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/projects/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestCreateProject(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Post("/projects", handlers.CreateProject)

	t.Run("Successfully create project, JangkaWaktu computed server-side", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "projects"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(id, time.Now(), time.Now()))
		mock.ExpectCommit()

		// Client sends jangka_waktu, which ProjectRequest has no field for and
		// must therefore be silently ignored — JangkaWaktu is always recomputed
		// from tanggal_mulai/tanggal_selesai.
		reqBody := map[string]any{
			"no_sp2k":         "SP2K-001",
			"judul_pekerjaan": "Pembangunan Gedung",
			"tanggal_mulai":   "2024-01-01",
			"tanggal_selesai": "2024-07-01",
			"nilai_pekerjaan": 1_000_000_000.0,
			"jangka_waktu":    999,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var created map[string]any
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &created)
		assert.Equal(t, float64(6), created["jangka_waktu"], "expected server-computed duration, not the client-supplied 999")
	})

	t.Run("duplicate no_sp2k is rejected with 409", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE LOWER(no_sp2k) = LOWER($1)`)).
			WithArgs("SP2K-001").
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k"}).AddRow(uuid.New(), "SP2K-001"))

		body, _ := json.Marshal(map[string]any{
			"no_sp2k":         "SP2K-001",
			"judul_pekerjaan": "Pembangunan Gedung",
			"tanggal_mulai":   "2024-01-01",
			"tanggal_selesai": "2024-07-01",
			"nilai_pekerjaan": 1_000_000_000.0,
		})
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	})

	t.Run("contract_status filter rejects unknown values", func(t *testing.T) {
		testutil.SetupMockDB(t)
		listApp := setupApp()
		listApp.Get("/projects", handlers.GetAllProjects)

		resp, _ := listApp.Test(httptest.NewRequest("GET", "/projects?contract_status=bogus", nil))
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		reqBody := map[string]any{"judul_pekerjaan": "Pembangunan Gedung"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("missing tanggal_selesai is rejected", func(t *testing.T) {
		reqBody := map[string]any{
			"no_sp2k":         "SP2K-002",
			"judul_pekerjaan": "Pembangunan Gedung",
			"tanggal_mulai":   "2024-01-01",
			"nilai_pekerjaan": 1_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("tanggal_selesai before tanggal_mulai is rejected", func(t *testing.T) {
		reqBody := map[string]any{
			"no_sp2k":         "SP2K-003",
			"judul_pekerjaan": "Pembangunan Gedung",
			"tanggal_mulai":   "2024-06-01",
			"tanggal_selesai": "2024-01-01",
			"nilai_pekerjaan": 1_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateProject(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Put("/projects/:id", handlers.UpdateProject)

	t.Run("Successfully update project", func(t *testing.T) {
		id := uuid.New()
		rows := sqlmock.NewRows([]string{
			"id", "no_sp2k", "judul_pekerjaan", "tanggal_mulai", "tanggal_selesai", "nilai_pekerjaan",
			"created_at", "updated_at",
		}).AddRow(id, "SP2K-001", "Pembangunan Gedung", "2024-01-01", "2024-07-01", 1_000_000_000.0, time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(rows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "projects" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody := map[string]any{
			"no_sp2k":         "SP2K-001",
			"judul_pekerjaan": "Pembangunan Gedung - Updated",
			"tanggal_mulai":   "2024-01-01",
			"tanggal_selesai": "2024-12-31",
			"nilai_pekerjaan": 1_500_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/projects/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Project not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		reqBody := map[string]any{
			"no_sp2k":         "SP2K-001",
			"judul_pekerjaan": "Pembangunan Gedung",
			"tanggal_mulai":   "2024-01-01",
			"tanggal_selesai": "2024-12-31",
			"nilai_pekerjaan": 1_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/projects/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteProject(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/projects/:id", handlers.DeleteProject)

	t.Run("Successfully delete project", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "projects" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/projects/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Project not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "projects" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/projects/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
