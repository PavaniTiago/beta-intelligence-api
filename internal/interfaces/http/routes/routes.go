package routes

import (
	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
	"github.com/PavaniTiago/beta-intelligence-api/internal/interfaces/http/handlers"
	"github.com/PavaniTiago/beta-intelligence-api/internal/interfaces/http/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/etag"
	"gorm.io/gorm"
)

func authMiddleware(c *fiber.Ctx) error {
	// TODO: Implementar autenticação
	return c.Next()
}

func SetupRoutes(app *fiber.App, db *gorm.DB) {
	// Add performance middleware
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Add ETag support for efficient caching
	app.Use(etag.New())

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
	sessionRepo := repositories.NewSessionRepository(db)

	// Use Cases
	userUseCase := usecases.NewUserUseCase(userRepo)
	eventUseCase := usecases.NewEventUseCase(eventRepo)
	professionUseCase := usecases.NewProfessionUseCase(professionRepo)
	funnelUseCase := usecases.NewFunnelUseCase(funnelRepo)
	sessionUseCase := usecases.NewSessionUseCase(sessionRepo)

	// Handlers
	userHandler := handlers.NewUserHandler(userUseCase, userRepo)
	eventHandler := handlers.NewEventHandler(eventUseCase)
	professionHandler := handlers.NewProfessionHandler(professionUseCase)
	funnelHandler := handlers.NewFunnelHandler(funnelUseCase)
	sessionHandler := handlers.NewSessionHandler(sessionUseCase)

	// Create handlers struct
	handlersStruct := handlers.NewHandlers(nil, db)

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

	// Sessions routes
	groups.Session.Get("/", sessionHandler.GetSessions)
	groups.Session.Get("/active", sessionHandler.GetActiveSessions)
	groups.Session.Get("/:id", sessionHandler.GetSessionByID)

	// Rotas de Performance
	setupPerformanceRoutes(groups.Public, handlersStruct.Performance)
}

// setupPerformanceRoutes configura as rotas de teste de performance
func setupPerformanceRoutes(router fiber.Router, performanceHandler *handlers.PerformanceHandler) {
	if performanceHandler != nil {
		perfGroup := router.Group("/performance")
		perfGroup.Get("/lead", performanceHandler.TestLeadPerformance)
		perfGroup.Get("/session", performanceHandler.TestSessionPerformance)
	}
}
