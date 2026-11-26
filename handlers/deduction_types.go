package handlers

import (
	"strings"

	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// deductionTypeNameTaken reports whether another deduction type
// already uses the name (case-insensitive). excludeID may be empty.
func deductionTypeNameTaken(name, excludeID string) (bool, error) {
	var count int64
	q := config.DB.Model(&models.DeductionType{}).Where("LOWER(nama) = LOWER(?)", name)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAllDeductionTypes godoc
// @Summary Get all deduction types
// @Tags DeductionTypes
// @Security BearerAuth
// @Param search query string false "Search by nama"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of deduction types with pagination"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/deduction-types [get]
func GetAllDeductionTypes(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Order("nama ASC")
	queryCount := config.DB.Model(&models.DeductionType{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("nama ILIKE ?", term)
		queryCount = queryCount.Where("nama ILIKE ?", term)
	}

	var items []models.DeductionType
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch deduction types"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, items, "deduction_types", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}
	return c.JSON(response)
}

// GetDeductionType godoc
// @Summary Get a deduction type by ID
// @Tags DeductionTypes
// @Security BearerAuth
// @Param id path string true "Deduction type ID (UUID)"
// @Success 200 {object} map[string]any "Deduction type details"
// @Failure 404 {object} map[string]any "Deduction type not found"
// @Router /api/deduction-types/{id} [get]
func GetDeductionType(c *fiber.Ctx) error {
	var item models.DeductionType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Deduction type not found"})
	}
	return c.JSON(fiber.Map{"deduction_type": item})
}

// CreateDeductionType godoc
// @Summary Create a deduction type (admin only)
// @Tags DeductionTypes
// @Security BearerAuth
// @Param request body models.DeductionType true "Deduction type details"
// @Success 201 {object} models.DeductionType "Created"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/deduction-types [post]
func CreateDeductionType(c *fiber.Ctx) error {
	var item models.DeductionType
	if err := c.BodyParser(&item); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	item.Nama = strings.TrimSpace(item.Nama)
	if item.Nama == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nama is required"})
	}
	taken, err := deductionTypeNameTaken(item.Nama, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create deduction type"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama already exists"})
	}
	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create deduction type"})
	}
	return c.Status(fiber.StatusCreated).JSON(item)
}

// UpdateDeductionType godoc
// @Summary Update a deduction type (admin only)
// @Tags DeductionTypes
// @Security BearerAuth
// @Param id path string true "Deduction type ID (UUID)"
// @Param request body models.DeductionType true "Updated details"
// @Success 200 {object} models.DeductionType "Updated"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 409 {object} map[string]any "Name already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/deduction-types/{id} [put]
func UpdateDeductionType(c *fiber.Ctx) error {
	var item models.DeductionType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&item).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Deduction type not found"})
	}
	if err := c.BodyParser(&item); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	item.Nama = strings.TrimSpace(item.Nama)
	if item.Nama == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "nama is required"})
	}
	taken, err := deductionTypeNameTaken(item.Nama, c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update deduction type"})
	}
	if taken {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "nama already exists"})
	}
	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update deduction type"})
	}
	return c.JSON(item)
}

// DeleteDeductionType godoc
// @Summary Delete a deduction type (admin only)
// @Description Rejected with 409 while invoice deductions still use it
// @Tags DeductionTypes
// @Security BearerAuth
// @Param id path string true "Deduction type ID (UUID)"
// @Success 200 {object} map[string]any "Deleted"
// @Failure 404 {object} map[string]any "Not found"
// @Failure 409 {object} map[string]any "Still in use"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/deduction-types/{id} [delete]
func DeleteDeductionType(c *fiber.Ctx) error {
	var used int64
	if err := config.DB.Model(&models.InvoiceDeduction{}).Where("deduction_type_id = ?", c.Params("id")).Count(&used).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete deduction type"})
	}
	if used > 0 {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Deduction type is still used by invoice deductions"})
	}
	result := config.DB.Delete(&models.DeductionType{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete deduction type"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Deduction type not found"})
	}
	return c.JSON(fiber.Map{"message": "Deduction type deleted successfully"})
}
