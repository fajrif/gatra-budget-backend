package handlers

import (
	"boilerplates-backend/config"
	"boilerplates-backend/helpers"
	"boilerplates-backend/models"

	"github.com/gofiber/fiber/v2"
)

// GetAllCities godoc
// @Summary Get all cities
// @Description Retrieve all cities with optional province_id filter, search, and pagination
// @Tags Cities
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param province_id query string false "Filter by province ID (UUID)"
// @Param search query string false "Search by name"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]interface{} "List of cities with pagination"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/cities [get]
func GetAllCities(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("Province").Preload("Province.Country").Order("name ASC")
	queryCount := config.DB.Model(&models.City{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if provinceID := c.Query("province_id"); provinceID != "" {
		query = query.Where("province_id = ?", provinceID)
		queryCount = queryCount.Where("province_id = ?", provinceID)
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("name ILIKE ?", term)
		queryCount = queryCount.Where("name ILIKE ?", term)
	}

	var cities []models.City
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&cities).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch cities"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, cities, "cities", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetCity godoc
// @Summary Get a city by ID
// @Tags Cities
// @Security BearerAuth
// @Param id path string true "City ID (UUID)"
// @Success 200 {object} map[string]interface{} "City details"
// @Failure 404 {object} map[string]interface{} "City not found"
// @Router /api/cities/{id} [get]
func GetCity(c *fiber.Ctx) error {
	var city models.City
	if err := config.DB.Preload("Province").Preload("Province.Country").Where("id = ?", c.Params("id")).First(&city).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "City not found"})
	}
	return c.JSON(fiber.Map{"city": city})
}

// CreateCity godoc
// @Summary Create a new city
// @Tags Cities
// @Security BearerAuth
// @Param request body models.City true "City details"
// @Success 201 {object} models.City "Created city"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/cities [post]
func CreateCity(c *fiber.Ctx) error {
	var city models.City
	if err := c.BodyParser(&city); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if city.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
	}
	if city.ProvinceID.String() == "00000000-0000-0000-0000-000000000000" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "province_id is required"})
	}
	if err := config.DB.Create(&city).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create city"})
	}
	return c.Status(fiber.StatusCreated).JSON(city)
}

// UpdateCity godoc
// @Summary Update a city
// @Tags Cities
// @Security BearerAuth
// @Param id path string true "City ID (UUID)"
// @Param request body models.City true "Updated city details"
// @Success 200 {object} models.City "Updated city"
// @Failure 404 {object} map[string]interface{} "City not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/cities/{id} [put]
func UpdateCity(c *fiber.Ctx) error {
	var city models.City
	if err := config.DB.Where("id = ?", c.Params("id")).First(&city).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "City not found"})
	}
	if err := c.BodyParser(&city); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&city).Updates(city).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update city"})
	}
	return c.JSON(city)
}

// DeleteCity godoc
// @Summary Delete a city
// @Tags Cities
// @Security BearerAuth
// @Param id path string true "City ID (UUID)"
// @Success 200 {object} map[string]interface{} "City deleted successfully"
// @Failure 404 {object} map[string]interface{} "City not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/cities/{id} [delete]
func DeleteCity(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.City{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete city"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "City not found"})
	}
	return c.JSON(fiber.Map{"message": "City deleted successfully"})
}
