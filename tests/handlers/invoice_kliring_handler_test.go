package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/tests/testutil"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// invoiceRowForKliring is a Submit invoice billed 2026-10-01 for Rp 100.000.000.
func invoiceRowForKliring(id uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "no_invoice", "nilai", "tgl_tagihan", "status", "metode_pembayaran", "periode_tagihan", "manajemen_fee"}).
		AddRow(id, "INV-001", 100_000_000.0, "2026-10-01", "Submit", "Kontrak", "{2026-10}", 0.0)
}

func putKliring(app *fiber.App, id uuid.UUID, body map[string]any) (int, string) {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/invoices/"+id.String()+"/kliring", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody)
}

func TestKliringInvoiceValidation(t *testing.T) {
	app := setupApp()
	app.Put("/invoices/:id/kliring", handlers.KliringInvoice)
	ppn := uuid.New().String()

	tests := []struct {
		name     string
		body     map[string]any
		contains string
	}{
		{
			name:     "missing tgl_bayar",
			body:     map[string]any{"nilai_penerimaan": 100_000_000.0},
			contains: "tgl_bayar wajib diisi",
		},
		{
			name:     "tgl_bayar on the billing date",
			body:     map[string]any{"tgl_bayar": "2026-10-01", "nilai_penerimaan": 100_000_000.0},
			contains: "setelah tgl_tagihan",
		},
		{
			name:     "gap not explained by deductions",
			body:     map[string]any{"tgl_bayar": "2026-10-20", "nilai_penerimaan": 98_000_000.0},
			contains: "masih kurang Rp 2.000.000,00",
		},
		{
			name: "deductions short of the gap",
			body: map[string]any{
				"tgl_bayar": "2026-10-20", "nilai_penerimaan": 98_000_000.0,
				"potongan": []map[string]any{{"deduction_type_id": ppn, "jumlah": 1_000_000.0}},
			},
			contains: "masih kurang Rp 1.000.000,00",
		},
		{
			name: "deductions exceed the gap",
			body: map[string]any{
				"tgl_bayar": "2026-10-20", "nilai_penerimaan": 98_000_000.0,
				"potongan": []map[string]any{{"deduction_type_id": ppn, "jumlah": 3_000_000.0}},
			},
			contains: "melebihi nilai tagihan sebesar Rp 1.000.000,00",
		},
		{
			name: "zero deduction amount",
			body: map[string]any{
				"tgl_bayar": "2026-10-20", "nilai_penerimaan": 100_000_000.0,
				"potongan": []map[string]any{{"deduction_type_id": ppn, "jumlah": 0.0}},
			},
			contains: "harus lebih dari 0",
		},
		{
			name: "deduction type id is not a uuid",
			body: map[string]any{
				"tgl_bayar": "2026-10-20", "nilai_penerimaan": 98_000_000.0,
				"potongan": []map[string]any{{"deduction_type_id": "PPh", "jumlah": 2_000_000.0}},
			},
			contains: "tidak valid",
		},
		{
			name:     "negative nilai_penerimaan",
			body:     map[string]any{"tgl_bayar": "2026-10-20", "nilai_penerimaan": -1.0},
			contains: "tidak boleh negatif",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutil.SetupMockDB(t)
			invoiceID := uuid.New()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE id = $1`)).
				WithArgs(invoiceID.String()).
				WillReturnRows(invoiceRowForKliring(invoiceID))

			status, body := putKliring(app, invoiceID, tt.body)
			assert.Equal(t, fiber.StatusBadRequest, status, body)
			assert.True(t, strings.Contains(body, tt.contains), "body %q should contain %q", body, tt.contains)
		})
	}
}

func TestKliringInvoiceNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Put("/invoices/:id/kliring", handlers.KliringInvoice)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE id = $1`)).
		WithArgs(id.String()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	status, _ := putKliring(app, id, map[string]any{"tgl_bayar": "2026-10-20", "nilai_penerimaan": 100_000_000.0})
	assert.Equal(t, fiber.StatusNotFound, status)
}

func TestCancelKliringRejectsSubmitInvoice(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Delete("/invoices/:id/kliring", handlers.CancelKliringInvoice)

	id := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "invoices" WHERE id = $1`)).
		WithArgs(id.String()).
		WillReturnRows(invoiceRowForKliring(id))

	req := httptest.NewRequest("DELETE", "/invoices/"+id.String()+"/kliring", nil)
	resp, _ := app.Test(req)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(body), "belum Kliring")
}

func TestCreateInvoiceRejectsClearingFields(t *testing.T) {
	app := setupApp()
	app.Post("/invoices", handlers.CreateInvoice)

	// The 400 is returned before any query, so no database expectations are set.
	testutil.SetupMockDB(t)
	status := postInvoice(app, invoiceBody(map[string]any{"status": "Kliring", "tgl_bayar": "2026-10-20"}))
	assert.Equal(t, fiber.StatusBadRequest, status)
}
