package handlers

import (
	"boilerplates-backend/config"
	"boilerplates-backend/helpers"
	"boilerplates-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetMyAddresses godoc
// @Summary Get current user's addresses
// @Description Get all addresses belonging to the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Success 200 {object} map[string]interface{} "List of addresses"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/me/addresses [get]
func GetMyAddresses(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	return getAddressesByUser(c, userID)
}

// GetUserAddresses godoc
// @Summary Get addresses by user ID (Admin only)
// @Description Get all addresses for a specific user. Only accessible by admin users.
// @Tags Addresses
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user_id path string true "User ID (UUID)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Success 200 {object} map[string]interface{} "List of addresses"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/users/{user_id}/addresses [get]
func GetUserAddresses(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if _, err := uuid.Parse(userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	return getAddressesByUser(c, userID)
}

func getAddressesByUser(c *fiber.Ctx, userID string) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Preload("Country").Preload("Province").Preload("City").Preload("Area").
		Where("user_id = ?", userID).Order("created_at DESC")
	queryCount := config.DB.Model(&models.Address{}).Where("user_id = ?", userID)

	var addresses []models.Address
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Find(&addresses).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch addresses"})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, addresses, "addresses", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create pagination response"})
	}

	return c.JSON(response)
}

// GetMyAddress godoc
// @Summary Get a specific address of the current user
// @Tags Addresses
// @Security BearerAuth
// @Param id path string true "Address ID (UUID)"
// @Success 200 {object} map[string]interface{} "Address details"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Router /api/me/addresses/{id} [get]
func GetMyAddress(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)
	var address models.Address
	if err := config.DB.Preload("Country").Preload("Province").Preload("City").Preload("Area").
		Where("id = ? AND user_id = ?", c.Params("id"), userID).First(&address).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Address not found"})
	}
	return c.JSON(fiber.Map{"address": address})
}

// CreateMyAddress godoc
// @Summary Create a new address for the current user
// @Tags Addresses
// @Security BearerAuth
// @Param request body models.Address true "Address details"
// @Success 201 {object} models.Address "Created address"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/me/addresses [post]
func CreateMyAddress(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var address models.Address
	if err := c.BodyParser(&address); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if address.Name == "" || address.Address == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Name and address are required"})
	}

	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Invalid user ID"})
	}
	address.UserID = parsedUserID

	if err := config.DB.Create(&address).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create address"})
	}
	return c.Status(fiber.StatusCreated).JSON(address)
}

// UpdateMyAddress godoc
// @Summary Update an address of the current user
// @Tags Addresses
// @Security BearerAuth
// @Param id path string true "Address ID (UUID)"
// @Param request body models.Address true "Updated address details"
// @Success 200 {object} models.Address "Updated address"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/me/addresses/{id} [put]
func UpdateMyAddress(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	var address models.Address
	if err := config.DB.Where("id = ? AND user_id = ?", c.Params("id"), userID).First(&address).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Address not found"})
	}
	if err := c.BodyParser(&address); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if err := config.DB.Model(&address).Updates(address).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update address"})
	}
	return c.JSON(address)
}

// DeleteMyAddress godoc
// @Summary Delete an address of the current user
// @Tags Addresses
// @Security BearerAuth
// @Param id path string true "Address ID (UUID)"
// @Success 200 {object} map[string]interface{} "Address deleted successfully"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/me/addresses/{id} [delete]
func DeleteMyAddress(c *fiber.Ctx) error {
	userID := c.Locals("userID").(string)

	result := config.DB.Where("id = ? AND user_id = ?", c.Params("id"), userID).Delete(&models.Address{})
	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete address"})
	}
	if result.RowsAffected == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Address not found"})
	}
	return c.JSON(fiber.Map{"message": "Address deleted successfully"})
}
