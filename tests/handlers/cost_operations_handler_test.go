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
)

func postCostOperation(t *testing.T, app *fiber.App, body map[string]any) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/cost-operations", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	out, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	_ = json.Unmarshal(out, &parsed)
	return resp.StatusCode, parsed
}

func TestCreateCostOperation(t *testing.T) {
	app := setupApp()
	app.Post("/cost-operations", handlers.CreateCostOperation)

	t.Run("rejects missing required fields and small amounts", func(t *testing.T) {
		valid := func() map[string]any {
			return map[string]any{
				"operational_cost_type_id": uuid.New().String(),
				"tanggal_transaksi":        "2024-03-15",
				"jumlah":                   500_000.0,
			}
		}
		cases := map[string]func(m map[string]any){
			"no type":       func(m map[string]any) { delete(m, "operational_cost_type_id") },
			"no date":       func(m map[string]any) { delete(m, "tanggal_transaksi") },
			"amount at min": func(m map[string]any) { m["jumlah"] = 100_000.0 },
			"no amount":     func(m map[string]any) { delete(m, "jumlah") },
		}
		for name, mutate := range cases {
			t.Run(name, func(t *testing.T) {
				testutil.SetupMockDB(t)
				body := valid()
				mutate(body)
				status, _ := postCostOperation(t, app, body)
				assert.Equal(t, fiber.StatusBadRequest, status)
			})
		}
	})

	t.Run("unknown cost type is 404", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		typeID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "operational_cost_types" WHERE id = $1`)).
			WithArgs(typeID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		status, _ := postCostOperation(t, app, map[string]any{
			"operational_cost_type_id": typeID.String(),
			"tanggal_transaksi":        "2024-03-15",
			"jumlah":                   500_000.0,
		})
		assert.Equal(t, fiber.StatusNotFound, status)
	})

	t.Run("sets bulan_realisasi from tanggal_transaksi", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		typeID := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "operational_cost_types" WHERE id = $1`)).
			WithArgs(typeID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nama_biaya"}).AddRow(typeID, "Konsumsi"))
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "cost_operations"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))
		mock.ExpectCommit()

		status, created := postCostOperation(t, app, map[string]any{
			"operational_cost_type_id": typeID.String(),
			"tanggal_transaksi":        "2024-03-15",
			"jumlah":                   500_000.0,
		})
		assert.Equal(t, fiber.StatusCreated, status)
		assert.Equal(t, "2024-03", created["bulan_realisasi"])
	})
}

func TestGetAllCostOperationsRejectsBadMonth(t *testing.T) {
	testutil.SetupMockDB(t)
	app := setupApp()
	app.Get("/cost-operations", handlers.GetAllCostOperations)

	resp, _ := app.Test(httptest.NewRequest("GET", "/cost-operations?month=2024-13", nil))
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeleteCostOperationNotFound(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Delete("/cost-operations/:id", handlers.DeleteCostOperation)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "cost_operations"`)).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	resp, _ := app.Test(httptest.NewRequest("DELETE", "/cost-operations/"+uuid.New().String(), nil))
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestDeleteOperationalCostTypeInUseIsConflict(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Delete("/operational-cost-types/:id", handlers.DeleteOperationalCostType)

	mock.ExpectQuery(regexp.QuoteMeta(`FROM "cost_operations"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	resp, _ := app.Test(httptest.NewRequest("DELETE", "/operational-cost-types/"+uuid.New().String(), nil))
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}

func TestCreateOperationalCostTypeDuplicateName(t *testing.T) {
	mock := testutil.SetupMockDB(t)
	app := setupApp()
	app.Post("/operational-cost-types", handlers.CreateOperationalCostType)

	mock.ExpectQuery(regexp.QuoteMeta(`FROM "operational_cost_types"`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req := httptest.NewRequest("POST", "/operational-cost-types", bytes.NewReader([]byte(`{"nama_biaya":" Konsumsi "}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
}
