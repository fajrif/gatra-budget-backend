package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// ValidateTransactionRequest validates the transaction request fields
func ValidateTransactionRequest(req *models.TransactionRequest) error {
	if req.ProjectID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "project_id is required")
	}
	if req.CostTypeID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "cost_type_id is required")
	}
	if req.TanggalTransaksi == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tanggal_transaksi is required")
	}
	if req.JumlahRealisasi <= 100_000 {
		return fiber.NewError(fiber.StatusBadRequest, "jumlah_realisasi is required and must be greater than 100,000")
	}
	return nil
}

// applyTransactionRequest copies a validated TransactionRequest onto a
// Transaction, parsing the date and recomputing bulan_realisasi server-side.
func applyTransactionRequest(t *models.Transaction, req *models.TransactionRequest, project *models.Project) error {
	tanggalTransaksi, err := helpers.ParseDateString(&req.TanggalTransaksi)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tanggal_transaksi: "+err.Error())
	}

	t.ProjectID = project.ID
	t.CostTypeID = helpers.ParseUUID(req.CostTypeID)
	t.DeskripsiRealisasi = req.DeskripsiRealisasi
	t.JumlahRealisasi = req.JumlahRealisasi
	t.JumlahTenagaKerja = req.JumlahTenagaKerja

	t.TanggalTransaksi = models.Date{Time: *tanggalTransaksi}

	bulanRealisasi := helpers.ComputeBulanRealisasi(t.TanggalTransaksi)
	t.BulanRealisasi = &bulanRealisasi

	return nil
}

// GetAllTransactions godoc
// @Summary Get all transactions
// @Description Retrieve transactions, optionally filtered by project or cost type, with pagination
// @Tags Transactions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param project_id query string false "Filter by project ID (UUID)"
// @Param cost_type_id query string false "Filter by cost type ID (UUID)"
// @Param search query string false "Search by deskripsi_realisasi"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of transactions with pagination"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/transactions [get]
func GetAllTransactions(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("CostType").Preload("Project").Order("tanggal_transaksi DESC")
	queryCount := config.DB.Model(&models.Transaction{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
		queryCount = queryCount.Where("project_id = ?", projectID)
	}

	if costTypeID := c.Query("cost_type_id"); costTypeID != "" {
		query = query.Where("cost_type_id = ?", costTypeID)
		queryCount = queryCount.Where("cost_type_id = ?", costTypeID)
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("deskripsi_realisasi ILIKE ?", term)
		queryCount = queryCount.Where("deskripsi_realisasi ILIKE ?", term)
	}

	var transactions []models.Transaction
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&transactions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch transactions"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, transactions, "transactions", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetTransaction godoc
// @Summary Get a transaction by ID
// @Tags Transactions
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Success 200 {object} map[string]any "Transaction details"
// @Failure 404 {object} map[string]any "Transaction not found"
// @Router /api/transactions/{id} [get]
func GetTransaction(c *fiber.Ctx) error {
	var transaction models.Transaction
	if err := config.DB.Preload("CostType").Preload("Project").Where("id = ?", c.Params("id")).First(&transaction).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Transaction not found"})
	}
	return c.JSON(fiber.Map{"transaction": transaction})
}

// CreateTransaction godoc
// @Summary Create a new transaction
// @Description Creates a transaction; bulan_realisasi is computed server-side from tanggal_transaksi
// @Tags Transactions
// @Security BearerAuth
// @Param request body models.TransactionRequest true "Transaction details"
// @Success 201 {object} models.Transaction "Created transaction"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Project or cost type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/transactions [post]
func CreateTransaction(c *fiber.Ctx) error {
	var req models.TransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateTransactionRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var project models.Project
	if err := config.DB.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	var costType models.CostType
	if err := config.DB.Where("id = ?", req.CostTypeID).First(&costType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost type not found"})
	}

	var transaction models.Transaction
	if err := applyTransactionRequest(&transaction, &req, &project); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	if email, ok := c.Locals("userEmail").(string); ok {
		transaction.CreatedBy = email
	}

	if err := config.DB.Create(&transaction).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create transaction"})
	}
	return c.Status(fiber.StatusCreated).JSON(transaction)
}

// UpdateTransaction godoc
// @Summary Update a transaction
// @Tags Transactions
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Param request body models.TransactionRequest true "Updated transaction details"
// @Success 200 {object} models.Transaction "Updated transaction"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Transaction not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/transactions/{id} [put]
func UpdateTransaction(c *fiber.Ctx) error {
	var transaction models.Transaction
	if err := config.DB.Where("id = ?", c.Params("id")).First(&transaction).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Transaction not found"})
	}

	var req models.TransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateTransactionRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var project models.Project
	if err := config.DB.Where("id = ?", req.ProjectID).First(&project).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
	}

	var costType models.CostType
	if err := config.DB.Where("id = ?", req.CostTypeID).First(&costType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost type not found"})
	}

	if err := applyTransactionRequest(&transaction, &req, &project); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	if err := config.DB.Save(&transaction).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update transaction"})
	}
	return c.JSON(transaction)
}

// DeleteTransaction godoc
// @Summary Delete a transaction
// @Tags Transactions
// @Security BearerAuth
// @Param id path string true "Transaction ID (UUID)"
// @Success 200 {object} map[string]any "Transaction deleted successfully"
// @Failure 404 {object} map[string]any "Transaction not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/transactions/{id} [delete]
func DeleteTransaction(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.Transaction{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete transaction"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Transaction not found"})
	}
	return c.JSON(fiber.Map{"message": "Transaction deleted successfully"})
}
