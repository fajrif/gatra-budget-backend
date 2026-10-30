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

func invoiceBody(overrides map[string]any) *bytes.Reader {
	body := map[string]any{
		"no_invoice":    "INV-001",
		"project_id":    uuid.New().String(),
		"nama_invoice":  "Tagihan Oktober",
		"tgl_tagihan":   "2026-10-01",
		"status":        "Submit",
		"nilai":         100_000_000.0,
		"manajemen_fee": 5_000_000.0,
	}
	for k, v := range overrides {
		if v == nil {
			delete(body, k)
		} else {
			body[k] = v
		}
	}
	b, _ := json.Marshal(body)
	return bytes.NewReader(b)
}

func postInvoice(app *fiber.App, body *bytes.Reader) int {
	req := httptest.NewRequest("POST", "/invoices", body)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	return resp.StatusCode
}

func TestCreateInvoiceValidation(t *testing.T) {
	app := setupApp()
	app.Post("/invoices", handlers.CreateInvoice)

	tests := []struct {
		name      string
		overrides map[string]any
	}{
		{"missing no_invoice", map[string]any{"no_invoice": nil}},
		{"missing project_id", map[string]any{"project_id": nil}},
		{"missing nama_invoice", map[string]any{"nama_invoice": nil}},
		{"missing tgl_tagihan", map[string]any{"tgl_tagihan": nil}},
		{"zero nilai", map[string]any{"nilai": 0.0}},
		{"unknown status", map[string]any{"status": "Paid"}},
		{"Clearing without tgl_bayar", map[string]any{"status": "Clearing"}},
		{"Submit with tgl_bayar", map[string]any{"tgl_bayar": "2026-10-20"}},
		{"tgl_bayar on the billing date", map[string]any{"status": "Clearing", "tgl_bayar": "2026-10-01"}},
		{"tgl_bayar before tgl_tagihan", map[string]any{"status": "Clearing", "tgl_bayar": "2026-09-30"}},
		{"negative fee", map[string]any{"manajemen_fee": -1.0}},
		{"fee above nilai", map[string]any{"manajemen_fee": 200_000_000.0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.SetupMockDB(t)
			assert.Equal(t, fiber.StatusBadRequest, postInvoice(app, invoiceBody(tt.overrides)))
		})
	}
}

func TestCreateInvoice(t *testing.T) {
	app := setupApp()
	app.Post("/invoices", handlers.CreateInvoice)

	t.Run("creates a Clearing invoice with its payment date", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k"}).AddRow(projectID, "SP2K-001"))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE no_invoice = $1`)).
			WithArgs("INV-001").
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "invoices"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))
		mock.ExpectCommit()

		req := httptest.NewRequest("POST", "/invoices", invoiceBody(map[string]any{
			"project_id": projectID.String(),
			"status":     "Clearing",
			"tgl_bayar":  "2026-10-20",
		}))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode, string(respBody))

		var created map[string]any
		json.Unmarshal(respBody, &created)
		assert.Equal(t, "Clearing", created["status"])
		assert.Equal(t, "2026-10-20", created["tgl_bayar"])
		assert.Equal(t, 5_000_000.0, created["manajemen_fee"])
	})

	t.Run("manajemen_fee defaults to zero", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k"}).AddRow(projectID, "SP2K-001"))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE no_invoice = $1`)).
			WithArgs("INV-001").
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "invoices"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))
		mock.ExpectCommit()

		req := httptest.NewRequest("POST", "/invoices", invoiceBody(map[string]any{
			"project_id":    projectID.String(),
			"manajemen_fee": nil,
		}))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode, string(respBody))

		var created map[string]any
		json.Unmarshal(respBody, &created)
		assert.Equal(t, 0.0, created["manajemen_fee"])
	})

	t.Run("unknown project returns 404", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WillReturnError(gorm.ErrRecordNotFound)

		assert.Equal(t, fiber.StatusNotFound, postInvoice(app, invoiceBody(nil)))
	})

	t.Run("duplicate no_invoice returns 409", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k"}).AddRow(projectID, "SP2K-001"))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE no_invoice = $1`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_invoice"}).AddRow(uuid.New(), "INV-001"))

		assert.Equal(t, fiber.StatusConflict, postInvoice(app, invoiceBody(map[string]any{"project_id": projectID.String()})))
	})
}

func TestGetAllInvoices(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/invoices", handlers.GetAllInvoices)

	mock.ExpectQuery(`SELECT \* FROM "invoices"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "no_invoice", "tgl_tagihan", "status", "nilai"}).
			AddRow(uuid.New(), "INV-001", "2026-10-01", "Submit", 100_000_000.0))
	mock.ExpectQuery(`SELECT count\(\*\) FROM "invoices"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	resp, _ := app.Test(httptest.NewRequest("GET", "/invoices", nil))
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var response map[string]any
	body, _ := io.ReadAll(resp.Body)
	json.Unmarshal(body, &response)
	assert.Len(t, response["invoices"], 1)
	assert.NotNil(t, response["pagination"])
}

func TestGetInvoiceNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/invoices/:id", handlers.GetInvoice)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE id = $1`)).
		WithArgs(id.String()).
		WillReturnError(gorm.ErrRecordNotFound)

	resp, _ := app.Test(httptest.NewRequest("GET", "/invoices/"+id.String(), nil))
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDeleteInvoice(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/invoices/:id", handlers.DeleteInvoice)

	t.Run("deletes", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "invoices" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		resp, _ := app.Test(httptest.NewRequest("DELETE", "/invoices/"+id.String(), nil))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "invoices" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		resp, _ := app.Test(httptest.NewRequest("DELETE", "/invoices/"+id.String(), nil))
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
