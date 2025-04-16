package routes

import (
	"github.com/PavaniTiago/beta-intelligence-api/internal/interfaces/http/handlers"
	"github.com/gofiber/fiber/v2"
)

func RegisterProductRoutes(app *fiber.App, productHandler *handlers.ProductHandler) {
	products := app.Group("/api/v1/products")

	// Route to get products with their funnels
	products.Get("/with-funnels", productHandler.GetProductsWithFunnels)
}
