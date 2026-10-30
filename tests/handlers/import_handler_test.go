package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"gatra-budget-backend/handlers"
	"gatra-budget-backend/helpers"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// The import endpoints reject an unusable body before touching the database.
func TestImportRowLimits(t *testing.T) {
	app := setupApp()
	app.Post("/projects/import", handlers.ImportProjects)
	app.Post("/transactions/import", handlers.ImportTransactions)
	app.Post("/invoices/import", handlers.ImportInvoices)

	tooMany := make([]map[string]any, helpers.MaxImportRows+1)
	for i := range tooMany {
		tooMany[i] = map[string]any{"row": i + 2}
	}

	bodies := map[string]any{
		"empty rows":   map[string]any{"rows": []any{}},
		"missing rows": map[string]any{},
		"too many":     map[string]any{"rows": tooMany},
	}
	for _, path := range []string{"/projects/import", "/transactions/import", "/invoices/import"} {
		for name, payload := range bodies {
			t.Run(path+" "+name, func(t *testing.T) {
				b, _ := json.Marshal(payload)
				req := httptest.NewRequest("POST", path, bytes.NewReader(b))
				req.Header.Set("Content-Type", "application/json")
				resp, _ := app.Test(req)
				assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
			})
		}
	}
}
