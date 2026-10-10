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

func TestGetAllContractTypes(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/contract-types", handlers.GetAllContractTypes)

	rows := sqlmock.NewRows([]string{"id", "name", "code", "description", "created_at", "updated_at"}).
		AddRow(uuid.New(), "PO", "PO", "Purchase order based contract", time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM "contract_types"`).WillReturnRows(rows)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "contract_types"`).WillReturnRows(countRows)

	req := httptest.NewRequest("GET", "/contract-types", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]any
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &response)
	assert.NotNil(t, response["contract_types"])
	assert.NotNil(t, response["pagination"])
}

func TestGetContractType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/contract-types/:id", handlers.GetContractType)

	t.Run("Successfully get contract type by ID", func(t *testing.T) {
		id := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "code", "description", "created_at", "updated_at"}).
			AddRow(id, "PO", "PO", "Purchase order based contract", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "contract_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/contract-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["contract_type"])
	})

	t.Run("Contract type not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "contract_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/contract-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestCreateContractType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Post("/contract-types", handlers.CreateContractType)

	t.Run("Successfully create contract type", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "contract_types"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(id, time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{"name": "Non-PO"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/contract-types", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing required field", func(t *testing.T) {
		reqBody := map[string]any{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/contract-types", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateContractType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Put("/contract-types/:id", handlers.UpdateContractType)

	t.Run("Successfully update contract type", func(t *testing.T) {
		id := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "code", "description", "created_at", "updated_at"}).
			AddRow(id, "PO", "PO", "Purchase order based contract", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "contract_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnRows(rows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "contract_types" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody := map[string]any{"name": "PO - Updated"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/contract-types/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Contract type not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "contract_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		reqBody := map[string]any{"name": "PO"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/contract-types/"+id.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteContractType(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/contract-types/:id", handlers.DeleteContractType)

	t.Run("Successfully delete contract type", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "contract_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/contract-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Contract type not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "contract_types" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/contract-types/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestContractTypeWriteRoutesAreAdminOnly(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userRole", "user")
		return c.Next()
	})
	app.Post("/contract-types", middleware.AdminOnly(), handlers.CreateContractType)

	reqBody := map[string]any{"name": "Non-PO"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/contract-types", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
