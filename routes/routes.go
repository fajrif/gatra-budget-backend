package routes

import (
	"gatra-budget-backend/handlers"
	"gatra-budget-backend/middleware"

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

	// Clients / Contract Types / Cost Types — reads open to any authenticated
	// user, writes admin-only (inline middleware, registered before the
	// AdminOnly() boundary below so reads stay available to everyone).
	clients := api.Group("/clients")
	clients.Get("/", handlers.GetAllClients)
	clients.Get("/:id", handlers.GetClient)
	clients.Post("/", middleware.AdminOnly(), handlers.CreateClient)
	clients.Put("/:id", middleware.AdminOnly(), handlers.UpdateClient)
	clients.Delete("/:id", middleware.AdminOnly(), handlers.DeleteClient)

	contractTypes := api.Group("/contract-types")
	contractTypes.Get("/", handlers.GetAllContractTypes)
	contractTypes.Get("/:id", handlers.GetContractType)
	contractTypes.Post("/", middleware.AdminOnly(), handlers.CreateContractType)
	contractTypes.Put("/:id", middleware.AdminOnly(), handlers.UpdateContractType)
	contractTypes.Delete("/:id", middleware.AdminOnly(), handlers.DeleteContractType)

	costTypes := api.Group("/cost-types")
	costTypes.Get("/", handlers.GetAllCostTypes)
	costTypes.Get("/:id", handlers.GetCostType)
	costTypes.Post("/", middleware.AdminOnly(), handlers.CreateCostType)
	costTypes.Put("/:id", middleware.AdminOnly(), handlers.UpdateCostType)
	costTypes.Delete("/:id", middleware.AdminOnly(), handlers.DeleteCostType)

	// Projects — full CRUD, any authenticated user
	projects := api.Group("/projects")
	projects.Get("/", handlers.GetAllProjects)
	projects.Post("/import", handlers.ImportProjects)
	projects.Get("/:id", handlers.GetProject)
	projects.Post("/", handlers.CreateProject)
	projects.Put("/:id", handlers.UpdateProject)
	projects.Delete("/:id", handlers.DeleteProject)

	// Transactions — full CRUD, any authenticated user
	transactions := api.Group("/transactions")
	transactions.Get("/", handlers.GetAllTransactions)
	transactions.Post("/import", handlers.ImportTransactions)
	transactions.Get("/:id", handlers.GetTransaction)
	transactions.Post("/", handlers.CreateTransaction)
	transactions.Put("/:id", handlers.UpdateTransaction)
	transactions.Delete("/:id", handlers.DeleteTransaction)

	// Invoices — full CRUD, any authenticated user
	invoices := api.Group("/invoices")
	invoices.Get("/", handlers.GetAllInvoices)
	invoices.Post("/import", handlers.ImportInvoices)
	invoices.Get("/:id", handlers.GetInvoice)
	invoices.Post("/", handlers.CreateInvoice)
	invoices.Put("/:id", handlers.UpdateInvoice)
	invoices.Delete("/:id", handlers.DeleteInvoice)

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

	// Financial reports (admin only). These are the only routes that return
	// remaining budget and profit margin; the plain project and transaction
	// payloads deliberately omit them.
	reports := api.Group("/reports")
	reports.Get("/projects-summary", handlers.GetProjectsSummaryReport)
	reports.Get("/projects/:id/summary", handlers.GetProjectSummaryReport)
	reports.Get("/monthly", handlers.GetMonthlyReport)
}
