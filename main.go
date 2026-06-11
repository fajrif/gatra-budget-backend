package main

import (
	"log"
	"os"
	"boilerplates-backend/config"
	"boilerplates-backend/routes"

	_ "boilerplates-backend/docs"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

// @title Boilerplates Backend API
// @version 1.0
// @description REST API for the Boilerplates Go + React starter kit
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@boilerplates.dev

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Connect to database
	config.ConnectDB()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		// Increase buffer sizes to handle large headers/cookies (fixes "Request Header Fields Too Large" error)
		ReadBufferSize:  16384, // 16KB (default is 4KB)
		WriteBufferSize: 16384, // 16KB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("CORS_ORIGIN"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Static file serving for uploads
	app.Static("/uploads", "./uploads")

	// root path
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Hello, boilerplates API is running 👋",
		})
	})

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "boilerplates API is running",
		})
	})

	// Setup routes
	routes.Setup(app)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
