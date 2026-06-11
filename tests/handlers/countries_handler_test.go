package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"boilerplates-backend/handlers"
	"boilerplates-backend/tests/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupApp() *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", "550e8400-e29b-41d4-a716-446655440000")
		c.Locals("userEmail", "user@example.com")
		c.Locals("userRole", "user")
		return c.Next()
	})
	return app
}

func TestGetAllCountries(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Get("/countries", handlers.GetAllCountries)

	t.Run("Successfully get all countries", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(uuid.New(), "Indonesia", "ID", time.Now(), time.Now()).
			AddRow(uuid.New(), "United States", "US", time.Now(), time.Now())

		mock.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(rows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "countries"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/countries", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)

		assert.NotNil(t, response["countries"])
		assert.NotNil(t, response["pagination"])
		countries := response["countries"].([]interface{})
		assert.Equal(t, 2, len(countries))
	})

	t.Run("Get all without pagination using all=true", func(t *testing.T) {
		db2, mock2, err := testutil.SetupMockDB()
		assert.NoError(t, err)
		defer testutil.CloseMockDB(db2)

		rows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(uuid.New(), "Indonesia", "ID", time.Now(), time.Now())

		mock2.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(rows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock2.ExpectQuery(`SELECT count\(\*\) FROM "countries"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/countries?all=true", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetCountry(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Get("/countries/:id", handlers.GetCountry)

	t.Run("Successfully get country by ID", func(t *testing.T) {
		countryID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "countries" WHERE id = $1`)).
			WithArgs(countryID.String()).
			WillReturnRows(rows)

		req := httptest.NewRequest("GET", "/countries/"+countryID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["country"])
	})

	t.Run("Country not found", func(t *testing.T) {
		countryID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "countries" WHERE id = $1`)).
			WithArgs(countryID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/countries/"+countryID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Country not found", response["error"])
	})
}

func TestCreateCountry(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Post("/countries", handlers.CreateCountry)

	t.Run("Successfully create country", func(t *testing.T) {
		countryID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "countries"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
				AddRow(countryID, "Malaysia", "MY", time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]interface{}{"name": "Malaysia", "code": "MY"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/countries", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing required fields", func(t *testing.T) {
		reqBody := map[string]interface{}{"name": "Malaysia"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/countries", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
		assert.Equal(t, "Name and code are required", response["error"])
	})

	t.Run("Invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/countries", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateCountry(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Put("/countries/:id", handlers.UpdateCountry)

	t.Run("Successfully update country", func(t *testing.T) {
		countryID := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "countries" WHERE id = $1`)).
			WithArgs(countryID.String()).
			WillReturnRows(rows)

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE "countries" SET`)).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		reqBody := map[string]interface{}{"name": "Republic of Indonesia", "code": "ID"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/countries/"+countryID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Country not found", func(t *testing.T) {
		countryID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "countries" WHERE id = $1`)).
			WithArgs(countryID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		reqBody := map[string]interface{}{"name": "Indonesia", "code": "ID"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/countries/"+countryID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteCountry(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Delete("/countries/:id", handlers.DeleteCountry)

	t.Run("Successfully delete country", func(t *testing.T) {
		countryID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "countries" WHERE id = $1`)).
			WithArgs(countryID.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/countries/"+countryID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Country deleted successfully", response["message"])
	})

	t.Run("Country not found", func(t *testing.T) {
		countryID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "countries" WHERE id = $1`)).
			WithArgs(countryID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/countries/"+countryID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
