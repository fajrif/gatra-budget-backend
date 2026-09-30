package handlers

import (
    "gatra-budget-backend/config"
    "gatra-budget-backend/models"
    "os"
    "strconv"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
    Email    string `json:"email" example:"user@example.com"`
    Password string `json:"password" example:"password123"`
}

type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

// Login godoc
// @Summary User login
// @Description Authenticate user and return JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} map[string]any "Login successful with token and user data"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 401 {object} map[string]any "Invalid email or password"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/auth/login [post]
func Login(c *fiber.Ctx) error {
    var req LoginRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Find user
    var user models.User
    if err := config.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid email or password",
        })
    }

    // Check password
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "error": "Invalid email or password",
        })
    }

    // Generate JWT token
    expireHours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRE_HOURS"))
    if expireHours == 0 {
        expireHours = 72
    }

    claims := Claims{
        UserID: user.ID.String(),
        Email:  user.Email,
        Role:   user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * time.Duration(expireHours))),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to generate token",
        })
    }

    return c.JSON(fiber.Map{
        "token": tokenString,
        "user": fiber.Map{
            "id":        user.ID,
            "email":     user.Email,
            "full_name": user.FullName,
            "role":      user.Role,
        },
    })
}

// GetMe godoc
// @Summary Get current user profile
// @Description Get authenticated user's profile information
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.User "User profile data"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 404 {object} map[string]any "User not found"
// @Router /api/me [get]
func GetMe(c *fiber.Ctx) error {
    userID := c.Locals("userID").(string)

    var user models.User
    if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    return c.JSON(fiber.Map{
        "id":         user.ID,
        "email":      user.Email,
        "full_name":  user.FullName,
        "role":       user.Role,
        "photo_url":  user.PhotoUrl,
        "created_at": user.CreatedAt,
        "updated_at": user.UpdatedAt,
    })
}

type UpdateMeRequest struct {
    FullName string `json:"full_name,omitempty" example:"John Doe Updated"`
    Password string `json:"password,omitempty" example:"newpassword123"`
}

// UpdateMe godoc
// @Summary Update current user profile
// @Description Update authenticated user's profile information (full_name and password)
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateMeRequest true "Update data (full_name, password)"
// @Success 200 {object} map[string]any "Updated user profile"
// @Failure 400 {object} map[string]any "Invalid request body"
// @Failure 401 {object} map[string]any "Unauthorized"
// @Failure 404 {object} map[string]any "User not found"
// @Failure 500 {object} map[string]any "Internal server error"
// @Router /api/me [put]
func UpdateMe(c *fiber.Ctx) error {
    userID := c.Locals("userID").(string)

    var user models.User
    if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "error": "User not found",
        })
    }

    var req UpdateMeRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }

    // Update full name if provided
    if req.FullName != "" {
        user.FullName = req.FullName
    }

    // Update password if provided
    if req.Password != "" {
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
        if err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to hash password",
            })
        }
        user.PasswordHash = string(hashedPassword)
    }

    if err := config.DB.Save(&user).Error; err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to update user",
        })
    }

    return c.JSON(fiber.Map{
        "id":         user.ID,
        "email":      user.Email,
        "full_name":  user.FullName,
        "role":       user.Role,
        "photo_url":  user.PhotoUrl,
        "created_at": user.CreatedAt,
        "updated_at": user.UpdatedAt,
    })
}
