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

func TestGetAllAreas(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/areas", handlers.GetAllAreas)

	t.Run("Successfully get areas (empty list)", func(t *testing.T) {
		mock.ExpectQuery(`SELECT \* FROM "areas"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id", "created_at", "updated_at"}))

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT count\(\*\) FROM "areas"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/areas", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["areas"])
		assert.NotNil(t, response["pagination"])
	})

	t.Run("Successfully get single area with preload", func(t *testing.T) {
		mock2 := testutil.SetupMockDB(t)

		cityID := uuid.New()
		areaID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "city_id", "created_at", "updated_at"}).
			AddRow(areaID, "Coblong", cityID, time.Now(), time.Now())
		mock2.ExpectQuery(`SELECT \* FROM "areas"`).WillReturnRows(rows)

		cityRows := sqlmock.NewRows([]string{"id", "name", "province_id", "created_at", "updated_at"}).
			AddRow(cityID, "Kota Bandung", uuid.New(), time.Now(), time.Now())
		mock2.ExpectQuery(`SELECT \* FROM "cities"`).WillReturnRows(cityRows)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock2.ExpectQuery(`SELECT count\(\*\) FROM "areas"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/areas", nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Filter by city_id", func(t *testing.T) {
		mock3 := testutil.SetupMockDB(t)

		cityID := uuid.New()

		mock3.ExpectQuery(`SELECT \* FROM "areas"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id", "created_at", "updated_at"}))

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock3.ExpectQuery(`SELECT count\(\*\) FROM "areas"`).WillReturnRows(countRows)

		req := httptest.NewRequest("GET", "/areas?city_id="+cityID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})
}

func TestGetArea(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/areas/:id", handlers.GetArea)

	t.Run("Successfully get area by ID", func(t *testing.T) {
		cityID := uuid.New()
		areaID := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "city_id", "created_at", "updated_at"}).
			AddRow(areaID, "Coblong", cityID, time.Now(), time.Now())
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "areas" WHERE id = $1`)).
			WithArgs(areaID.String()).
			WillReturnRows(rows)

		cityRows := sqlmock.NewRows([]string{"id", "name", "province_id", "created_at", "updated_at"}).
			AddRow(cityID, "Kota Bandung", uuid.New(), time.Now(), time.Now())
		mock.ExpectQuery(`SELECT \* FROM "cities"`).WillReturnRows(cityRows)

		req := httptest.NewRequest("GET", "/areas/"+areaID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.NotNil(t, response["area"])
	})

	t.Run("Area not found", func(t *testing.T) {
		areaID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "areas" WHERE id = $1`)).
			WithArgs(areaID.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/areas/"+areaID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Area not found", response["error"])
	})
}

func TestCreateArea(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Post("/areas", handlers.CreateArea)

	t.Run("Successfully create area", func(t *testing.T) {
		cityID := uuid.New()
		areaID := uuid.New()

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "areas"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "city_id", "created_at", "updated_at"}).
				AddRow(areaID, "Coblong", cityID, time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{"name": "Coblong", "city_id": cityID.String()}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/areas", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	})

	t.Run("Missing name", func(t *testing.T) {
		cityID := uuid.New()
		reqBody := map[string]any{"city_id": cityID.String()}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/areas", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]any
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
		assert.Equal(t, "Name is required", response["error"])
	})

	t.Run("Missing city_id", func(t *testing.T) {
		reqBody := map[string]any{"name": "Coblong"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/areas", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

		var response map[string]any
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &response)
		assert.Equal(t, "city_id is required", response["error"])
	})
}

func TestDeleteArea(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/areas/:id", handlers.DeleteArea)

	t.Run("Successfully delete area", func(t *testing.T) {
		areaID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "areas" WHERE id = $1`)).
			WithArgs(areaID.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/areas/"+areaID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var response map[string]any
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &response)
		assert.Equal(t, "Area deleted successfully", response["message"])
	})

	t.Run("Area not found", func(t *testing.T) {
		areaID := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "areas" WHERE id = $1`)).
			WithArgs(areaID.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/areas/"+areaID.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
