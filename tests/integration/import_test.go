package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"gatra-budget-backend/config"
	"gatra-budget-backend/handlers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

func importApp() *fiber.App {
	a := app()
	a.Post("/projects/import", handlers.ImportProjects)
	a.Post("/transactions/import", handlers.ImportTransactions)
	a.Post("/invoices/import", handlers.ImportInvoices)
	return a
}

// postImport sends {"rows": rows} and returns status and decoded body.
func postImport(t *testing.T, a *fiber.App, path string, rows []map[string]any) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"rows": rows})
	resp := doJSON(t, a, "POST", path, string(raw))
	return resp.StatusCode, readBody(t, resp)
}

// importErrorsOf flattens the 422 body to "row:field" keys.
func importErrorsOf(t *testing.T, body map[string]any) map[string]string {
	t.Helper()
	out := map[string]string{}
	list, _ := body["errors"].([]any)
	for _, e := range list {
		m := e.(map[string]any)
		key := string(rune('0'+int(m["row"].(float64)))) + ":" + m["field"].(string)
		out[key] = m["message"].(string)
	}
	return out
}

func countRows(t *testing.T, model any) int64 {
	t.Helper()
	var n int64
	if err := config.DB.Model(model).Count(&n).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	return n
}

func validProjectRow(row int, sp2k string) map[string]any {
	return map[string]any{
		"row": row, "no_sp2k": sp2k, "judul_pekerjaan": "Proyek " + sp2k,
		"tanggal_mulai": "2026-01-01", "tanggal_selesai": "2026-12-31", "nilai_pekerjaan": "1000000000",
	}
}

func TestImportProjects(t *testing.T) {
	tx := withTx(t)
	newTestUser(t, tx, "import-projects@gatrahitawasana.com", "user")
	client := models.Client{Name: "PT Import Klien"}
	contract := models.ContractType{Name: "Lumpsum Import"}
	tx.Create(&client)
	tx.Create(&contract)
	a := importApp()

	t.Run("collects every error across rows and saves nothing", func(t *testing.T) {
		before := countRows(t, &models.Project{})
		bad := validProjectRow(3, "BAD-1")
		bad["tanggal_selesai"] = "2025-12-31" // before tanggal_mulai
		empty := map[string]any{"row": 4}     // everything blank
		unknown := validProjectRow(5, "BAD-3")
		unknown["client"] = "PT Tidak Ada"
		unknown["nilai_pekerjaan"] = "abc"
		dup := validProjectRow(6, "bad-1") // same SP2K as row 3, different case

		status, body := postImport(t, a, "/projects/import",
			[]map[string]any{validProjectRow(2, "OK-1"), bad, empty, unknown, dup})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %v", status, body)
		}

		errs := importErrorsOf(t, body)
		for _, key := range []string{
			"3:tanggal_selesai", "4:no_sp2k", "4:judul_pekerjaan", "4:tanggal_mulai",
			"4:tanggal_selesai", "4:nilai_pekerjaan", "5:client", "5:nilai_pekerjaan", "6:no_sp2k",
		} {
			if _, ok := errs[key]; !ok {
				t.Errorf("expected an error for %s, got %v", key, errs)
			}
		}
		if _, ok := errs["2:no_sp2k"]; ok {
			t.Errorf("valid row 2 must not be reported: %v", errs)
		}
		if after := countRows(t, &models.Project{}); after != before {
			t.Fatalf("a failed import must save nothing: %d -> %d", before, after)
		}
	})

	t.Run("creates, then updates the same no_sp2k on re-import", func(t *testing.T) {
		row := validProjectRow(2, "IMP-1")
		row["client"] = " pt import klien " // case/space-insensitive
		row["contract_type"] = "Lumpsum Import"

		status, body := postImport(t, a, "/projects/import", []map[string]any{row})
		if status != http.StatusOK || body["created"] != float64(1) || body["updated"] != float64(0) {
			t.Fatalf("expected 1 created, got %d: %v", status, body)
		}

		row["judul_pekerjaan"] = "Judul Diperbarui"
		row["no_sp2k"] = "imp-1"
		status, body = postImport(t, a, "/projects/import", []map[string]any{row})
		if status != http.StatusOK || body["created"] != float64(0) || body["updated"] != float64(1) {
			t.Fatalf("expected 1 updated, got %d: %v", status, body)
		}

		var projects []models.Project
		tx.Where("LOWER(no_sp2k) = ?", "imp-1").Find(&projects)
		if len(projects) != 1 || projects[0].JudulPekerjaan != "Judul Diperbarui" {
			t.Fatalf("expected one updated project, got %+v", projects)
		}
		if projects[0].ClientID == nil || *projects[0].ClientID != client.ID {
			t.Fatalf("client not resolved by name: %+v", projects[0].ClientID)
		}
		if projects[0].JangkaWaktu == nil || *projects[0].JangkaWaktu != 11 {
			t.Fatalf("jangka_waktu should be server-computed")
		}
	})
}

