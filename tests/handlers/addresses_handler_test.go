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

const testUserID = "550e8400-e29b-41d4-a716-446655440000"

func setupUserApp() *fiber.App {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("userID", testUserID)
		c.Locals("userEmail", "user@example.com")
		c.Locals("userRole", "user")
		return c.Next()
	})
	return app
}

func addrCols() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "user_id", "name", "address",
		"country_id", "province_id", "city_id", "area_id", "zipcode",
		"created_at", "updated_at",
	})
}

func TestGetMyAddresses(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupUserApp()
	app.Get("/me/addresses", handlers.GetMyAddresses)

	t.Run("Returns 200 with empty list", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM "addresses"`).WillReturnRows(addrCols())
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "addresses"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/me/addresses", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["addresses"])
		assert.NotNil(t, response["pagination"])
	})

	t.Run("Returns 200 with pagination params", func(t *testing.T) {
		mock2 := testutil.SetupMockDB(t)

		mock2.ExpectQuery(`SELECT \* FROM "addresses"`).WillReturnRows(addrCols())
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(5)
		mock2.ExpectQuery(`SELECT count\(\*\) FROM "addresses"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/me/addresses?page=1&limit=10", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		pagination := response["pagination"].(map[string]any)
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(10), pagination["limit"])
	})
}

func TestGetMyAddress(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupUserApp()
	app.Get("/me/addresses/:id", handlers.GetMyAddress)

	t.Run("Address not found", func(t *testing.T) {
		addrID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "addresses" WHERE id = $1 AND user_id = $2`)).
			WithArgs(addrID.String(), testUserID).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/me/addresses/"+addrID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Address not found", response["error"])
	})
}

func TestCreateMyAddress(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupUserApp()
	app.Post("/me/addresses", handlers.CreateMyAddress)

	t.Run("Successfully create address", func(t *testing.T) {
		addrID := uuid.New()
		countryID := uuid.New()
		provinceID := uuid.New()
		cityID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "addresses"`)).
			WillReturnRows(addrCols().
				AddRow(addrID, testUserID, "Home", "Jl. Sudirman No. 1",
					countryID, provinceID, cityID, nil, "12345",
					time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{
			"name":        "Home",
			"address":     "Jl. Sudirman No. 1",
			"country_id":  countryID.String(),
			"province_id": provinceID.String(),
			"city_id":     cityID.String(),
			"zipcode":     "12345",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/me/addresses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing required fields - no address text", func(t *testing.T) {
		countryID := uuid.New()
		provinceID := uuid.New()
		cityID := uuid.New()
		reqBody := map[string]any{
			"name":        "Home",
			"country_id":  countryID.String(),
			"province_id": provinceID.String(),
			"city_id":     cityID.String(),
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/me/addresses", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Invalid request body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/me/addresses", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestUpdateMyAddress(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupUserApp()
	app.Put("/me/addresses/:id", handlers.UpdateMyAddress)

	t.Run("Address not found", func(t *testing.T) {
		addrID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "addresses" WHERE id = $1 AND user_id = $2`)).
			WithArgs(addrID.String(), testUserID).
			WillReturnError(gorm.ErrRecordNotFound)

		reqBody := map[string]any{"name": "Office"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("PUT", "/me/addresses/"+addrID.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteMyAddress(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupUserApp()
	app.Delete("/me/addresses/:id", handlers.DeleteMyAddress)

	t.Run("Successfully delete address", func(t *testing.T) {
		addrID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "addresses" WHERE id = $1 AND user_id = $2`)).
			WithArgs(addrID.String(), testUserID).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/me/addresses/"+addrID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Address deleted successfully", response["message"])
	})

	t.Run("Address not found", func(t *testing.T) {
		addrID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "addresses" WHERE id = $1 AND user_id = $2`)).
			WithArgs(addrID.String(), testUserID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/me/addresses/"+addrID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestGetUserAddresses(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupAdminApp()
	app.Get("/users/:user_id/addresses", handlers.GetUserAddresses)

	t.Run("Admin can get any user's addresses", func(t *testing.T) {
		targetUserID := uuid.New()

		mock.ExpectQuery(`SELECT \* FROM "addresses"`).WillReturnRows(addrCols())
		countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "addresses"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/users/"+targetUserID.String()+"/addresses", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Invalid user_id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/invalid-uuid/addresses", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Invalid user ID", response["error"])
	})
}
