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

func TestGetAllBudgetItems(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/budget-items", handlers.GetAllBudgetItems)

	costTypeID := uuid.New()
	rows := sqlmock.NewRows([]string{"id", "project_id", "cost_type_id", "jumlah_anggaran", "is_parent", "created_at", "updated_at"}).
		AddRow(uuid.New(), uuid.New(), costTypeID, 1_000_000.0, true, time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM "budget_items"`).WillReturnRows(rows)

	// Preload("CostType") fires its own query since cost_type_id is non-null.
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE "cost_types"."id" = $1`)).
		WithArgs(costTypeID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama_biaya"}).AddRow(costTypeID, "Gaji & Tunjangan"))

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "budget_items"`).WillReturnRows(countRows)

	req := httptest.NewRequest("GET", "/budget-items", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestGetBudgetItem(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/budget-items/:id", handlers.GetBudgetItem)

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "budget_items" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/budget-items/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestCreateBudgetItem(t *testing.T) {
	app := setupApp()
	app.Post("/budget-items", handlers.CreateBudgetItem)

	t.Run("Missing required fields", func(t *testing.T) {
		testutil.SetupMockDB(t)
		reqBody := map[string]any{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/budget-items", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid kategori_anggaran rejected", func(t *testing.T) {
		testutil.SetupMockDB(t)
		reqBody := map[string]any{
			"project_id":        uuid.New().String(),
			"cost_type_id":      uuid.New().String(),
			"kategori_anggaran": "yearly",
			"total_anggaran":    1_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/budget-items", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("lumpsum creates exactly one row, no children", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()
		costTypeID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k", "tanggal_mulai", "nilai_pekerjaan", "jangka_waktu", "status_kontrak"}).
				AddRow(projectID, "SP2K-001", "2024-01-01", 1_000_000_000.0, 3, "Active"))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE id = $1`)).
			WithArgs(costTypeID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nama_biaya"}).AddRow(costTypeID, "Gaji & Tunjangan"))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "budget_items"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{
			"project_id":        projectID.String(),
			"cost_type_id":      costTypeID.String(),
			"kategori_anggaran": "lumpsum",
			"total_anggaran":    1_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/budget-items", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode, string(respBody))
	})

	t.Run("monthly creates a parent plus jangka_waktu children", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()
		costTypeID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k", "tanggal_mulai", "nilai_pekerjaan", "jangka_waktu", "status_kontrak"}).
				AddRow(projectID, "SP2K-001", "2024-01-01", 1_000_000_000.0, 3, "Active"))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE id = $1`)).
			WithArgs(costTypeID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nama_biaya"}).AddRow(costTypeID, "Gaji & Tunjangan"))

		mock.ExpectBegin()
		// parent
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "budget_items"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))
		// 3 children, since the mocked project's jangka_waktu is 3
		for i := 0; i < 3; i++ {
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "budget_items"`)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(uuid.New(), time.Now(), time.Now()))
		}
		mock.ExpectCommit()

		reqBody := map[string]any{
			"project_id":        projectID.String(),
			"cost_type_id":      costTypeID.String(),
			"kategori_anggaran": "monthly",
			"total_anggaran":    3_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/budget-items", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode, string(respBody))
	})
}

func TestUpdateBudgetItem(t *testing.T) {
	app := setupApp()
	app.Put("/budget-items/:id", handlers.UpdateBudgetItem)

	t.Run("non-parent row is a plain update, no splitter involved", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		id := uuid.New()
		projectID := uuid.New()
		parentID := uuid.New()
		costTypeID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "budget_items" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "project_id", "cost_type_id", "jumlah_anggaran", "is_parent", "parent_budget_id", "created_at", "updated_at",
			}).AddRow(id, projectID, costTypeID, 100_000.0, false, parentID, time.Now(), time.Now()))

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "budget_items" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody := map[string]any{
			"project_id":        projectID.String(),
			"cost_type_id":      costTypeID.String(),
			"kategori_anggaran": "monthly",
			"total_anggaran":    100_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/budget-items/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, string(respBody))
	})

	t.Run("parent lumpsum row updates without touching children", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		id := uuid.New()
		projectID := uuid.New()
		costTypeID := uuid.New()
		kategori := "lumpsum"

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "budget_items" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "project_id", "cost_type_id", "kategori_anggaran", "jumlah_anggaran", "is_parent", "created_at", "updated_at",
			}).AddRow(id, projectID, costTypeID, kategori, 1_000_000.0, true, time.Now(), time.Now()))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k", "tanggal_mulai", "nilai_pekerjaan", "jangka_waktu", "status_kontrak"}).
				AddRow(projectID, "SP2K-001", "2024-01-01", 1_000_000_000.0, 3, "Active"))

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "budget_items" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody := map[string]any{
			"project_id":        projectID.String(),
			"cost_type_id":      costTypeID.String(),
			"kategori_anggaran": "lumpsum",
			"total_anggaran":    1_500_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/budget-items/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, string(respBody))
	})

	t.Run("parent monthly row resyncs: update, delete old children, rebuild", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		id := uuid.New()
		projectID := uuid.New()
		costTypeID := uuid.New()
		kategori := "monthly"

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "budget_items" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "project_id", "cost_type_id", "kategori_anggaran", "jumlah_anggaran", "is_parent", "created_at", "updated_at",
			}).AddRow(id, projectID, costTypeID, kategori, 3_000_000.0, true, time.Now(), time.Now()))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k", "tanggal_mulai", "nilai_pekerjaan", "jangka_waktu", "status_kontrak"}).
				AddRow(projectID, "SP2K-001", "2024-01-01", 1_000_000_000.0, 2, "Active"))

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "budget_items" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "budget_items" WHERE parent_budget_id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(0, 2))
		// 2 rebuilt children, since the mocked project's jangka_waktu is 2
		for i := 0; i < 2; i++ {
			mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "budget_items"`)).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(uuid.New(), time.Now(), time.Now()))
		}
		mock.ExpectCommit()

		reqBody := map[string]any{
			"project_id":        projectID.String(),
			"cost_type_id":      costTypeID.String(),
			"kategori_anggaran": "monthly",
			"total_anggaran":    4_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/budget-items/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, string(respBody))
	})

	t.Run("Budget item not found", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "budget_items" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		reqBody := map[string]any{
			"project_id":        uuid.New().String(),
			"cost_type_id":      uuid.New().String(),
			"kategori_anggaran": "lumpsum",
			"total_anggaran":    1_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/budget-items/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteBudgetItem(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/budget-items/:id", handlers.DeleteBudgetItem)

	t.Run("Successfully delete budget item", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "budget_items" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/budget-items/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Budget item not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "budget_items" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/budget-items/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
