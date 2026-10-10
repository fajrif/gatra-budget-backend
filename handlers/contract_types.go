package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// GetAllContractTypes godoc
// @Summary Get all contract types
// @Description Retrieve all contract types with optional search and pagination
// @Tags ContractTypes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of contract types with pagination"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/contract-types [get]
func GetAllContractTypes(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Order("name ASC")
	queryCount := config.DB.Model(&models.ContractType{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("name ILIKE ?", term)
		queryCount = queryCount.Where("name ILIKE ?", term)
	}

	var contractTypes []models.ContractType
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&contractTypes).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch contract types"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, contractTypes, "contract_types", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetContractType godoc
// @Summary Get a contract type by ID
// @Tags ContractTypes
// @Security BearerAuth
// @Param id path string true "Contract type ID (UUID)"
// @Success 200 {object} map[string]any "Contract type details"
// @Failure 404 {object} map[string]any "Contract type not found"
// @Router /api/contract-types/{id} [get]
func GetContractType(c *fiber.Ctx) error {
	var contractType models.ContractType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&contractType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contract type not found"})
	}
	return c.JSON(fiber.Map{"contract_type": contractType})
}

// CreateContractType godoc
// @Summary Create a new contract type
// @Tags ContractTypes
// @Security BearerAuth
// @Param request body models.ContractType true "Contract type details"
// @Success 201 {object} models.ContractType "Created contract type"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/contract-types [post]
func CreateContractType(c *fiber.Ctx) error {
	var contractType models.ContractType
	if err := c.BodyParser(&contractType); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if contractType.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
	}
	if err := config.DB.Create(&contractType).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create contract type"})
	}
	return c.Status(fiber.StatusCreated).JSON(contractType)
}

// UpdateContractType godoc
// @Summary Update a contract type
// @Tags ContractTypes
// @Security BearerAuth
// @Param id path string true "Contract type ID (UUID)"
// @Param request body models.ContractType true "Updated contract type details"
// @Success 200 {object} models.ContractType "Updated contract type"
// @Failure 404 {object} map[string]any "Contract type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/contract-types/{id} [put]
func UpdateContractType(c *fiber.Ctx) error {
	var contractType models.ContractType
	if err := config.DB.Where("id = ?", c.Params("id")).First(&contractType).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contract type not found"})
	}
	if err := c.BodyParser(&contractType); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&contractType).Updates(contractType).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update contract type"})
	}
	return c.JSON(contractType)
}

// DeleteContractType godoc
// @Summary Delete a contract type
// @Tags ContractTypes
// @Security BearerAuth
// @Param id path string true "Contract type ID (UUID)"
// @Success 200 {object} map[string]any "Contract type deleted successfully"
// @Failure 404 {object} map[string]any "Contract type not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/contract-types/{id} [delete]
func DeleteContractType(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.ContractType{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete contract type"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Contract type not found"})
	}
	return c.JSON(fiber.Map{"message": "Contract type deleted successfully"})
}
