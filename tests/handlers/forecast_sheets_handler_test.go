package handlers_test

import (
	"bytes"
	"net/http/httptest"
	"regexp"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/tests/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateForecastSheetValidation(t *testing.T) {
	app := setupApp()
	app.Post("/forecast-sheets", handlers.CreateForecastSheet)

	for name, body := range map[string]string{
		"bad json":       `{`,
		"missing nama":   `{"bulan":"2026-10"}`,
		"blank nama":     `{"nama":"  ","bulan":"2026-10"}`,
		"missing bulan":  `{"nama":"Perkiraan Oct, 2026"}`,
		"bad bulan":      `{"nama":"X","bulan":"2026-13"}`,
		"blank item":     `{"nama":"X","bulan":"2026-10","expenses":[{"item":" ","tanggal":"2026-10-05","jumlah":1000}]}`,
		"zero jumlah":    `{"nama":"X","bulan":"2026-10","expenses":[{"item":"Listrik","tanggal":"2026-10-05","jumlah":0}]}`,
		"bad tanggal":    `{"nama":"X","bulan":"2026-10","expenses":[{"item":"Listrik","tanggal":"05/10/2026","jumlah":1000}]}`,
		"missing tgl":    `{"nama":"X","bulan":"2026-10","expenses":[{"item":"Listrik","jumlah":1000}]}`,
		"other month":    `{"nama":"X","bulan":"2026-10","expenses":[{"item":"Listrik","tanggal":"2026-11-01","jumlah":1000}]}`,
		"other year":     `{"nama":"X","bulan":"2026-10","expenses":[{"item":"Listrik","tanggal":"2025-10-05","jumlah":1000}]}`,
		"second row bad": `{"nama":"X","bulan":"2026-10","expenses":[{"item":"A","tanggal":"2026-10-05","jumlah":1000},{"item":"B","tanggal":"2026-09-30","jumlah":1000}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			testutil.SetupMockDB(t) // no statement is expected: validation fails first
			req := httptest.NewRequest("POST", "/forecast-sheets", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			resp, _ := app.Test(req)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestCreateForecastSheetDuplicateName(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Post("/forecast-sheets", handlers.CreateForecastSheet)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "forecast_sheets" WHERE LOWER(nama) = LOWER($1)`)).
		WithArgs("Perkiraan Oct, 2026").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req := httptest.NewRequest("POST", "/forecast-sheets", bytes.NewBufferString(`{"nama":"Perkiraan Oct, 2026","bulan":"2026-10"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestCreateForecastSheetUnknownInvoice(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Post("/forecast-sheets", handlers.CreateForecastSheet)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "forecast_sheets" WHERE LOWER(nama) = LOWER($1)`)).
		WithArgs("Baru").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, status FROM "invoices"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status"}))

	body := `{"nama":"Baru","bulan":"2026-10","invoice_ids":["` + uuid.New().String() + `"]}`
	req := httptest.NewRequest("POST", "/forecast-sheets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateForecastSheetInvalidInvoiceID(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Post("/forecast-sheets", handlers.CreateForecastSheet)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT count(*) FROM "forecast_sheets" WHERE LOWER(nama) = LOWER($1)`)).
		WithArgs("Baru").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	req := httptest.NewRequest("POST", "/forecast-sheets", bytes.NewBufferString(`{"nama":"Baru","bulan":"2026-10","invoice_ids":["bukan-uuid"]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetForecastSheetNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Get("/forecast-sheets/:id", handlers.GetForecastSheet)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "forecast_sheets" WHERE id = $1`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	resp, _ := app.Test(httptest.NewRequest("GET", "/forecast-sheets/"+uuid.New().String(), nil))
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateForecastSheetNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Put("/forecast-sheets/:id", handlers.UpdateForecastSheet)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "forecast_sheets" WHERE id = $1`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	req := httptest.NewRequest("PUT", "/forecast-sheets/"+uuid.New().String(), bytes.NewBufferString(`{"nama":"X","bulan":"2026-10"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDeleteForecastSheetNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Delete("/forecast-sheets/:id", handlers.DeleteForecastSheet)

	id := uuid.New().String()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "forecast_sheets" WHERE id = $1`)).
		WithArgs(id).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	resp, _ := app.Test(httptest.NewRequest("DELETE", "/forecast-sheets/"+id, nil))
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
