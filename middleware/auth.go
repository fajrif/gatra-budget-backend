package middleware

import (
    "os"
    "strings"
    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    Role   string `json:"role"`
    jwt.RegisteredClaims
}

func AuthRequired() fiber.Handler {
    return func(c *fiber.Ctx) error {
        authHeader := c.Get("Authorization")
        if authHeader == "" {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Missing authorization header",
            })
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Invalid token format",
            })
        }

        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err != nil || !token.Valid {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Invalid or expired token",
            })
        }

        claims, ok := token.Claims.(*Claims)
        if !ok {
            return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
                "error": "Invalid token claims",
            })
        }

        c.Locals("userID", claims.UserID)
        c.Locals("userEmail", claims.Email)
        c.Locals("userRole", claims.Role)

        return c.Next()
    }
}

func AdminOnly() fiber.Handler {
    return func(c *fiber.Ctx) error {
        role := c.Locals("userRole")
        if role != "admin" {
            return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
                "error": "Admin access required",
            })
        }
        return c.Next()
    }
}
