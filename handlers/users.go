package handlers

import (
	"gatra-budget-backend/config"
	"gatra-budget-backend/helpers"
	"gatra-budget-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ValidateUserRequest validates the user request fields
func ValidateUserRequest(req *models.UserRequest) error {
	if req.Email == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Email is required")
	}
	if req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Password is required")
	}
	if req.FullName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Full name is required")
	}
	if !helpers.IsValidEmail(req.Email) {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid email format")
	}
	// Additional validations can be added here
	if !helpers.IsStrongPassword(req.Password) {
		return fiber.NewError(fiber.StatusBadRequest, "Password must be at least 8 characters long, must contain at least one number, one uppercase letter, one lowercase letter, and one special character")
	}
	if req.Role != "" && req.Role != "user" && req.Role != "admin" && req.Role != "operator" {
		return fiber.NewError(fiber.StatusBadRequest, "Role must be either 'user' or 'admin' or 'operator'")
	}
	return nil
}

// GetAllUsers godoc
// @Summary Get all users (Admin only)
// @Description Retrieve all users with pagination and optional search filter. Only accessible by admin users.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20)"
// @Param all query bool false "Get all records without pagination"
// @Param search query string false "Search by email, full name"
// @Success 200 {object} map[string]any "List of users with pagination"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/users [get]
func GetAllUsers(c *fiber.Ctx) error {
	pagination := helpers.GetPaginationParams(c)

	query := config.DB.Model(&models.User{})

	// add params for not using pagination
	if c.Query("all") == "true" {
		pagination.Limit = -1 // No limit
		pagination.Offset = 0 // No offset
	}

	// Filter search
	if searchQuery := c.Query("search"); searchQuery != "" {
		searchTerm := "%" + searchQuery + "%"
		query = query.Where("email ILIKE ? OR full_name ILIKE ?", searchTerm, searchTerm)
	}

	// Create a separate query for counting
	queryCount := config.DB.Model(&models.User{})
	if searchQuery := c.Query("search"); searchQuery != "" {
		searchTerm := "%" + searchQuery + "%"
		queryCount = queryCount.Where("email ILIKE ? OR full_name ILIKE ?", searchTerm, searchTerm)
	}

	// Get users
	var users []models.User
	if err := query.Offset(pagination.Offset).Limit(pagination.Limit).Order("created_at DESC").Find(&users).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch users",
		})
	}

	// Format response
	userList := make([]fiber.Map, 0)
	for _, user := range users {
		userList = append(userList, fiber.Map{
			"id":           user.ID,
			"email":        user.Email,
			"full_name":    user.FullName,
			"role":         user.Role,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		})
	}

	response, err := helpers.CreatePaginationResponse(queryCount, userList, "users", pagination.Page, pagination.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create pagination response",
		})
	}

	return c.JSON(response)
}

// GetUser godoc
// @Summary Get user by ID (Admin only)
// @Description Retrieve a specific user by their ID. Only accessible by admin users.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} map[string]any "User details"
// @Failure 400 {object} map[string]any "Invalid user ID"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 404 {object} map[string]any "User not found"
// @Router /api/users/{id} [get]
func GetUser(c *fiber.Ctx) error {
	id := c.Params("id")

	// Validate UUID
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var user models.User
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"id":           user.ID,
		"email":        user.Email,
		"full_name":    user.FullName,
		"role":         user.Role,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}

// CreateUser godoc
// @Summary Create a new user (Admin only)
// @Description Create a new user with email, password, full name, and role. Only accessible by admin users.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.UserRequest true "User creation details"
// @Success 201 {object} map[string]any "User created successfully"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 409 {object} map[string]any "Email already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/users [post]
func CreateUser(c *fiber.Ctx) error {
	var req models.UserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate user fields
	if err := ValidateUserRequest(&req); err != nil {
		return c.Status(err.(*fiber.Error).Code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Check if user exists
	var existingUser models.User
	if err := config.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Email already exists",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to hash password",
		})
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FullName:     req.FullName,
		Role:         req.Role,
	}

	if err := config.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User created successfully",
		"user": fiber.Map{
			"id":           user.ID,
			"email":        user.Email,
			"full_name":    user.FullName,
			"role":         user.Role,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		},
	})
}

// UpdateUser godoc
// @Summary Update user (Admin only)
// @Description Update user information including email, password, full name, and role. Only accessible by admin users.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Param request body models.UserRequest true "User update details"
// @Success 200 {object} map[string]any "Updated user details"
// @Failure 400 {object} map[string]any "Invalid request or user ID"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 404 {object} map[string]any "User not found"
// @Failure 409 {object} map[string]any "Email already exists"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/users/{id} [put]
func UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	// Validate UUID
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var user models.User
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	var req models.UserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Check if email is being updated and if it already exists
	if req.Email != "" && req.Email != user.Email {
		if !helpers.IsValidEmail(req.Email) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid email format",
			})
		}
		var existingUser models.User
		if err := config.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Email already exists",
			})
		}
		user.Email = req.Email
	}

	// Update password if provided
	if req.Password != "" {
		if !helpers.IsStrongPassword(req.Password) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Password must be at least 8 characters long, must contain at least one number, one uppercase letter, one lowercase letter, and one special character",
			})
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to hash password",
			})
		}
		user.PasswordHash = string(hashedPassword)
	}

	// Update full name if provided
	if req.FullName != "" {
		user.FullName = req.FullName
	}

	// Update role if provided
	if req.Role != "" {
		if req.Role != "user" && req.Role != "admin" && req.Role != "operator" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Role must be either 'user' or 'admin' or 'operator'",
			})
		}
		user.Role = req.Role
	}

	if err := config.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update user",
		})
	}

	return c.JSON(fiber.Map{
		"id":           user.ID,
		"email":        user.Email,
		"full_name":    user.FullName,
		"role":         user.Role,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}

// DeleteUser godoc
// @Summary Delete user (Admin only)
// @Description Delete a user by their ID. Only accessible by admin users.
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (UUID)"
// @Success 200 {object} map[string]any "User deleted successfully"
// @Failure 400 {object} map[string]any "Invalid user ID"
// @Failure 403 {object} map[string]any "Admin access required"
// @Failure 404 {object} map[string]any "User not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/users/{id} [delete]
func DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	// Validate UUID
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var user models.User
	if err := config.DB.Where("id = ?", id).First(&user).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	if err := config.DB.Delete(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete user",
		})
	}

	return c.JSON(fiber.Map{
		"message": "User deleted successfully",
	})
}