func TestImportTransactions(t *testing.T) {
	tx := withTx(t)
	project, costType := newTestProject(t, tx)
	a := importApp()

	row := func(n int) map[string]any {
		return map[string]any{
			"row": n, "no_sp2k": project.NoSP2K, "cost_type": costType.NamaBiaya,
			"tanggal_transaksi": "2026-03-15", "jumlah_realisasi": "2500000",
		}
	}

	t.Run("reports unknown project, cost type and bad amounts without saving", func(t *testing.T) {
		before := countRows(t, &models.Transaction{})
		a1 := row(3)
		a1["no_sp2k"] = "TIDAK-ADA"
		a2 := row(4)
		a2["cost_type"] = "Tidak Ada"
		a3 := row(5)
		a3["jumlah_realisasi"] = "50000" // below the 100,000 minimum
		a4 := row(6)
		a4["jumlah_tenaga_kerja"] = "-2"

		status, body := postImport(t, a, "/transactions/import", []map[string]any{row(2), a1, a2, a3, a4})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %v", status, body)
		}
		errs := importErrorsOf(t, body)
		for _, key := range []string{"3:no_sp2k", "4:cost_type", "5:jumlah_realisasi", "6:jumlah_tenaga_kerja"} {
			if _, ok := errs[key]; !ok {
				t.Errorf("expected an error for %s, got %v", key, errs)
			}
		}
		if after := countRows(t, &models.Transaction{}); after != before {
			t.Fatalf("a failed import must save nothing: %d -> %d", before, after)
		}
	})

	t.Run("every row creates a transaction with a server-computed month", func(t *testing.T) {
		status, body := postImport(t, a, "/transactions/import", []map[string]any{row(2), row(3)})
		if status != http.StatusOK || body["created"] != float64(2) {
			t.Fatalf("expected 2 created, got %d: %v", status, body)
		}
		var saved []models.Transaction
		tx.Where("project_id = ?", project.ID).Find(&saved)
		if len(saved) != 2 || saved[0].BulanRealisasi == nil || *saved[0].BulanRealisasi != "2026-03" {
			t.Fatalf("unexpected saved transactions: %+v", saved)
		}
	})
}

func TestImportInvoices(t *testing.T) {
	tx := withTx(t)
	project, _ := newTestProject(t, tx)
	a := importApp()

	row := func(n int, no string) map[string]any {
		return map[string]any{
			"row": n, "no_sp2k": project.NoSP2K, "no_invoice": no, "nama_invoice": "Termin",
			"tgl_tagihan": "2026-10-01", "status": "submit", "nilai": "100000000", "manajemen_fee": "5000000",
		}
	}

	t.Run("applies the invoice rules per row and saves nothing", func(t *testing.T) {
		before := countRows(t, &models.Invoice{})
		fee := row(3, "INV-X2")
		fee["manajemen_fee"] = "200000000"
		paidBefore := row(4, "INV-X3")
		paidBefore["status"] = "Clearing"
		paidBefore["tgl_bayar"] = "2026-09-01"
		clearingNoDate := row(5, "INV-X4")
		clearingNoDate["status"] = "Clearing"
		submitWithDate := row(6, "INV-X5")
		submitWithDate["tgl_bayar"] = "2026-10-20"
		badStatus := row(7, "INV-X6")
		badStatus["status"] = "Paid"
		dup := row(8, "INV-X1")

		status, body := postImport(t, a, "/invoices/import",
			[]map[string]any{row(2, "INV-X1"), fee, paidBefore, clearingNoDate, submitWithDate, badStatus, dup})
		if status != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %v", status, body)
		}
		errs := importErrorsOf(t, body)
		for _, key := range []string{
			"3:manajemen_fee", "4:tgl_bayar", "5:tgl_bayar", "6:tgl_bayar", "7:status", "8:no_invoice",
		} {
			if _, ok := errs[key]; !ok {
				t.Errorf("expected an error for %s, got %v", key, errs)
			}
		}
		if after := countRows(t, &models.Invoice{}); after != before {
			t.Fatalf("a failed import must save nothing: %d -> %d", before, after)
		}
	})

	t.Run("creates, then updates by no_invoice", func(t *testing.T) {
		paid := row(3, "INV-OK-2")
		paid["status"] = "CLEARING"
		paid["tgl_bayar"] = "2026-10-20"

		status, body := postImport(t, a, "/invoices/import", []map[string]any{row(2, "INV-OK-1"), paid})
		if status != http.StatusOK || body["created"] != float64(2) {
			t.Fatalf("expected 2 created, got %d: %v", status, body)
		}

		changed := row(2, "INV-OK-1")
		changed["nilai"] = "150000000"
		status, body = postImport(t, a, "/invoices/import", []map[string]any{changed, row(3, "INV-OK-3")})
		if status != http.StatusOK || body["created"] != float64(1) || body["updated"] != float64(1) {
			t.Fatalf("expected 1 created and 1 updated, got %d: %v", status, body)
		}

		var inv models.Invoice
		tx.Where("no_invoice = ?", "INV-OK-1").First(&inv)
		if inv.Nilai != 150_000_000 {
			t.Fatalf("expected updated nilai, got %v", inv.Nilai)
		}
		var clearing models.Invoice
		tx.Where("no_invoice = ?", "INV-OK-2").First(&clearing)
		if clearing.Status != models.InvoiceStatusClearing || clearing.TglBayar == nil {
			t.Fatalf("expected Clearing with tgl_bayar, got %+v", clearing)
		}
	})
}
