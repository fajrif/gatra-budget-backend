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

func TestGetAllProvinces(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Get("/provinces", handlers.GetAllProvinces)

	t.Run("Successfully get all provinces", func(t *testing.T) {
		countryID := uuid.New()
		provinceID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "country_id", "created_at", "updated_at"}).
			AddRow(provinceID, "DKI Jakarta", countryID, time.Now(), time.Now())

		mock.ExpectQuery(`SELECT \* FROM "provinces"`).WillReturnRows(rows)

		countryRows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(countryRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "provinces"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/provinces", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["provinces"])
	})

	t.Run("Filter by country_id", func(t *testing.T) {
		db2, mock2, err := testutil.SetupMockDB()
		assert.NoError(t, err)
		defer testutil.CloseMockDB(db2)

		countryID := uuid.New()
		provinceID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "country_id", "created_at", "updated_at"}).
			AddRow(provinceID, "DKI Jakarta", countryID, time.Now(), time.Now())

		mock2.ExpectQuery(`SELECT \* FROM "provinces"`).WillReturnRows(rows)

		countryRows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())
		mock2.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(countryRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock2.ExpectQuery(`SELECT count\(\*\) FROM "provinces"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/provinces?country_id="+countryID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetProvince(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Get("/provinces/:id", handlers.GetProvince)

	t.Run("Successfully get province by ID", func(t *testing.T) {
		countryID := uuid.New()
		provinceID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "country_id", "created_at", "updated_at"}).
			AddRow(provinceID, "DKI Jakarta", countryID, time.Now(), time.Now())

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "provinces" WHERE id = $1`)).
			WithArgs(provinceID.String()).
			WillReturnRows(rows)

		countryRows := sqlmock.NewRows([]string{"id", "name", "code", "created_at", "updated_at"}).
			AddRow(countryID, "Indonesia", "ID", time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "countries"`).WillReturnRows(countryRows)

		req := httptest.NewRequest("GET", "/provinces/"+provinceID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["province"])
	})

	t.Run("Province not found", func(t *testing.T) {
		provinceID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "provinces" WHERE id = $1`)).
			WithArgs(provinceID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/provinces/"+provinceID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Province not found", response["error"])
	})
}

func TestCreateProvince(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Post("/provinces", handlers.CreateProvince)

	t.Run("Successfully create province", func(t *testing.T) {
		countryID := uuid.New()
		provinceID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "provinces"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "country_id", "created_at", "updated_at"}).
				AddRow(provinceID, "DKI Jakarta", countryID, time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]interface{}{"name": "DKI Jakarta", "country_id": countryID.String()}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/provinces", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing name", func(t *testing.T) {
		countryID := uuid.New()
		reqBody := map[string]interface{}{"country_id": countryID.String()}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/provinces", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
		assert.Equal(t, "Name is required", response["error"])
	})

	t.Run("Missing country_id", func(t *testing.T) {
		reqBody := map[string]interface{}{"name": "DKI Jakarta"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/provinces", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
		assert.Equal(t, "country_id is required", response["error"])
	})
}

func TestDeleteProvince(t *testing.T) {
	db, mock, err := testutil.SetupMockDB()
	assert.NoError(t, err)
	defer testutil.CloseMockDB(db)

	app := setupApp()
	app.Delete("/provinces/:id", handlers.DeleteProvince)

	t.Run("Successfully delete province", func(t *testing.T) {
		provinceID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "provinces" WHERE id = $1`)).
			WithArgs(provinceID.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/provinces/"+provinceID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Province deleted successfully", response["message"])
	})

	t.Run("Province not found", func(t *testing.T) {
		provinceID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "provinces" WHERE id = $1`)).
			WithArgs(provinceID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/provinces/"+provinceID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
