package handlers

import (
	"fmt"
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ForecastSheetListItem is a sheet plus the totals shown in the list.
type ForecastSheetListItem struct {
	models.ForecastSheet
	TotalPengeluaran float64 `gorm:"column:total_pengeluaran" json:"total_pengeluaran"`
	// TotalPemasukan only counts ticked invoices that are still unpaid (Submit).
	TotalPemasukan float64 `gorm:"column:total_pemasukan" json:"total_pemasukan"`
	JumlahInvoice  int64   `gorm:"column:jumlah_invoice" json:"jumlah_invoice"`
}

// ForecastSheetDetail is a sheet with its expense lines and ticked invoice IDs.
type ForecastSheetDetail struct {
	models.ForecastSheet
	Expenses   []models.ForecastExpense `json:"expenses"`
	InvoiceIDs []string                 `json:"invoice_ids"`
}

const forecastListSelect = `forecast_sheets.*,
	COALESCE((SELECT SUM(e.jumlah) FROM forecast_expenses e WHERE e.forecast_sheet_id = forecast_sheets.id), 0) AS total_pengeluaran,
	COALESCE((SELECT SUM(i.nilai) FROM forecast_invoices fi JOIN invoices i ON i.id = fi.invoice_id
		WHERE fi.forecast_sheet_id = forecast_sheets.id AND i.status = 'Submit'), 0) AS total_pemasukan,
	(SELECT COUNT(*) FROM forecast_invoices fi JOIN invoices i ON i.id = fi.invoice_id
		WHERE fi.forecast_sheet_id = forecast_sheets.id AND i.status = 'Submit') AS jumlah_invoice`

// forecastNameTaken reports whether another sheet already uses the name
// (case-insensitive). excludeID may be empty.
func forecastNameTaken(db *gorm.DB, name, excludeID string) (bool, error) {
	var count int64
	q := db.Model(&models.ForecastSheet{}).Where("LOWER(nama) = LOWER(?)", name)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ValidateForecastSheetRequest validates the sheet fields and returns the
// expense lines parsed into models (in the order sent). Every expense date must
// fall inside the sheet's month.
func ValidateForecastSheetRequest(req *models.ForecastSheetRequest) ([]models.ForecastExpense, error) {
	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "nama is required")
	}
	if !monthQueryPattern.MatchString(req.Bulan) {
		return nil, fiber.NewError(fiber.StatusBadRequest, "bulan must be in YYYY-MM format")
	}

	expenses := make([]models.ForecastExpense, 0, len(req.Expenses))
	for i, e := range req.Expenses {
		row := i + 1
		item := strings.TrimSpace(e.Item)
		if item == "" {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("pengeluaran baris %d: item is required", row))
		}
		if e.Jumlah <= 0 {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("pengeluaran baris %d: jumlah must be greater than 0", row))
		}
		tanggal, err := helpers.ParseDateString(&e.Tanggal)
		if err != nil || tanggal == nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("pengeluaran baris %d: tanggal must be in YYYY-MM-DD format", row))
		}
		if tanggal.Format("2006-01") != req.Bulan {
			return nil, fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("pengeluaran baris %d: tanggal must be within %s", row, req.Bulan))
		}
		expenses = append(expenses, models.ForecastExpense{
			Item:    item,
			Tanggal: models.Date{Time: *tanggal},
			Jumlah:  e.Jumlah,
			Urutan:  i,
		})
	}
	return expenses, nil
}

// resolveForecastInvoices returns the distinct requested invoice IDs that exist
// and are still unpaid. Unknown IDs are a 400; IDs of invoices that have since
// been paid (Kliring) are dropped, since their money is already in the balance.
func resolveForecastInvoices(db *gorm.DB, ids []string) ([]uuid.UUID, error) {
	seen := make(map[uuid.UUID]bool, len(ids))
	var wanted []uuid.UUID
	for _, raw := range ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusBadRequest, "invalid invoice id: "+raw)
		}
		if !seen[id] {
			seen[id] = true
			wanted = append(wanted, id)
		}
	}
	if len(wanted) == 0 {
		return nil, nil
	}

	var rows []struct {
		ID     uuid.UUID
		Status string
	}
	if err := db.Table("invoices").Select("id, status").Where("id IN ?", wanted).Scan(&rows).Error; err != nil {
		return nil, err
	}
	known := make(map[uuid.UUID]string, len(rows))
	for _, r := range rows {
		known[r.ID] = r.Status
	}

	kept := make([]uuid.UUID, 0, len(wanted))
	for _, id := range wanted {
		status, ok := known[id]
		if !ok {
			return nil, fiber.NewError(fiber.StatusBadRequest, "invoice not found: "+id.String())
		}
		if status == models.InvoiceStatusSubmit {
			kept = append(kept, id)
		}
	}
	return kept, nil
}

