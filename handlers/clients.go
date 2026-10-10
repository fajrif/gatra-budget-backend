package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
)

// GetAllClients godoc
// @Summary Get all clients
// @Description Retrieve all clients with optional search and pagination
// @Tags Clients
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Success 200 {object} map[string]any "List of clients with pagination"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/clients [get]
func GetAllClients(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Order("name ASC")
	queryCount := config.DB.Model(&models.Client{})

	if c.Query("all") == "true" {
		pagination.Limit = -1
		pagination.Offset = 0
	}

	if search := c.Query("search"); search != "" {
		term := "%" + search + "%"
		query = query.Where("name ILIKE ?", term)
		queryCount = queryCount.Where("name ILIKE ?", term)
	}

	var clients []models.Client
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&clients).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch clients"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, clients, "clients", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetClient godoc
// @Summary Get a client by ID
// @Tags Clients
// @Security BearerAuth
// @Param id path string true "Client ID (UUID)"
// @Success 200 {object} map[string]any "Client details"
// @Failure 404 {object} map[string]any "Client not found"
// @Router /api/clients/{id} [get]
func GetClient(c *fiber.Ctx) error {
	var client models.Client
	if err := config.DB.Where("id = ?", c.Params("id")).First(&client).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Client not found"})
	}
	return c.JSON(fiber.Map{"client": client})
}

// CreateClient godoc
// @Summary Create a new client
// @Tags Clients
// @Security BearerAuth
// @Param request body models.Client true "Client details"
// @Success 201 {object} models.Client "Created client"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/clients [post]
func CreateClient(c *fiber.Ctx) error {
	var client models.Client
	if err := c.BodyParser(&client); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if client.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name is required"})
	}
	if err := config.DB.Create(&client).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create client"})
	}
	return c.Status(fiber.StatusCreated).JSON(client)
}

// UpdateClient godoc
// @Summary Update a client
// @Tags Clients
// @Security BearerAuth
// @Param id path string true "Client ID (UUID)"
// @Param request body models.Client true "Updated client details"
// @Success 200 {object} models.Client "Updated client"
// @Failure 404 {object} map[string]any "Client not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/clients/{id} [put]
func UpdateClient(c *fiber.Ctx) error {
	var client models.Client
	if err := config.DB.Where("id = ?", c.Params("id")).First(&client).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Client not found"})
	}
	if err := c.BodyParser(&client); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&client).Updates(client).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update client"})
	}
	return c.JSON(client)
}

// DeleteClient godoc
// @Summary Delete a client
// @Tags Clients
// @Security BearerAuth
// @Param id path string true "Client ID (UUID)"
// @Success 200 {object} map[string]any "Client deleted successfully"
// @Failure 404 {object} map[string]any "Client not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/clients/{id} [delete]
func DeleteClient(c *fiber.Ctx) error {
	result := config.DB.Delete(&models.Client{}, "id = ?", c.Params("id"))
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete client"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Client not found"})
	}
	return c.JSON(fiber.Map{"message": "Client deleted successfully"})
}
