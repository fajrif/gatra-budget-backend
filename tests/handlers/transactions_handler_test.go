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

func TestCreateTransaction(t *testing.T) {
	app := setupApp()
	app.Post("/transactions", handlers.CreateTransaction)

	t.Run("jumlah_realisasi below the minimum is rejected", func(t *testing.T) {
		testutil.SetupMockDB(t)
		reqBody := map[string]any{
			"project_id":        uuid.New().String(),
			"cost_type_id":      uuid.New().String(),
			"tanggal_transaksi": "2024-03-15",
			"jumlah_realisasi":  50_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})

	t.Run("sets bulan_realisasi (YYYY-MM) from tanggal_transaksi", func(t *testing.T) {
		mock := testutil.SetupMockDB(t)
		projectID := uuid.New()
		costTypeID := uuid.New()

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "projects" WHERE id = $1`)).
			WithArgs(projectID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "no_sp2k", "tanggal_mulai", "tanggal_selesai", "nilai_pekerjaan"}).
				AddRow(projectID, "SP2K-001", "2024-01-01", "2024-12-31", 1_000_000_000.0))

		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "cost_types" WHERE id = $1`)).
			WithArgs(costTypeID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nama_biaya"}).AddRow(costTypeID, "Gaji & Tunjangan"))

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "transactions"`)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
				AddRow(uuid.New(), time.Now(), time.Now()))
		mock.ExpectCommit()

		reqBody := map[string]any{
			"project_id":        projectID.String(),
			"cost_type_id":      costTypeID.String(),
			"tanggal_transaksi": "2024-03-15",
			"jumlah_realisasi":  500_000_000.0,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/transactions", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, _ := app.Test(req)
		respBody, _ := io.ReadAll(resp.Body)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode, string(respBody))

		var created map[string]any
		json.Unmarshal(respBody, &created)
		assert.Equal(t, "2024-03", created["bulan_realisasi"])
		assert.NotContains(t, created, "nilai_management_fee")
		assert.NotContains(t, created, "sla_status")
	})
}

func TestGetTransaction(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Get("/transactions/:id", handlers.GetTransaction)

	t.Run("not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "transactions" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnError(gorm.ErrRecordNotFound)

		req := httptest.NewRequest("GET", "/transactions/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}

func TestDeleteTransaction(t *testing.T) {
	mock := testutil.SetupMockDB(t)

	app := setupApp()
	app.Delete("/transactions/:id", handlers.DeleteTransaction)

	t.Run("Successfully delete transaction", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "transactions" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/transactions/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("Transaction not found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM "transactions" WHERE id = $1`)).
			WithArgs(id.String()).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		req := httptest.NewRequest("DELETE", "/transactions/"+id.String(), nil)
		resp, _ := app.Test(req)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	})
}
