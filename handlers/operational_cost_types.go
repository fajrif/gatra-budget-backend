package handlers

import (
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// operationalCostTypeNameTaken reports whether another operational cost type
// already uses the name (case-insensitive). excludeID may be empty.
func operationalCostTypeNameTaken(name, excludeID string) (bool, error) {
	var count int64
	q := config.DB.Model(&models.OperationalCostType{}).Where("LOWER(nama_biaya) = LOWER(?)", name)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAllOperationalCostTypes godoc
// @Summary Get all operational cost types
// @Tags OperationalCostTypes
// @Security BearerAuth
// @Param search query string false "Search by nama_biaya"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of operational cost types with pagination"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/operational-cost-types [get]
func GetAllOperationalCostTypes(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Order("nama_biaya ASC")
	queryCount := config.DB.Model(&models.OperationalCostType{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("nama_biaya ILIKE ?", term)
		queryCount = queryCount.Where("nama_biaya ILIKE ?", term)
	}

	var items []models.OperationalCostType
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch operational cost types"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, items, "operational_cost_types", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}
	return c.JSON(response)
}

// GetOperationalCostType godoc
// @Summary Get an operational cost type by ID
// @Tags OperationalCostTypes
// @Security BearerAuth
// @Param id path string true "Operational cost type ID (UUID)"
// @Success 200 {object} map[string]any "Operational cost type details"
// @Failure 404 {object} map[string]any "Operational cost type not found"
// @Router /api/operational-cost-types/{id} [get]
func GetOperationalCostType(c *fiber.Ctx) error {
	var item models.OperationalCostType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operational cost type not found"})
	}
	return c.JSON(fiber.Map{"operational_cost_type": item})
}

// CreateOperationalCostType godoc
// @Summary Create an operational cost type (admin only)
// @Tags OperationalCostTypes
// @Security BearerAuth
// @Param request body models.OperationalCostType true "Operational cost type details"
// @Success 201 {object} models.OperationalCostType "Created"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/operational-cost-types [post]
func CreateOperationalCostType(c *fiber.Ctx) error {
	var item models.OperationalCostType
	if err := c.BodyParser(&item); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	item.NamaBiaya = strings.TrimSpace(item.NamaBiaya)
	if item.NamaBiaya == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nama_biaya is required"})
	}
	taken, err := operationalCostTypeNameTaken(item.NamaBiaya, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create operational cost type"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama_biaya already exists"})
	}
	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create operational cost type"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateOperationalCostType godoc
// @Summary Update an operational cost type (admin only)
// @Tags OperationalCostTypes
// @Security BearerAuth
// @Param id path string true "Operational cost type ID (UUID)"
// @Param request body models.OperationalCostType true "Updated details"
// @Success 200 {object} models.OperationalCostType "Updated"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/operational-cost-types/{id} [put]
func UpdateOperationalCostType(c *fiber.Ctx) error {
	var item models.OperationalCostType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operational cost type not found"})
	}
	if err := c.BodyParser(&item); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	item.NamaBiaya = strings.TrimSpace(item.NamaBiaya)
	if item.NamaBiaya == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nama_biaya is required"})
	}
	taken, err := operationalCostTypeNameTaken(item.NamaBiaya, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update operational cost type"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama_biaya already exists"})
	}
	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update operational cost type"})
	}
	return c.JSON(item)
}

// DeleteOperationalCostType godoc
// @Summary Delete an operational cost type (admin only)
// @Description Rejected with 409 while cost operations still use it
// @Tags OperationalCostTypes
// @Security BearerAuth
// @Param id path string true "Operational cost type ID (UUID)"
// @Success 200 {object} map[string]any "Deleted"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 409 {object} map[string]any "Still in use"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/operational-cost-types/{id} [delete]
func DeleteOperationalCostType(c *fiber.Ctx) error {
	var used int64
	if err := config.DB.Model(&models.CostOperation{}).Where("operational_cost_type_id = ?", c.Params("id")).Count(&used).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete operational cost type"})
	}
	if used > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Operational cost type is still used by cost operations"})
	}
	result := config.DB.Delete(&models.OperationalCostType{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete operational cost type"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Operational cost type not found"})
	}
	return c.JSON(fiber.Map{"message": "Operational cost type deleted successfully"})
}