// loadForecastSheet reads a sheet with its expenses and ticked invoice IDs.
// Ticked invoices that are no longer unpaid are left out.
func loadForecastSheet(db *gorm.DB, id string) (*ForecastSheetDetail, error) {
	var sheet models.ForecastSheet
	if err := db.Where("id = ?", id).First(&sheet).Error; err != nil {
		return nil, err
	}
	detail := &ForecastSheetDetail{ForecastSheet: sheet, Expenses: []models.ForecastExpense{}, InvoiceIDs: []string{}}
	if err := db.Where("forecast_sheet_id = ?", sheet.ID).Order("urutan ASC").Find(&detail.Expenses).Error; err != nil {
		return nil, err
	}
	if err := db.Table("forecast_invoices fi").
		Select("fi.invoice_id").
		Joins("JOIN invoices i ON i.id = fi.invoice_id").
		Where("fi.forecast_sheet_id = ? AND i.status = ?", sheet.ID, models.InvoiceStatusSubmit).
		Order("i.tgl_tagihan ASC, i.no_invoice ASC").
		Pluck("fi.invoice_id", &detail.InvoiceIDs).Error; err != nil {
		return nil, err
	}
	return detail, nil
}

// saveForecastSheet writes the sheet, replacing its expenses and ticked
// invoices, in one transaction. sheet.ID is empty for a new sheet.
func saveForecastSheet(sheet *models.ForecastSheet, req *models.ForecastSheetRequest, expenses []models.ForecastExpense, invoiceIDs []uuid.UUID) error {
	return config.DB.Transaction(func(tx *gorm.DB) error {
		sheet.Nama = req.Nama
		sheet.Bulan = req.Bulan
		sheet.Saldo = req.Saldo
		if sheet.ID == uuid.Nil {
			if err := tx.Create(sheet).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Save(sheet).Error; err != nil {
				return err
			}
			if err := tx.Delete(&models.ForecastExpense{}, "forecast_sheet_id = ?", sheet.ID).Error; err != nil {
				return err
			}
			if err := tx.Delete(&models.ForecastInvoice{}, "forecast_sheet_id = ?", sheet.ID).Error; err != nil {
				return err
			}
		}

		for i := range expenses {
			expenses[i].ForecastSheetID = sheet.ID
		}
		if len(expenses) > 0 {
			if err := tx.Create(&expenses).Error; err != nil {
				return err
			}
		}
		if len(invoiceIDs) > 0 {
			links := make([]models.ForecastInvoice, len(invoiceIDs))
			for i, id := range invoiceIDs {
				links[i] = models.ForecastInvoice{ForecastSheetID: sheet.ID, InvoiceID: id}
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAllForecastSheets godoc
// @Summary Get all forecast sheets (admin only)
// @Description Sheets with total_pengeluaran, total_pemasukan (ticked invoices still unpaid) and jumlah_invoice
// @Tags ForecastSheets
// @Security BearerAuth
// @Param search query string false "Search by nama"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of forecast sheets with pagination"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/forecast-sheets [get]
func GetAllForecastSheets(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Model(&models.ForecastSheet{}).
		Select(forecastListSelect).
		Order("forecast_sheets.bulan DESC, forecast_sheets.nama ASC")
	queryCount := config.DB.Model(&models.ForecastSheet{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("forecast_sheets.nama ILIKE ?", term)
		queryCount = queryCount.Where("nama ILIKE ?", term)
	}

	var items []ForecastSheetListItem
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Scan(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch forecast sheets"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, items, "forecast_sheets", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}
	return c.JSON(response)
}

// GetForecastSheet godoc
// @Summary Get a forecast sheet by ID (admin only)
// @Description The sheet with its expenses and the IDs of its ticked invoices that are still unpaid
// @Tags ForecastSheets
// @Security BearerAuth
// @Param id path string true "Forecast sheet ID (UUID)"
// @Success 200 {object} map[string]any "Forecast sheet details"
// @Failure 404 {object} map[string]any "Forecast sheet not found"
// @Router /api/forecast-sheets/{id} [get]
func GetForecastSheet(c *fiber.Ctx) error {
	detail, err := loadForecastSheet(config.DB, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Forecast sheet not found"})
	}
	return c.JSON(fiber.Map{"forecast_sheet": detail})
}

// CreateForecastSheet godoc
// @Summary Create a forecast sheet (admin only)
// @Description Saves the whole sheet. Expense dates must fall inside bulan; ticked invoices that are not unpaid are dropped.
// @Tags ForecastSheets
// @Security BearerAuth
// @Param request body models.ForecastSheetRequest true "Forecast sheet"
// @Success 201 {object} map[string]any "Created"
// @Failure 400 {object} map[string]any "Invalid request"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/forecast-sheets [post]
func CreateForecastSheet(c *fiber.Ctx) error {
	var req models.ForecastSheetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	expenses, err := ValidateForecastSheetRequest(&req)
	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	taken, err := forecastNameTaken(config.DB, req.Nama, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create forecast sheet"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama already exists"})
	}
	invoiceIDs, err := resolveForecastInvoices(config.DB, req.InvoiceIDs)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create forecast sheet"})
	}

	sheet := models.ForecastSheet{}
	if email, ok := c.Locals("userEmail").(string); ok {
		sheet.CreatedBy = email
	}
	if err := saveForecastSheet(&sheet, &req, expenses, invoiceIDs); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create forecast sheet"})
	}

	detail, err := loadForecastSheet(config.DB, sheet.ID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load forecast sheet"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"forecast_sheet": detail})
}

// UpdateForecastSheet godoc
// @Summary Update a forecast sheet (admin only)
// @Description Replaces the sheet's fields, expenses and ticked invoices with the request
// @Tags ForecastSheets
// @Security BearerAuth
// @Param id path string true "Forecast sheet ID (UUID)"
// @Param request body models.ForecastSheetRequest true "Forecast sheet"
// @Success 200 {object} map[string]any "Updated"
// @Failure 400 {object} map[string]any "Invalid request"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/forecast-sheets/{id} [put]
func UpdateForecastSheet(c *fiber.Ctx) error {
	var sheet models.ForecastSheet
	if err := config.DB.Where("id = ?", c.Params("id")).First(&sheet).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Forecast sheet not found"})
	}

	var req models.ForecastSheetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	expenses, err := ValidateForecastSheetRequest(&req)
	if err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	taken, err := forecastNameTaken(config.DB, req.Nama, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update forecast sheet"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama already exists"})
	}
	invoiceIDs, err := resolveForecastInvoices(config.DB, req.InvoiceIDs)
	if err != nil {
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{"error": fe.Message})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update forecast sheet"})
	}

	if err := saveForecastSheet(&sheet, &req, expenses, invoiceIDs); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update forecast sheet"})
	}

	detail, err := loadForecastSheet(config.DB, sheet.ID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to load forecast sheet"})
	}
	return c.JSON(fiber.Map{"forecast_sheet": detail})
}

// DeleteForecastSheet godoc
// @Summary Delete a forecast sheet (admin only)
// @Description Its expenses and ticked invoices are removed with it
// @Tags ForecastSheets
// @Security BearerAuth
// @Param id path string true "Forecast sheet ID (UUID)"
// @Success 200 {object} map[string]any "Deleted"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/forecast-sheets/{id} [delete]
func DeleteForecastSheet(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.ForecastSheet{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete forecast sheet"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Forecast sheet not found"})
	}
	return c.JSON(fiber.Map{"message": "Forecast sheet deleted successfully"})
}
