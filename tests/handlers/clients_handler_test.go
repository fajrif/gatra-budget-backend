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

func TestGetAllClients(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/clients", handlers.GetAllClients)

	rows := sqlmock.NewRows([]string{"id", "name", "contact_name", "phone", "address", "created_at", "updated_at"}).
		AddRow(uuid.New(), "PT Mitra Jaya", "Budi", "081234567890", "Jakarta", time.Now(), time.Now())

	mock.ExpectQuery(`SELECT \* FROM "clients"`).WillReturnRows(rows)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery(`SELECT count\(\*\) FROM "clients"`).WillReturnRows(countRows)

	req := httptest.NewRequest("GET", "/clients", nil)
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]any
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &response)
	assert.NotNil(t, response["clients"])
	assert.NotNil(t, response["pagination"])
}

func TestGetClient(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/clients/:id", handlers.GetClient)

	t.Run("Successfully get client by ID", func(t *testing.T) {
		clientID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "contact_name", "phone", "address", "created_at", "updated_at"}).
			AddRow(clientID, "PT Mitra Jaya", "Budi", "081234567890", "Jakarta", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "clients" WHERE id = $1`)).
			WithArgs(clientID.String()).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/clients/"+clientID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["client"])
	})

	t.Run("Client not found", func(t *testing.T) {
		clientID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "clients" WHERE id = $1`)).
			WithArgs(clientID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/clients/"+clientID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestCreateClient(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Post("/clients", handlers.CreateClient)

	t.Run("Successfully create client", func(t *testing.T) {
		clientID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "clients"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(clientID, time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{"name": "PT Mitra Jaya"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/clients", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing required field", func(t *testing.T) {
		reqBody := map[string]any{}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/clients", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateClient(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Put("/clients/:id", handlers.UpdateClient)

	t.Run("Successfully update client", func(t *testing.T) {
		clientID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "contact_name", "phone", "address", "created_at", "updated_at"}).
			AddRow(clientID, "PT Mitra Jaya", "Budi", "081234567890", "Jakarta", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "clients" WHERE id = $1`)).
			WithArgs(clientID.String()).
			WillReturnRows(rows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "clients" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody := map[string]any{"name": "PT Mitra Jaya Sejahtera"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/clients/"+clientID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Client not found", func(t *testing.T) {
		clientID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "clients" WHERE id = $1`)).
			WithArgs(clientID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		reqBody := map[string]any{"name": "PT Mitra Jaya"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/clients/"+clientID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteClient(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/clients/:id", handlers.DeleteClient)

	t.Run("Successfully delete client", func(t *testing.T) {
		clientID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "clients" WHERE id = $1`)).
			WithArgs(clientID.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/clients/"+clientID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Client not found", func(t *testing.T) {
		clientID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "clients" WHERE id = $1`)).
			WithArgs(clientID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/clients/"+clientID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

// TestClientWriteRoutesAreAdminOnly exercises the inline middleware.AdminOnly()
// wiring from routes.go directly (not just middleware.AdminOnly() in isolation,
// which tests/middleware already covers) to confirm a plain "user" role is
// rejected before CreateClient ever runs.
func TestClientWriteRoutesAreAdminOnly(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userRole", "user")
		return c.Next()
	})
	app.Post("/clients", middleware.AdminOnly(), handlers.CreateClient)

	reqBody := map[string]any{"name": "PT Mitra Jaya"}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/clients", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
