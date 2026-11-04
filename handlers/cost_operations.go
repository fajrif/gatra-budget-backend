package handlers

import (
	"regexp"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

var monthQueryPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// ValidateCostOperationRequest validates the cost operation request fields.
func ValidateCostOperationRequest(req *models.CostOperationRequest) error {
	if req.OperationalCostTypeID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "operational_cost_type_id is required")
	}
	if req.TanggalTransaksi == "" {
		return fiber.NewError(fiber.StatusBadRequest, "tanggal_transaksi is required")
	}
	if req.Jumlah <= 100_000 {
		return fiber.NewError(fiber.StatusBadRequest, "jumlah is required and must be greater than 100,000")
	}
	return nil
}

// applyCostOperationRequest copies a validated request onto a CostOperation,
// parsing the date and recomputing bulan_realisasi server-side.
func applyCostOperationRequest(o *models.CostOperation, req *models.CostOperationRequest) error {
	tanggal, err := helpers.ParseDateString(&req.TanggalTransaksi)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tanggal_transaksi: "+err.Error())
	}

	o.OperationalCostTypeID = helpers.ParseUUID(req.OperationalCostTypeID)
	o.Deskripsi = req.Deskripsi
	o.Jumlah = req.Jumlah
	o.TanggalTransaksi = models.Date{Time: *tanggal}

	bulan := helpers.ComputeBulanRealisasi(o.TanggalTransaksi)
	o.BulanRealisasi = &bulan
	return nil
}

// GetAllCostOperations godoc
// @Summary Get all operational expenses
// @Tags CostOperations
// @Security BearerAuth
// @Param operational_cost_type_id query string false "Filter by operational cost type ID (UUID)"
// @Param month query string false "Filter by month (YYYY-MM)"
// @Param search query string false "Search by deskripsi"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of cost operations with pagination"
// @Failure 400 {object} map[string]any "Invalid month"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-operations [get]
func GetAllCostOperations(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("OperationalCostType").Order("tanggal_transaksi DESC")
	queryCount := config.DB.Model(&models.CostOperation{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if typeID := c.Query("operational_cost_type_id"); typeID != "" {
		query = query.Where("operational_cost_type_id = ?", typeID)
		queryCount = queryCount.Where("operational_cost_type_id = ?", typeID)
	}

	if month := c.Query("month"); month != "" {
		if !monthQueryPattern.MatchString(month) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "month must be in YYYY-MM format"})
		}
		query = query.Where("bulan_realisasi = ?", month)
		queryCount = queryCount.Where("bulan_realisasi = ?", month)
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("deskripsi ILIKE ?", term)
		queryCount = queryCount.Where("deskripsi ILIKE ?", term)
	}

	var items []models.CostOperation
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch cost operations"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, items, "cost_operations", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}
	return c.JSON(response)
}

// GetCostOperation godoc
// @Summary Get an operational expense by ID
// @Tags CostOperations
// @Security BearerAuth
// @Param id path string true "Cost operation ID (UUID)"
// @Success 200 {object} map[string]any "Cost operation details"
// @Failure 404 {object} map[string]any "Cost operation not found"
// @Router /api/cost-operations/{id} [get]
func GetCostOperation(c *fiber.Ctx) error {
	var item models.CostOperation
	if err := config.DB.Preload("OperationalCostType").Where("id = ?", c.Params("id")).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost operation not found"})
	}
	return c.JSON(fiber.Map{"cost_operation": item})
}

// CreateCostOperation godoc
// @Summary Create an operational expense
// @Description bulan_realisasi is computed server-side from tanggal_transaksi
// @Tags CostOperations
// @Security BearerAuth
// @Param request body models.CostOperationRequest true "Cost operation details"
// @Success 201 {object} models.CostOperation "Created"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Operational cost type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-operations [post]
func CreateCostOperation(c *fiber.Ctx) error {
	var req models.CostOperationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateCostOperationRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var costType models.OperationalCostType
	if err := config.DB.Where("id = ?", req.OperationalCostTypeID).First(&costType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operational cost type not found"})
	}

	var item models.CostOperation
	if err := applyCostOperationRequest(&item, &req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	if email, ok := c.Locals("userEmail").(string); ok {
		item.CreatedBy = email
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create cost operation"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateCostOperation godoc
// @Summary Update an operational expense
// @Tags CostOperations
// @Security BearerAuth
// @Param id path string true "Cost operation ID (UUID)"
// @Param request body models.CostOperationRequest true "Updated details"
// @Success 200 {object} models.CostOperation "Updated"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-operations/{id} [put]
func UpdateCostOperation(c *fiber.Ctx) error {
	var item models.CostOperation
	if err := config.DB.Where("id = ?", c.Params("id")).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost operation not found"})
	}

	var req models.CostOperationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := ValidateCostOperationRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}

	var costType models.OperationalCostType
	if err := config.DB.Where("id = ?", req.OperationalCostTypeID).First(&costType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operational cost type not found"})
	}

	if err := applyCostOperationRequest(&item, &req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{"error": err.Error()})
	}
	item.OperationalCostType = nil

	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update cost operation"})
	}
	return c.JSON(item)
}

// DeleteCostOperation godoc
// @Summary Delete an operational expense
// @Tags CostOperations
// @Security BearerAuth
// @Param id path string true "Cost operation ID (UUID)"
// @Success 200 {object} map[string]any "Deleted"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-operations/{id} [delete]
func DeleteCostOperation(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.CostOperation{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete cost operation"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost operation not found"})
	}
	return c.JSON(fiber.Map{"message": "Cost operation deleted successfully"})
}
