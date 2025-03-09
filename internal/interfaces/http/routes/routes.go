package routes

import (
	"github.com/PavaniTiago/beta-intelligence/internal/application/usecases"
	"github.com/PavaniTiago/beta-intelligence/internal/domain/repositories"
	"github.com/PavaniTiago/beta-intelligence/internal/interfaces/http/handlers"
	"github.com/PavaniTiago/beta-intelligence/internal/interfaces/http/middleware"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func authMiddleware(c *fiber.Ctx) error {
	// TODO: Implementar autenticação
	return c.Next()
}

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// Repositories
	userRepo := repositories.NewUserRepository(db)
	eventRepo := repositories.NewEventRepository(db)
	professionRepo := repositories.NewProfessionRepository(db)
	funnelRepo := repositories.NewFunnelRepository(db)

	// Use Cases
	userUseCase := usecases.NewUserUseCase(userRepo)
	eventUseCase := usecases.NewEventUseCase(eventRepo)
	professionUseCase := usecases.NewProfessionUseCase(professionRepo)
	funnelUseCase := usecases.NewFunnelUseCase(funnelRepo)

	// Handlers
	userHandler := handlers.NewUserHandler(userUseCase, userRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)
	professionHandler := handlers.NewProfessionHandler(professionUseCase)
	funnelHandler := handlers.NewFunnelHandler(funnelUseCase)

	// Routes
	groups := middleware.SetupRouteGroups(app, authMiddleware)

	// Rota antiga de users
	groups.Public.Get("/users", userHandler.GetUsers)

	// Rotas para leads
	groups.Lead.Get("/", userHandler.GetLeads)

	// Rotas para clientes
	groups.Client.Get("/", userHandler.GetClients)

	// Rotas para anônimos
	groups.Anonymous.Get("/", userHandler.GetAnonymous)

	// Events routes
	groups.Public.Get("/events", eventHandler.GetEvents)

	// Professions routes
	groups.Public.Get("/professions", professionHandler.GetProfessions)

	// Funnels routes
	groups.Public.Get("/funnels", funnelHandler.GetFunnels)
}
