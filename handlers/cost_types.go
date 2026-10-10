package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// GetAllCostTypes godoc
// @Summary Get all cost types
// @Description Retrieve all cost types with optional search and pagination
// @Tags CostTypes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by nama_biaya"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of cost types with pagination"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-types [get]
func GetAllCostTypes(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Order("nama_biaya ASC")
	queryCount := config.DB.Model(&models.CostType{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("nama_biaya ILIKE ?", term)
		queryCount = queryCount.Where("nama_biaya ILIKE ?", term)
	}

	var costTypes []models.CostType
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&costTypes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch cost types"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, costTypes, "cost_types", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetCostType godoc
// @Summary Get a cost type by ID
// @Tags CostTypes
// @Security BearerAuth
// @Param id path string true "Cost type ID (UUID)"
// @Success 200 {object} map[string]any "Cost type details"
// @Failure 404 {object} map[string]any "Cost type not found"
// @Router /api/cost-types/{id} [get]
func GetCostType(c *fiber.Ctx) error {
	var costType models.CostType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&costType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost type not found"})
	}
	return c.JSON(fiber.Map{"cost_type": costType})
}

// CreateCostType godoc
// @Summary Create a new cost type
// @Tags CostTypes
// @Security BearerAuth
// @Param request body models.CostType true "Cost type details"
// @Success 201 {object} models.CostType "Created cost type"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-types [post]
func CreateCostType(c *fiber.Ctx) error {
	var costType models.CostType
	if err := c.BodyParser(&costType); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if costType.NamaBiaya == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nama_biaya is required"})
	}
	if err := config.DB.Create(&costType).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create cost type"})
	}
	return c.Status(fiber.StatusCreated).JSON(costType)
}

// UpdateCostType godoc
// @Summary Update a cost type
// @Tags CostTypes
// @Security BearerAuth
// @Param id path string true "Cost type ID (UUID)"
// @Param request body models.CostType true "Updated cost type details"
// @Success 200 {object} models.CostType "Updated cost type"
// @Failure 404 {object} map[string]any "Cost type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-types/{id} [put]
func UpdateCostType(c *fiber.Ctx) error {
	var costType models.CostType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&costType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost type not found"})
	}
	if err := c.BodyParser(&costType); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&costType).Updates(costType).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update cost type"})
	}
	return c.JSON(costType)
}

// DeleteCostType godoc
// @Summary Delete a cost type
// @Tags CostTypes
// @Security BearerAuth
// @Param id path string true "Cost type ID (UUID)"
// @Success 200 {object} map[string]any "Cost type deleted successfully"
// @Failure 404 {object} map[string]any "Cost type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/cost-types/{id} [delete]
func DeleteCostType(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.CostType{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete cost type"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Cost type not found"})
	}
	return c.JSON(fiber.Map{"message": "Cost type deleted successfully"})
}
