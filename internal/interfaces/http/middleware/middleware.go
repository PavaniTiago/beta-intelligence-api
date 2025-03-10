package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func SetupMiddlewares(app *fiber.App) {
	// CORS configuration
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://beta-intelligence.vercel.app, http://localhost:3000",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes
	}))

	// Common middleware
	app.Use(func(c *fiber.Ctx) error {
		return c.Next()
	})
}

// RouteGroups define os grupos de rotas da API
type RouteGroups struct {
	Public    fiber.Router
	Lead      fiber.Router
	Client    fiber.Router
	Anonymous fiber.Router
}

// SetupRouteGroups configura os grupos de rotas com seus respectivos middlewares
func SetupRouteGroups(app *fiber.App, authMiddleware func(c *fiber.Ctx) error) RouteGroups {
	// Grupo público (sem autenticação)
	public := app.Group("/")

	// Grupo para leads (com autenticação)
	lead := app.Group("/lead")
	lead.Use(authMiddleware)

	// Grupo para clientes (com autenticação)
	client := app.Group("/client")
	client.Use(authMiddleware)

	// Grupo para usuários anônimos
	anonymous := app.Group("/anonymous")

	return RouteGroups{
		Public:    public,
		Lead:      lead,
		Client:    client,
		Anonymous: anonymous,
	}
}
