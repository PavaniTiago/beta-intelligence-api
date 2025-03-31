package handlers

import (
	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Handlers struct {
	useCases    *usecases.UseCases
	Performance *PerformanceHandler
}

func NewHandlers(useCases *usecases.UseCases, db *gorm.DB) *Handlers {
	return &Handlers{
		useCases:    useCases,
		Performance: NewPerformanceHandler(db),
	}
}

func (h *Handlers) RegisterRoutes(app *fiber.App) {
	v1 := app.Group("/api/v1")

	// Users routes
	users := v1.Group("/users")
	users.Post("/", h.CreateUser)
	users.Get("/", h.GetUsers)
	users.Get("/:id", h.GetUser)
	users.Put("/:id", h.UpdateUser)
	users.Delete("/:id", h.DeleteUser)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
		})
	})
}

// Handler methods
func (h *Handlers) CreateUser(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "User created",
	})
}

func (h *Handlers) GetUsers(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "List users",
	})
}

func (h *Handlers) GetUser(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"message": "Get user",
		"id":      id,
	})
}

func (h *Handlers) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"message": "Update user",
		"id":      id,
	})
}

func (h *Handlers) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"message": "Delete user",
		"id":      id,
	})
}
