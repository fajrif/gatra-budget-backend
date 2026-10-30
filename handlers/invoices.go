package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// ValidateInvoiceRequest validates the invoice request fields. The tgl_bayar
// rule mirrors the database CHECK: a Clearing (paid) invoice must have a
// payment date, and a Submit invoice must not.
func ValidateInvoiceRequest(req *models.InvoiceRequest) error {
	if req.NoInvoice == "" {
		return fiber.NewError(fiber.StatusBadRequest, "no_invoice is required")
	}
	if req.ProjectID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id is required")
	}
	if req.NamaInvoice == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nama_invoice is required")
	}
	if req.TglTagihan == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tgl_tagihan is required")
	}
	if req.Nilai <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "nilai must be greater than 0")
	}
	if req.Status != models.InvoiceStatusSubmit && req.Status != models.InvoiceStatusClearing {
		return fiber.NewError(fiber.StatusBadRequest, "status must be 'Submit' or 'Clearing'")
	}

	hasTglBayar := req.TglBayar != nil && *req.TglBayar != ""
	if req.Status == models.InvoiceStatusClearing && !hasTglBayar {
		return fiber.NewError(fiber.StatusBadRequest, "tgl_bayar is required when status is Clearing")
	}
	if req.Status == models.InvoiceStatusSubmit && hasTglBayar {
		return fiber.NewError(fiber.StatusBadRequest, "tgl_bayar must be empty when status is Submit")
	}
	if hasTglBayar && *req.TglBayar <= req.TglTagihan {
		return fiber.NewError(fiber.StatusBadRequest, "tgl_bayar harus setelah tgl_tagihan")
	}

	if req.ManajemenFee != nil {
		if *req.ManajemenFee < 0 {
			return fiber.NewError(fiber.StatusBadRequest, "manajemen_fee must not be negative")
		}
		if *req.ManajemenFee > req.Nilai {
			return fiber.NewError(fiber.StatusBadRequest, "manajemen_fee tidak boleh melebihi nilai")
		}
	}
	return nil
}

// applyInvoiceRequest copies a validated InvoiceRequest onto an Invoice,
// parsing the dates. CreatedBy is set by the caller on create only.
func applyInvoiceRequest(inv *models.Invoice, req *models.InvoiceRequest) error {
	tglTagihan, err := helpers.ParseDateString(&req.TglTagihan)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tgl_tagihan: "+err.Error())
	}
	tglBayar, err := helpers.ParseDateString(req.TglBayar)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tgl_bayar: "+err.Error())
	}

	inv.NoInvoice = req.NoInvoice
	inv.ProjectID = helpers.ParseUUID(req.ProjectID)
	inv.NamaInvoice = req.NamaInvoice
	inv.Status = req.Status
	inv.Nilai = req.Nilai
	inv.Description = req.Description
	inv.ManajemenFee = 0
	if req.ManajemenFee != nil {
		inv.ManajemenFee = *req.ManajemenFee
	}

	inv.TglTagihan = models.Date{Time: *tglTagihan}
	if tglBayar != nil {
		inv.TglBayar = &models.Date{Time: *tglBayar}
	} else {
		inv.TglBayar = nil
	}
	return nil
}

// GetAllInvoices godoc
// @Summary Get all invoices
// @Description Retrieve invoices, optionally filtered by project, status or billing month, with pagination
// @Tags Invoices
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id query string false "Filter by project ID (UUID)"
// @Param status query string false "Filter by status (Submit, Clearing)"
// @Param month query string false "Filter by tgl_tagihan month (YYYY-MM)"
// @Param search query string false "Search by no_invoice or nama_invoice"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of invoices with pagination"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/invoices [get]
func GetAllInvoices(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("Project").Order("tgl_tagihan DESC, created_at DESC")
	queryCount := config.DB.Model(&models.Invoice{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
		queryCount = queryCount.Where("project_id = ?", projectID)
	}

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
		queryCount = queryCount.Where("status = ?", status)
	}

	if month := c.Query("month"); month != "" {
		query = query.Where("to_char(tgl_tagihan, 'YYYY-MM') = ?", month)
		queryCount = queryCount.Where("to_char(tgl_tagihan, 'YYYY-MM') = ?", month)
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		cond := "no_invoice ILIKE ? OR nama_invoice ILIKE ?"
		query = query.Where(cond, term, term)
		queryCount = queryCount.Where(cond, term, term)
	}

	var invoices []models.Invoice
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&invoices).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch invoices"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, invoices, "invoices", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetInvoice godoc
// @Summary Get an invoice by ID
// @Tags Invoices
// @Security BearerAuth
// @Param id path string true "Invoice ID (UUID)"
// @Success 200 {object} map[string]any "Invoice details"
// @Failure 404 {object} map[string]any "Invoice not found"
// @Router /api/invoices/{id} [get]
func GetInvoice(c *fiber.Ctx) error {
	var invoice models.Invoice
	if err := config.DB.Preload("Project").Where("id = ?", c.Params("id")).First(&invoice).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
	}
	return c.JSON(fiber.Map{"invoice": invoice})
}

// CreateInvoice godoc
// @Summary Create a new invoice
// @Tags Invoices
// @Security BearerAuth
// @Param request body models.InvoiceRequest true "Invoice details"
// @Success 201 {object} models.Invoice "Created invoice"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Project not found"
// @Failure 409 {object} map[string]any "no_invoice already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/invoices [post]
func CreateInvoice(c *fiber.Ctx) error {
	var req models.InvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateInvoiceRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var project models.Project
	if err := config.DB.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	var existing models.Invoice
	if err := config.DB.Where("no_invoice = ?", req.NoInvoice).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "no_invoice already exists"})
	}

	var invoice models.Invoice
	if err := applyInvoiceRequest(&invoice, &req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	if email, ok := c.Locals("userEmail").(string); ok {
		invoice.CreatedBy = email
	}

	if err := config.DB.Create(&invoice).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create invoice"})
	}
	return c.Status(fiber.StatusCreated).JSON(invoice)
}

// UpdateInvoice godoc
// @Summary Update an invoice
// @Tags Invoices
// @Security BearerAuth
// @Param id path string true "Invoice ID (UUID)"
// @Param request body models.InvoiceRequest true "Updated invoice details"
// @Success 200 {object} models.Invoice "Updated invoice"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Invoice or project not found"
// @Failure 409 {object} map[string]any "no_invoice already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/invoices/{id} [put]
func UpdateInvoice(c *fiber.Ctx) error {
	var invoice models.Invoice
	if err := config.DB.Where("id = ?", c.Params("id")).First(&invoice).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
	}

	var req models.InvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateInvoiceRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var project models.Project
	if err := config.DB.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	var existing models.Invoice
	if err := config.DB.Where("no_invoice = ? AND id <> ?", req.NoInvoice, invoice.ID).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "no_invoice already exists"})
	}

	if err := applyInvoiceRequest(&invoice, &req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	if err := config.DB.Save(&invoice).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update invoice"})
	}
	return c.JSON(invoice)
}

// DeleteInvoice godoc
// @Summary Delete an invoice
// @Tags Invoices
// @Security BearerAuth
// @Param id path string true "Invoice ID (UUID)"
// @Success 200 {object} map[string]any "Invoice deleted successfully"
// @Failure 404 {object} map[string]any "Invoice not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/invoices/{id} [delete]
func DeleteInvoice(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.Invoice{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete invoice"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Invoice not found"})
	}
	return c.JSON(fiber.Map{"message": "Invoice deleted successfully"})
}
