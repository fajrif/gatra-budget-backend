package middleware_test

import (
	"io"
	"net/http/httptest"
	"os"
	"boilerplates-backend/middleware"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

// Claims struct for testing JWT tokens
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func TestAuthRequired(t *testing.T) {
	// Set up test environment
	os.Setenv("JWT_SECRET", "test-secret-key")
	defer os.Unsetenv("JWT_SECRET")

	// Create a test Fiber app
	app := fiber.New()

	// Apply middleware
	app.Use(middleware.AuthRequired())

	// Add a test route
	app.Get("/protected", func(c *fiber.Ctx) error {
		userID := c.Locals("userID").(string)
		userEmail := c.Locals("userEmail").(string)
		userRole := c.Locals("userRole").(string)
		return c.JSON(fiber.Map{
			"user_id":    userID,
			"user_email": userEmail,
			"user_role":  userRole,
		})
	})

	t.Run("Missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Missing authorization header")
	})

	t.Run("Invalid token format - no Bearer prefix", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "InvalidTokenFormat")
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid token format")
	})

	t.Run("Invalid token - malformed", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid or expired token")
	})

	t.Run("Expired token", func(t *testing.T) {
		// Create an expired token
		claims := Claims{
			UserID: "test-user-id",
			Email:  "test@example.com",
			Role:   "user",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret-key"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Invalid or expired token")
	})

	t.Run("Valid token", func(t *testing.T) {
		// Create a valid token
		claims := Claims{
			UserID: "test-user-id-123",
			Email:  "valid@example.com",
			Role:   "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret-key"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "test-user-id-123")
		assert.Contains(t, string(body), "valid@example.com")
		assert.Contains(t, string(body), "admin")
	})

	t.Run("Token with wrong secret", func(t *testing.T) {
		// Create a token with different secret
		claims := Claims{
			UserID: "test-user-id",
			Email:  "test@example.com",
			Role:   "user",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("wrong-secret-key"))

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		resp, _ := app.Test(req)

		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	})
}

func TestAdminOnly(t *testing.T) {
	app := fiber.New()

	// Apply admin middleware
	app.Use(middleware.AdminOnly())

	// Add a test route
	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Admin access granted"})
	})

	t.Run("Non-admin user denied", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)

		// Create a test context with user role
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}

		// Since there's no userRole in locals, it will be nil
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	})

	t.Run("Admin user allowed", func(t *testing.T) {
		// Create app with manual locals setting
		testApp := fiber.New()

		testApp.Use(func(c *fiber.Ctx) error {
			c.Locals("userRole", "admin")
			return c.Next()
		})

		testApp.Use(middleware.AdminOnly())

		testApp.Get("/admin", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "Admin access granted"})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		resp, _ := testApp.Test(req)

		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Admin access granted")
	})

	t.Run("Regular user denied", func(t *testing.T) {
		// Create app with manual locals setting
		testApp := fiber.New()

		testApp.Use(func(c *fiber.Ctx) error {
			c.Locals("userRole", "user")
			return c.Next()
		})

		testApp.Use(middleware.AdminOnly())

		testApp.Get("/admin", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "Admin access granted"})
		})

		req := httptest.NewRequest("GET", "/admin", nil)
		resp, _ := testApp.Test(req)

		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		assert.Contains(t, string(body), "Admin access required")
	})
}
