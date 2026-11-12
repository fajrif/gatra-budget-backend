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
		"no_invoice":        "INV-001",
		"project_id":        uuid.New().String(),
		"nama_invoice":      "Tagihan Oktober",
		"tgl_tagihan":       "2026-10-01",
		"status":            "Submit",
		"metode_pembayaran": "Kontrak",
		"periode_tagihan":   []string{"2026-10"},
		"nilai":             100_000_000.0,
		"manajemen_fee":     5_000_000.0,
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

// projectRows is a project whose contract runs 2026-09-01 to 2026-12-31.
func projectRows(id uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "no_sp2k", "tanggal_mulai", "tanggal_selesai"}).
		AddRow(id, "SP2K-001", "2026-09-01", "2026-12-31")
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
		{"Kliring without tgl_bayar", map[string]any{"status": "Kliring"}},
		{"Submit with tgl_bayar", map[string]any{"tgl_bayar": "2026-10-20"}},
		{"tgl_bayar on the billing date", map[string]any{"status": "Kliring", "tgl_bayar": "2026-10-01"}},
		{"tgl_bayar before tgl_tagihan", map[string]any{"status": "Kliring", "tgl_bayar": "2026-09-30"}},
		{"missing metode_pembayaran", map[string]any{"metode_pembayaran": nil}},
		{"unknown metode_pembayaran", map[string]any{"metode_pembayaran": "Cash"}},
		{"missing periode_tagihan", map[string]any{"periode_tagihan": nil}},
		{"empty periode_tagihan", map[string]any{"periode_tagihan": []string{}}},
		{"bad periode format", map[string]any{"periode_tagihan": []string{"2026-13"}}},
		{"periode without year", map[string]any{"periode_tagihan": []string{"Okt"}}},
		{"duplicate periode", map[string]any{"metode_pembayaran": "SPR", "periode_tagihan": []string{"2026-10", "2026-10"}}},
		{"Kontrak with two months", map[string]any{"periode_tagihan": []string{"2026-09", "2026-10"}}},
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

	t.Run("creates a Kliring invoice with its payment date", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(projectRows(projectID))
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
			"status":     "Kliring",
			"tgl_bayar":  "2026-10-20",
		}))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode, string(respBody))

		var created map[string]any
		json.Unmarshal(respBody, &created)
		assert.Equal(t, "Kliring", created["status"])
		assert.Equal(t, "2026-10-20", created["tgl_bayar"])
		assert.Equal(t, 5_000_000.0, created["manajemen_fee"])
	})

	t.Run("manajemen_fee defaults to zero", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(projectRows(projectID))
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

	t.Run("SPR invoice keeps several billing months, sorted", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(projectRows(projectID))
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE no_invoice = $1`)).
			WithArgs("INV-001").
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "invoices"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))
		mock.ExpectCommit()

		req := httptest.NewRequest("POST", "/invoices", invoiceBody(map[string]any{
			"project_id":        projectID.String(),
			"metode_pembayaran": "SPR",
			"periode_tagihan":   []string{"2026-11", "2026-10"},
		}))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode, string(respBody))

		var created map[string]any
		json.Unmarshal(respBody, &created)
		assert.Equal(t, "SPR", created["metode_pembayaran"])
		assert.Equal(t, []any{"2026-10", "2026-11"}, created["periode_tagihan"])
	})

	for _, month := range []string{"2026-08", "2027-01"} {
		t.Run("billing month "+month+" outside the project contract returns 400", func(t *testing.T) {
			mock := testutil.SetupMockDB(t)
			projectID := uuid.New()

			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
				WithArgs(projectID.String()).
				WillReturnRows(projectRows(projectID))

			status := postInvoice(app, invoiceBody(map[string]any{
				"project_id":      projectID.String(),
				"periode_tagihan": []string{month},
			}))
			assert.Equal(t, fiber.StatusBadRequest, status)
		})
	}

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
			WillReturnRows(projectRows(projectID))
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

func TestGetAllInvoicesGroupAndYearFilters(t *testing.T) {
	t.Run("project_group_id and periode_year are applied to both queries", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		app := setupApp()
		app.Get("/invoices", handlers.GetAllInvoices)
		groupID := uuid.New().String()

		mock.ExpectQuery(`SELECT \* FROM "invoices" WHERE project_id IN \(SELECT id FROM projects WHERE project_group_id = \$1\) AND substr\(periode_tagihan\[1\], 1, 4\) = \$2`).
			WithArgs(groupID, "2026").
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_invoice"}).AddRow(uuid.New(), "INV-001"))
		mock.ExpectQuery(`SELECT count\(\*\) FROM "invoices" WHERE project_id IN \(SELECT id FROM projects WHERE project_group_id = \$1\) AND substr\(periode_tagihan\[1\], 1, 4\) = \$2`).
			WithArgs(groupID, "2026").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		resp, _ := app.Test(httptest.NewRequest("GET", "/invoices?project_group_id="+groupID+"&periode_year=2026", nil))
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	for _, year := range []string{"26", "20266", "abcd", "2026-01"} {
		t.Run("invalid year "+year+" returns 400", func(t *testing.T) {
			testutil.SetupMockDB(t)
			app := setupApp()
			app.Get("/invoices", handlers.GetAllInvoices)

			resp, _ := app.Test(httptest.NewRequest("GET", "/invoices?periode_year="+year, nil))
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		})
	}
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
