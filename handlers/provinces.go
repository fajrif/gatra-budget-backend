package handlers

import (
	"boilerplates-backend/config"
	"boilerplates-backend/helpers"
	"boilerplates-backend/models"

	"github.com/gofiber/fiber/v2"
)

// GetAllProvinces godoc
// @Summary Get all provinces
// @Description Retrieve all provinces with optional country_id filter, search, and pagination
// @Tags Provinces
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param country_id query string false "Filter by country ID (UUID)"
// @Param search query string false "Search by name"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]interface{} "List of provinces with pagination"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/provinces [get]
func GetAllProvinces(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("Country").Order("name ASC")
	queryCount := config.DB.Model(&models.Province{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if countryID := c.Query("country_id"); countryID != "" {
		query = query.Where("country_id = ?", countryID)
		queryCount = queryCount.Where("country_id = ?", countryID)
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("name ILIKE ?", term)
		queryCount = queryCount.Where("name ILIKE ?", term)
	}

	var provinces []models.Province
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&provinces).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch provinces"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, provinces, "provinces", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetProvince godoc
// @Summary Get a province by ID
// @Tags Provinces
// @Security BearerAuth
// @Param id path string true "Province ID (UUID)"
// @Success 200 {object} map[string]interface{} "Province details"
// @Failure 404 {object} map[string]interface{} "Province not found"
// @Router /api/provinces/{id} [get]
func GetProvince(c *fiber.Ctx) error {
	var province models.Province
	if err := config.DB.Preload("Country").Where("id = ?", c.Params("id")).First(&province).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Province not found"})
	}
	return c.JSON(fiber.Map{"province": province})
}

// CreateProvince godoc
// @Summary Create a new province
// @Tags Provinces
// @Security BearerAuth
// @Param request body models.Province true "Province details"
// @Success 201 {object} models.Province "Created province"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/provinces [post]
func CreateProvince(c *fiber.Ctx) error {
	var province models.Province
	if err := c.BodyParser(&province); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if province.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
	}
	if province.CountryID.String() == "00000000-0000-0000-0000-000000000000" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "country_id is required"})
	}
	if err := config.DB.Create(&province).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create province"})
	}
	return c.Status(fiber.StatusCreated).JSON(province)
}

// UpdateProvince godoc
// @Summary Update a province
// @Tags Provinces
// @Security BearerAuth
// @Param id path string true "Province ID (UUID)"
// @Param request body models.Province true "Updated province details"
// @Success 200 {object} models.Province "Updated province"
// @Failure 404 {object} map[string]interface{} "Province not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/provinces/{id} [put]
func UpdateProvince(c *fiber.Ctx) error {
	var province models.Province
	if err := config.DB.Where("id = ?", c.Params("id")).First(&province).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Province not found"})
	}
	if err := c.BodyParser(&province); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&province).Updates(province).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update province"})
	}
	return c.JSON(province)
}

// DeleteProvince godoc
// @Summary Delete a province
// @Tags Provinces
// @Security BearerAuth
// @Param id path string true "Province ID (UUID)"
// @Success 200 {object} map[string]interface{} "Province deleted successfully"
// @Failure 404 {object} map[string]interface{} "Province not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/provinces/{id} [delete]
func DeleteProvince(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.Province{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete province"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Province not found"})
	}
	return c.JSON(fiber.Map{"message": "Province deleted successfully"})
}
