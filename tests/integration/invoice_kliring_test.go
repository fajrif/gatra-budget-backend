package integration

import (
	"fmt"
	"net/http"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// newDeductionType inserts a deduction type inside the test transaction.
func newDeductionType(t *testing.T, tx *gorm.DB, nama string) models.DeductionType {
	t.Helper()
	dt := models.DeductionType{Nama: nama}
	if err := tx.Create(&dt).Error; err != nil {
		t.Fatalf("create deduction type %q: %v", nama, err)
	}
	return dt
}

// kliringBody builds a clearing request with the given potongan (jumlah each).
func kliringBody(bayar string, penerimaan float64, deductionTypeID string, jumlah ...float64) string {
	potongan := ""
	for i, j := range jumlah {
		if i > 0 {
			potongan += ","
		}
		potongan += fmt.Sprintf(`{"deduction_type_id":"%s","jumlah":%.2f}`, deductionTypeID, j)
	}
	return fmt.Sprintf(`{"tgl_bayar":"%s","nilai_penerimaan":%.2f,"potongan":[%s]}`, bayar, penerimaan, potongan)
}

// kliringApp returns an app with the Kliring and invoice routes.
func kliringApp() *fiber.App {
	a := app()
	a.Put("/invoices/:id/kliring", handlers.KliringInvoice)
	a.Delete("/invoices/:id/kliring", handlers.CancelKliringInvoice)
	a.Get("/invoices/:id", handlers.GetInvoice)
	a.Delete("/invoices/:id", handlers.DeleteInvoice)
	return a
}

func TestKliringRejectsUnbalancedAmounts(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	pph := newDeductionType(t, tx, "PPh Badan")
	inv := newInvoice(t, project.ID, "INV-K-UNB")
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	a := kliringApp()

	// 98jt received with no deduction leaves 2jt unexplained.
	resp := doJSON(t, a, "PUT", "/invoices/"+inv.ID.String()+"/kliring", kliringBody("2026-10-25", 98_000_000, pph.ID.String()))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unexplained gap, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	// The rejected clearing must not change the invoice.
	var stored models.Invoice
	tx.First(&stored, "id = ?", inv.ID)
	if stored.Status != models.InvoiceStatusSubmit || stored.TglBayar != nil || stored.NilaiPenerimaan != nil {
		t.Fatalf("a rejected clearing must leave the invoice as Submit, got %+v", stored)
	}
}

func TestKliringRejectsUnknownDeductionType(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	inv := newInvoice(t, project.ID, "INV-K-UNK")
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	resp := doJSON(t, kliringApp(), "PUT", "/invoices/"+inv.ID.String()+"/kliring",
		kliringBody("2026-10-25", 98_000_000, uuid.NewString(), 2_000_000))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an unknown deduction type, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
}

func TestKliringWithMultipleDeductionsAndCancel(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	pph := newDeductionType(t, tx, "PPh Badan")
	denda := newDeductionType(t, tx, "Denda")
	inv := newInvoice(t, project.ID, "INV-K-MULTI")
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	a := kliringApp()
	path := "/invoices/" + inv.ID.String() + "/kliring"

	// 100jt billed, 98jt received: PPh Badan 1jt + Denda 1jt balances the invoice.
	resp := doJSON(t, a, "PUT", path, kliringBody("2026-10-25", 98_000_000, pph.ID.String(), 1_000_000))
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 with only 1jt of 2jt explained, got %d", resp.StatusCode)
	}
	resp = doJSON(t, a, "PUT", path, fmt.Sprintf(`{"tgl_bayar":"2026-10-25","nilai_penerimaan":98000000,"potongan":[
		{"deduction_type_id":"%s","jumlah":1000000},{"deduction_type_id":"%s","jumlah":1000000,"keterangan":"terlambat"}]}`,
		pph.ID, denda.ID))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	cleared := readBody(t, resp)
	if len(cleared["deductions"].([]any)) != 2 {
		t.Fatalf("expected two deductions, got %v", cleared["deductions"])
	}

	// Cancelling returns the invoice to Submit and removes its deductions.
	resp = doJSON(t, a, "DELETE", path, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 cancelling, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	back := readBody(t, resp)
	if back["status"] != "Submit" || back["tgl_bayar"] != nil || back["nilai_penerimaan"] != nil {
		t.Fatalf("expected Submit without clearing data, got %v", back)
	}
	var count int64
	tx.Model(&models.InvoiceDeduction{}).Where("invoice_id = ?", inv.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected the deductions to be removed, found %d", count)
	}
}

func TestDeductionTypeInUseCannotBeDeletedAndDeleteInvoiceCascades(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	pph := newDeductionType(t, tx, "PPh Badan")
	inv := newInvoice(t, project.ID, "INV-K-USE")
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	a := kliringApp()
	a.Delete("/deduction-types/:id", handlers.DeleteDeductionType)

	resp := doJSON(t, a, "PUT", "/invoices/"+inv.ID.String()+"/kliring", kliringBody("2026-10-25", 98_000_000, pph.ID.String(), 2_000_000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	resp = doJSON(t, a, "DELETE", "/deduction-types/"+pph.ID.String(), "")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 deleting a deduction type in use, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	// Deleting the invoice removes its deductions through ON DELETE CASCADE.
	resp = doJSON(t, a, "DELETE", "/invoices/"+inv.ID.String(), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 deleting the invoice, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
	var count int64
	tx.Model(&models.InvoiceDeduction{}).Where("invoice_id = ?", inv.ID).Count(&count)
	if count != 0 {
		t.Fatalf("expected ON DELETE CASCADE to remove the deductions, found %d", count)
	}
}

func TestDeductionTypeNameIsUnique(t *testing.T) {
	tx := withTx(t)
	newDeductionType(t, tx, "PPh Badan")
	a := kliringApp()
	a.Post("/deduction-types", handlers.CreateDeductionType)

	resp := doJSON(t, a, "POST", "/deduction-types", `{"nama":"pph badan"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 for a duplicate name (case-insensitive), got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}
}

func TestMonthlyReportSplitsReceiptAndDeductions(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	pph := newDeductionType(t, tx, "PPh Badan")
	inv := newInvoice(t, project.ID, "INV-K-RPT")
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}
	a := kliringApp()
	a.Get("/reports/monthly", handlers.GetMonthlyReport)

	resp := doJSON(t, a, "PUT", "/invoices/"+inv.ID.String()+"/kliring", kliringBody("2026-10-25", 98_000_000, pph.ID.String(), 2_000_000))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, readRawBody(t, resp))
	}

	resp = doJSON(t, a, "GET", "/reports/monthly?year=2026", "")
	body := readBody(t, resp)
	rows := body["rows"].([]any)
	oct := rows[9].(map[string]any)
	if oct["month"] != "2026-10" {
		t.Fatalf("expected row 10 to be 2026-10, got %v", oct["month"])
	}
	assertFloat(t, oct, "tagihan", 100_000_000)
	assertFloat(t, oct, "penerimaan", 98_000_000)
	assertFloat(t, oct, "potongan", 2_000_000)

	totals := body["totals"].(map[string]any)
	assertFloat(t, totals, "penerimaan", 98_000_000)
	assertFloat(t, totals, "potongan", 2_000_000)
}

func TestImportKliringWithJenisPotongan(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	newDeductionType(t, tx, "PPh Badan")
	a := importApp()

	row := func(n int, no string) map[string]any {
		return map[string]any{
			"row": n, "no_sp2k": project.NoSP2K, "no_invoice": no, "nama_invoice": "Termin",
			"tgl_tagihan": "2026-10-01", "status": "Kliring", "tgl_bayar": "2026-10-25",
			"nilai": "100000000", "metode_pembayaran": "kontrak", "periode_tagihan": "2026-09",
		}
	}

	t.Run("a lower nilai_penerimaan needs jenis_potongan", func(t *testing.T) {
		short := row(2, "INV-IK-1")
		short["nilai_penerimaan"] = "98000000"
		status, body := postImport(t, a, "/invoices/import", []map[string]any{short})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %v", status, body)
		}
		if _, ok := importErrorsOf(t, body)["2:jenis_potongan"]; !ok {
			t.Fatalf("expected an error on jenis_potongan, got %v", importErrorsOf(t, body))
		}
	})

	t.Run("the gap is saved as one deduction of the named type", func(t *testing.T) {
		withGap := row(2, "INV-IK-2")
		withGap["nilai_penerimaan"] = "98000000"
		withGap["jenis_potongan"] = "pph badan"
		status, body := postImport(t, a, "/invoices/import", []map[string]any{withGap, row(3, "INV-IK-3")})
		if status != http.StatusOK || body["created"] != float64(2) {
			t.Fatalf("expected 2 created, got %d: %v", status, body)
		}

		var inv models.Invoice
		tx.Where("no_invoice = ?", "INV-IK-2").First(&inv)
		if inv.Status != models.InvoiceStatusKliring || inv.NilaiPenerimaan == nil || *inv.NilaiPenerimaan != 98_000_000 {
			t.Fatalf("expected Kliring with nilai_penerimaan 98.000.000, got %+v", inv)
		}
		var deductions []models.InvoiceDeduction
		tx.Where("invoice_id = ?", inv.ID).Find(&deductions)
		if len(deductions) != 1 || deductions[0].Jumlah != 2_000_000 {
			t.Fatalf("expected one 2.000.000 deduction, got %+v", deductions)
		}
	})

	t.Run("jenis_potongan without a gap is refused", func(t *testing.T) {
		extra := row(2, "INV-IK-4")
		extra["jenis_potongan"] = "PPh Badan"
		status, body := postImport(t, a, "/invoices/import", []map[string]any{extra})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %v", status, body)
		}
	})

}
