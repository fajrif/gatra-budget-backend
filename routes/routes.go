package routes

import (
	"boilerplates-backend/handlers"
	"boilerplates-backend/middleware"

	"github.com/gofiber/fiber/v2"

	fiberSwagger "github.com/swaggo/fiber-swagger"
)

func Setup(app *fiber.App) {
	// Swagger documentation route
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	api := app.Group("/api")

	// Public routes
	auth := api.Group("/auth")
	auth.Post("/login", handlers.Login)

	// Protected routes
	api.Use(middleware.AuthRequired())

	// Profile routes
	api.Get("/me", handlers.GetMe)
	api.Put("/me", handlers.UpdateMe)

	// Upload routes
	api.Post("/upload/:resource/:field/:id", handlers.UploadResourceField)
	api.Delete("/upload/:resource/:field/:id", handlers.DeleteResourceField)

	// Address hierarchy routes
	countries := api.Group("/countries")
	countries.Get("/", handlers.GetAllCountries)
	countries.Get("/:id", handlers.GetCountry)
	countries.Post("/", handlers.CreateCountry)
	countries.Put("/:id", handlers.UpdateCountry)
	countries.Delete("/:id", handlers.DeleteCountry)

	provinces := api.Group("/provinces")
	provinces.Get("/", handlers.GetAllProvinces)
	provinces.Get("/:id", handlers.GetProvince)
	provinces.Post("/", handlers.CreateProvince)
	provinces.Put("/:id", handlers.UpdateProvince)
	provinces.Delete("/:id", handlers.DeleteProvince)

	cities := api.Group("/cities")
	cities.Get("/", handlers.GetAllCities)
	cities.Get("/:id", handlers.GetCity)
	cities.Post("/", handlers.CreateCity)
	cities.Put("/:id", handlers.UpdateCity)
	cities.Delete("/:id", handlers.DeleteCity)

	areas := api.Group("/areas")
	areas.Get("/", handlers.GetAllAreas)
	areas.Get("/:id", handlers.GetArea)
	areas.Post("/", handlers.CreateArea)
	areas.Put("/:id", handlers.UpdateArea)
	areas.Delete("/:id", handlers.DeleteArea)

	// Current user's addresses
	meAddresses := api.Group("/me/addresses")
	meAddresses.Get("/", handlers.GetMyAddresses)
	meAddresses.Get("/:id", handlers.GetMyAddress)
	meAddresses.Post("/", handlers.CreateMyAddress)
	meAddresses.Put("/:id", handlers.UpdateMyAddress)
	meAddresses.Delete("/:id", handlers.DeleteMyAddress)

	// Admin only routes
	api.Use(middleware.AdminOnly())

	// Users management routes (admin only)
	users := api.Group("/users")
	users.Get("/", handlers.GetAllUsers)
	users.Get("/:id", handlers.GetUser)
	users.Post("/", handlers.CreateUser)
	users.Put("/:id", handlers.UpdateUser)
	users.Delete("/:id", handlers.DeleteUser)

	// Admin can view any user's addresses
	users.Get("/:user_id/addresses", handlers.GetUserAddresses)
}
