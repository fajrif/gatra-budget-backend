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
	"gatra-budget-backend/middleware"
	"gatra-budget-backend/tests/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGetAllCostTypes(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/cost-types", handlers.GetAllCostTypes)

	rows := sqlmock.NewRows([]string{"id", "nama_biaya", "kode", "deskripsi", "created_at", "updated_at"}).
		AddRow(uuid.New(), "Gaji & Tunjangan", "SAL", "Payroll expenses", time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM "cost_types"`).WillReturnRows(rows)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "cost_types"`).WillReturnRows(countRows)

	req := httptest.NewRequest("GET", "/cost-types", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]any
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &response)
	assert.NotNil(t, response["cost_types"])
	assert.NotNil(t, response["pagination"])
}

func TestGetCostType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/cost-types/:id", handlers.GetCostType)

	t.Run("Successfully get cost type by ID", func(t *testing.T) {
		id := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "nama_biaya", "kode", "deskripsi", "created_at", "updated_at"}).
			AddRow(id, "Gaji & Tunjangan", "SAL", "Payroll expenses", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/cost-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["cost_type"])
	})

	t.Run("Cost type not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/cost-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestCreateCostType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Post("/cost-types", handlers.CreateCostType)

	t.Run("Successfully create cost type", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "cost_types"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(id, time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{"nama_biaya": "Gaji & Tunjangan"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/cost-types", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing required field", func(t *testing.T) {
		reqBody := map[string]any{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/cost-types", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateCostType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Put("/cost-types/:id", handlers.UpdateCostType)

	t.Run("Successfully update cost type", func(t *testing.T) {
		id := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "nama_biaya", "kode", "deskripsi", "created_at", "updated_at"}).
			AddRow(id, "Gaji & Tunjangan", "SAL", "Payroll expenses", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(rows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "cost_types" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody := map[string]any{"nama_biaya": "Gaji & Tunjangan - Updated"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/cost-types/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Cost type not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		reqBody := map[string]any{"nama_biaya": "Gaji & Tunjangan"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/cost-types/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteCostType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/cost-types/:id", handlers.DeleteCostType)

	t.Run("Successfully delete cost type", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "cost_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/cost-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Cost type not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "cost_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/cost-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestCostTypeWriteRoutesAreAdminOnly(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userRole", "user")
		return c.Next()
	})
	app.Post("/cost-types", middleware.AdminOnly(), handlers.CreateCostType)

	reqBody := map[string]any{"nama_biaya": "Gaji & Tunjangan"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/cost-types", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
