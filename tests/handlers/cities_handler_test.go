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

func TestGetAllCities(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/cities", handlers.GetAllCities)

	t.Run("Successfully get all cities", func(t *testing.T) {
		provinceID := uuid.New()
		countryID := uuid.New()
		cityID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "province_id", "created_at", "updated_at"}).
			AddRow(cityID, "Kota Bandung", provinceID, time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "cities"`).WillReturnRows(rows)

		provinceRows := sqlmock.NewRows([]string{"id", "name", "country_id", "created_at", "updated_at"}).
			AddRow(provinceID, "Jawa Barat", countryID, time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "provinces"`).WillReturnRows(provinceRows)

		countryRows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(countryRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "cities"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/cities", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["cities"])
	})

	t.Run("Filter by province_id", func(t *testing.T) {
		mock2 := testutil.SetupMockDB(t)

		provinceID := uuid.New()
		countryID := uuid.New()
		cityID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "province_id", "created_at", "updated_at"}).
			AddRow(cityID, "Kota Bandung", provinceID, time.Now(), time.Now())
		mock2.ExpectQuery(`SELECT \* FROM "cities"`).WillReturnRows(rows)

		provinceRows := sqlmock.NewRows([]string{"id", "name", "country_id", "created_at", "updated_at"}).
			AddRow(provinceID, "Jawa Barat", countryID, time.Now(), time.Now())
		mock2.ExpectQuery(`SELECT \* FROM "provinces"`).WillReturnRows(provinceRows)

		countryRows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())
		mock2.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(countryRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock2.ExpectQuery(`SELECT count\(\*\) FROM "cities"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/cities?province_id="+provinceID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetCity(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/cities/:id", handlers.GetCity)

	t.Run("Successfully get city by ID", func(t *testing.T) {
		provinceID := uuid.New()
		countryID := uuid.New()
		cityID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "province_id", "created_at", "updated_at"}).
			AddRow(cityID, "Kota Bandung", provinceID, time.Now(), time.Now())
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cities" WHERE id = $1`)).
			WithArgs(cityID.String()).
			WillReturnRows(rows)

		provinceRows := sqlmock.NewRows([]string{"id", "name", "country_id", "created_at", "updated_at"}).
			AddRow(provinceID, "Jawa Barat", countryID, time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "provinces"`).WillReturnRows(provinceRows)

		countryRows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(countryRows)

		req := httptest.NewRequest("GET", "/cities/"+cityID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["city"])
	})

	t.Run("City not found", func(t *testing.T) {
		cityID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cities" WHERE id = $1`)).
			WithArgs(cityID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/cities/"+cityID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "City not found", response["error"])
	})
}

func TestCreateCity(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Post("/cities", handlers.CreateCity)

	t.Run("Successfully create city", func(t *testing.T) {
		provinceID := uuid.New()
		cityID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "cities"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "province_id", "created_at", "updated_at"}).
				AddRow(cityID, "Kota Bandung", provinceID, time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{"name": "Kota Bandung", "province_id": provinceID.String()}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/cities", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing name", func(t *testing.T) {
		provinceID := uuid.New()
		reqBody := map[string]any{"province_id": provinceID.String()}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/cities", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Missing province_id", func(t *testing.T) {
		reqBody := map[string]any{"name": "Kota Bandung"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/cities", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

func TestDeleteCity(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/cities/:id", handlers.DeleteCity)

	t.Run("Successfully delete city", func(t *testing.T) {
		cityID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "cities" WHERE id = $1`)).
			WithArgs(cityID.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/cities/"+cityID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "City deleted successfully", response["message"])
	})

	t.Run("City not found", func(t *testing.T) {
		cityID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "cities" WHERE id = $1`)).
			WithArgs(cityID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/cities/"+cityID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
