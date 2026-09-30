package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// GetAllAreas godoc
// @Summary Get all areas
// @Description Retrieve all areas with optional city_id filter, search, and pagination
// @Tags Areas
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param city_id query string false "Filter by city ID (UUID)"
// @Param search query string false "Search by name"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of areas with pagination"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/areas [get]
func GetAllAreas(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("City").Order("name ASC")
	queryCount := config.DB.Model(&models.Area{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if cityID := c.Query("city_id"); cityID != "" {
		query = query.Where("city_id = ?", cityID)
		queryCount = queryCount.Where("city_id = ?", cityID)
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("name ILIKE ?", term)
		queryCount = queryCount.Where("name ILIKE ?", term)
	}

	var areas []models.Area
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&areas).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch areas"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, areas, "areas", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetArea godoc
// @Summary Get an area by ID
// @Tags Areas
// @Security BearerAuth
// @Param id path string true "Area ID (UUID)"
// @Success 200 {object} map[string]any "Area details"
// @Failure 404 {object} map[string]any "Area not found"
// @Router /api/areas/{id} [get]
func GetArea(c *fiber.Ctx) error {
	var area models.Area
	if err := config.DB.Preload("City").Where("id = ?", c.Params("id")).First(&area).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Area not found"})
	}
	return c.JSON(fiber.Map{"area": area})
}

// CreateArea godoc
// @Summary Create a new area
// @Tags Areas
// @Security BearerAuth
// @Param request body models.Area true "Area details"
// @Success 201 {object} models.Area "Created area"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/areas [post]
func CreateArea(c *fiber.Ctx) error {
	var area models.Area
	if err := c.BodyParser(&area); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if area.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
	}
	if area.CityID.String() == "00000000-0000-0000-0000-000000000000" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "city_id is required"})
	}
	if err := config.DB.Create(&area).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create area"})
	}
	return c.Status(fiber.StatusCreated).JSON(area)
}

// UpdateArea godoc
// @Summary Update an area
// @Tags Areas
// @Security BearerAuth
// @Param id path string true "Area ID (UUID)"
// @Param request body models.Area true "Updated area details"
// @Success 200 {object} models.Area "Updated area"
// @Failure 404 {object} map[string]any "Area not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/areas/{id} [put]
func UpdateArea(c *fiber.Ctx) error {
	var area models.Area
	if err := config.DB.Where("id = ?", c.Params("id")).First(&area).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Area not found"})
	}
	if err := c.BodyParser(&area); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&area).Updates(area).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update area"})
	}
	return c.JSON(area)
}

// DeleteArea godoc
// @Summary Delete an area
// @Tags Areas
// @Security BearerAuth
// @Param id path string true "Area ID (UUID)"
// @Success 200 {object} map[string]any "Area deleted successfully"
// @Failure 404 {object} map[string]any "Area not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/areas/{id} [delete]
func DeleteArea(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.Area{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete area"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Area not found"})
	}
	return c.JSON(fiber.Map{"message": "Area deleted successfully"})
}
