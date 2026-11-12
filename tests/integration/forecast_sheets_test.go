package integration

import (
	"net/http"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

func forecastApp() *fiber.App {
	a := app()
	a.Get("/forecast-sheets", handlers.GetAllForecastSheets)
	a.Get("/forecast-sheets/:id", handlers.GetForecastSheet)
	a.Post("/forecast-sheets", handlers.CreateForecastSheet)
	a.Put("/forecast-sheets/:id", handlers.UpdateForecastSheet)
	a.Delete("/forecast-sheets/:id", handlers.DeleteForecastSheet)
	return a
}

func sheetFrom(t *testing.T, resp *http.Response, want int) map[string]any {
	t.Helper()
	if resp.StatusCode != want {
		t.Fatalf("expected %d, got %d: %s", want, resp.StatusCode, readRawBody(t, resp))
	}
	return readBody(t, resp)["forecast_sheet"].(map[string]any)
}

func TestForecastSheetLifecycle(t *testing.T) {
	tx := withTx(t)
	a := forecastApp()
	project, _ := newTestProject(t, tx)

	inv1 := newInvoice(t, project.ID, "FC-1")
	inv2 := newInvoice(t, project.ID, "FC-2")
	inv2.Nilai = 250_000_000
	inv3 := newInvoice(t, project.ID, "FC-3") // will be paid before the sheet is saved
	inv3.Status = models.InvoiceStatusKliring
	paid := inv3.TglTagihan
	inv3.TglBayar = &paid
	for _, inv := range []*models.Invoice{&inv1, &inv2, &inv3} {
		if err := tx.Create(inv).Error; err != nil {
			t.Fatalf("create invoice: %v", err)
		}
	}

	// Create: the paid invoice is dropped, the unpaid ones are kept.
	body := `{"nama":"Perkiraan Oct, 2026","bulan":"2026-10","saldo":500000000,
		"expenses":[{"item":"Gaji","tanggal":"2026-10-25","jumlah":300000000},{"item":"Listrik","tanggal":"2026-10-05","jumlah":15000000}],
		"invoice_ids":["` + inv1.ID.String() + `","` + inv2.ID.String() + `","` + inv3.ID.String() + `","` + inv1.ID.String() + `"]}`
	sheet := sheetFrom(t, doJSON(t, a, "POST", "/forecast-sheets", body), http.StatusCreated)
	id := sheet["id"].(string)
	if got := len(sheet["invoice_ids"].([]any)); got != 2 {
		t.Fatalf("expected 2 ticked invoices (duplicate and paid dropped), got %d", got)
	}
	exp := sheet["expenses"].([]any)
	if len(exp) != 2 || exp[0].(map[string]any)["item"] != "Gaji" {
		t.Fatalf("expenses must keep the order sent: %v", exp)
	}

	// Names are unique regardless of case.
	resp := doJSON(t, a, "POST", "/forecast-sheets", `{"nama":" perkiraan oct, 2026 ","bulan":"2026-11"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}

	// List totals.
	resp = doJSON(t, a, "GET", "/forecast-sheets?all=true", "")
	list := readBody(t, resp)["forecast_sheets"].([]any)
	if len(list) != 1 {
		t.Fatalf("expected 1 sheet, got %d", len(list))
	}
	row := list[0].(map[string]any)
	assertFloat(t, row, "total_pengeluaran", 315_000_000)
	assertFloat(t, row, "total_pemasukan", 350_000_000)
	assertFloat(t, row, "jumlah_invoice", 2)

	// Update replaces expenses and ticks; the name may stay the same.
	body = `{"nama":"Perkiraan Oct, 2026","bulan":"2026-10","saldo":100,
		"expenses":[{"item":"Sewa","tanggal":"2026-10-01","jumlah":50000000}],
		"invoice_ids":["` + inv2.ID.String() + `"]}`
	sheet = sheetFrom(t, doJSON(t, a, "PUT", "/forecast-sheets/"+id, body), http.StatusOK)
	if len(sheet["expenses"].([]any)) != 1 || len(sheet["invoice_ids"].([]any)) != 1 {
		t.Fatalf("update must replace children: %v", sheet)
	}
	var kids int64
	tx.Model(&models.ForecastExpense{}).Where("forecast_sheet_id = ?", id).Count(&kids)
	if kids != 1 {
		t.Fatalf("old expense rows must be gone, found %d", kids)
	}

	// An expense outside the sheet's month is rejected and nothing changes.
	resp = doJSON(t, a, "PUT", "/forecast-sheets/"+id,
		`{"nama":"Perkiraan Oct, 2026","bulan":"2026-10","expenses":[{"item":"X","tanggal":"2026-11-02","jumlah":1000}]}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	tx.Model(&models.ForecastExpense{}).Where("forecast_sheet_id = ?", id).Count(&kids)
	if kids != 1 {
		t.Fatalf("a rejected save must not touch the sheet, found %d expenses", kids)
	}

	// A ticked invoice that is paid afterwards disappears from the sheet and totals.
	tx.Model(&models.Invoice{}).Where("id = ?", inv2.ID).Updates(map[string]any{"status": models.InvoiceStatusKliring, "tgl_bayar": "2026-10-20"})
	sheet = sheetFrom(t, doJSON(t, a, "GET", "/forecast-sheets/"+id, ""), http.StatusOK)
	if len(sheet["invoice_ids"].([]any)) != 0 {
		t.Fatalf("a paid invoice must not stay ticked: %v", sheet["invoice_ids"])
	}
	resp = doJSON(t, a, "GET", "/forecast-sheets?search=oct", "")
	row = readBody(t, resp)["forecast_sheets"].([]any)[0].(map[string]any)
	assertFloat(t, row, "total_pemasukan", 0)
	assertFloat(t, row, "jumlah_invoice", 0)

	// Unknown invoice IDs are a 400.
	resp = doJSON(t, a, "PUT", "/forecast-sheets/"+id,
		`{"nama":"Perkiraan Oct, 2026","bulan":"2026-10","invoice_ids":["00000000-0000-0000-0000-000000000001"]}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown invoice, got %d", resp.StatusCode)
	}

	// Delete cascades to the children but leaves invoices alone.
	resp = doJSON(t, a, "DELETE", "/forecast-sheets/"+id, "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	tx.Model(&models.ForecastExpense{}).Where("forecast_sheet_id = ?", id).Count(&kids)
	if kids != 0 {
		t.Fatalf("expenses must be deleted with the sheet, found %d", kids)
	}
	var invoices int64
	tx.Model(&models.Invoice{}).Where("id IN ?", []any{inv1.ID, inv2.ID, inv3.ID}).Count(&invoices)
	if invoices != 3 {
		t.Fatalf("invoices must survive, found %d", invoices)
	}
	if resp = doJSON(t, a, "GET", "/forecast-sheets/"+id, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", resp.StatusCode)
	}
}

func TestForecastSheetDeletedInvoiceDropsTick(t *testing.T) {
	tx := withTx(t)
	a := forecastApp()
	project, _ := newTestProject(t, tx)
	inv := newInvoice(t, project.ID, "FC-DEL")
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatal(err)
	}
	sheet := sheetFrom(t, doJSON(t, a, "POST", "/forecast-sheets",
		`{"nama":"Hapus","bulan":"2026-10","invoice_ids":["`+inv.ID.String()+`"]}`), http.StatusCreated)

	if err := tx.Delete(&models.Invoice{}, "id = ?", inv.ID).Error; err != nil {
		t.Fatalf("deleting an invoice ticked on a sheet must work: %v", err)
	}
	got := sheetFrom(t, doJSON(t, a, "GET", "/forecast-sheets/"+sheet["id"].(string), ""), http.StatusOK)
	if len(got["invoice_ids"].([]any)) != 0 {
		t.Fatalf("expected no ticked invoices, got %v", got["invoice_ids"])
	}
}

func TestInvoiceListPaidMonthFilter(t *testing.T) {
	tx := withTx(t)
	a := app()
	a.Get("/invoices", handlers.GetAllInvoices)
	project, _ := newTestProject(t, tx)

	paidOct := newInvoice(t, project.ID, "PM-OCT")
	paidOct.Status = models.InvoiceStatusKliring
	d1 := models.Date{Time: mustParseDate(t, "2026-10-20")}
	paidOct.TglBayar = &d1
	paidNov := newInvoice(t, project.ID, "PM-NOV")
	paidNov.Status = models.InvoiceStatusKliring
	d2 := models.Date{Time: mustParseDate(t, "2026-11-03")}
	paidNov.TglBayar = &d2
	unpaid := newInvoice(t, project.ID, "PM-UNPAID")
	for _, inv := range []*models.Invoice{&paidOct, &paidNov, &unpaid} {
		if err := tx.Create(inv).Error; err != nil {
			t.Fatal(err)
		}
	}

	resp := doJSON(t, a, "GET", "/invoices?all=true&status=Kliring&paid_month=2026-10", "")
	list := readBody(t, resp)["invoices"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["no_invoice"] != "PM-OCT" {
		t.Fatalf("expected only PM-OCT, got %v", list)
	}
	if resp = doJSON(t, a, "GET", "/invoices?paid_month=2026-13", ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for a bad paid_month, got %d", resp.StatusCode)
	}
}
