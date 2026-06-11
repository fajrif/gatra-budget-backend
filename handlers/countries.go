package handlers

import (
	"boilerplates-backend/config"
	"boilerplates-backend/helpers"
	"boilerplates-backend/models"

	"github.com/gofiber/fiber/v2"
)

// GetAllCountries godoc
// @Summary Get all countries
// @Description Retrieve all countries with optional search and pagination
// @Tags Countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name or code"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]interface{} "List of countries with pagination"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/countries [get]
func GetAllCountries(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Order("name ASC")
	queryCount := config.DB.Model(&models.Country{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		cond := "name ILIKE ? OR code ILIKE ?"
		query = query.Where(cond, term, term)
		queryCount = queryCount.Where(cond, term, term)
	}

	var countries []models.Country
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&countries).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch countries"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, countries, "countries", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetCountry godoc
// @Summary Get a country by ID
// @Description Retrieve a single country by its UUID
// @Tags Countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Country ID (UUID)"
// @Success 200 {object} map[string]interface{} "Country details"
// @Failure 404 {object} map[string]interface{} "Country not found"
// @Router /api/countries/{id} [get]
func GetCountry(c *fiber.Ctx) error {
	var country models.Country
	if err := config.DB.Where("id = ?", c.Params("id")).First(&country).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Country not found"})
	}
	return c.JSON(fiber.Map{"country": country})
}

// CreateCountry godoc
// @Summary Create a new country
// @Description Create a new country entry
// @Tags Countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.Country true "Country details"
// @Success 201 {object} models.Country "Created country"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/countries [post]
func CreateCountry(c *fiber.Ctx) error {
	var country models.Country
	if err := c.BodyParser(&country); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if country.Name == "" || country.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and code are required"})
	}
	if err := config.DB.Create(&country).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create country"})
	}
	return c.Status(fiber.StatusCreated).JSON(country)
}

// UpdateCountry godoc
// @Summary Update a country
// @Description Update an existing country by ID
// @Tags Countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Country ID (UUID)"
// @Param request body models.Country true "Updated country details"
// @Success 200 {object} models.Country "Updated country"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 404 {object} map[string]interface{} "Country not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/countries/{id} [put]
func UpdateCountry(c *fiber.Ctx) error {
	var country models.Country
	if err := config.DB.Where("id = ?", c.Params("id")).First(&country).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Country not found"})
	}
	if err := c.BodyParser(&country); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&country).Updates(country).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update country"})
	}
	return c.JSON(country)
}

// DeleteCountry godoc
// @Summary Delete a country
// @Description Delete a country by ID
// @Tags Countries
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Country ID (UUID)"
// @Success 200 {object} map[string]interface{} "Country deleted successfully"
// @Failure 404 {object} map[string]interface{} "Country not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/countries/{id} [delete]
func DeleteCountry(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.Country{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete country"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Country not found"})
	}
	return c.JSON(fiber.Map{"message": "Country deleted successfully"})
}
