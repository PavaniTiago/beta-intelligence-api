package handlers

import (
	"strconv"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/gofiber/fiber/v2"
)

type ProductHandler struct {
	productUseCase usecases.ProductUseCase
}

func NewProductHandler(productUseCase usecases.ProductUseCase) *ProductHandler {
	return &ProductHandler{productUseCase}
}

func (h *ProductHandler) GetProductsWithFunnels(c *fiber.Ctx) error {
	// Get query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "1000"))

	// Get sort parameters
	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")

	// Validate sort direction
	if sortDirection != "asc" && sortDirection != "desc" {
		sortDirection = "desc"
	}

	// Validate sortBy field and build orderBy
	validSortFields := map[string]string{
		"product_id":    "product_id",
		"created_at":    "created_at",
		"product_name":  "product_name",
		"profession_id": "profession_id",
	}

	orderBy := "created_at desc" // default ordering
	if field, ok := validSortFields[sortBy]; ok {
		orderBy = field + " " + sortDirection
	}

	products, total, err := h.productUseCase.GetProductsWithFunnels(page, limit, orderBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Build response with simplified structure
	var responseProducts []fiber.Map
	for _, product := range products {
		// Create simplified funnel objects for this product
		simplifiedFunnels := []fiber.Map{}
		for _, funnel := range product.Funnels {
			simplifiedFunnels = append(simplifiedFunnels, fiber.Map{
				"funnel_id":   funnel.FunnelID,
				"funnel_name": funnel.FunnelName,
				"funnel_tag":  funnel.FunnelTag,
				"is_active":   funnel.IsActive,
			})
		}

		// Create product object with simplified funnels
		responseProducts = append(responseProducts, fiber.Map{
			"product_id":    product.ProductID,
			"product_name":  product.ProductName,
			"profession_id": product.ProfessionID,
			"funnels":       simplifiedFunnels,
		})
	}

	return c.JSON(fiber.Map{
		"data": responseProducts,
		"meta": fiber.Map{
			"total":             total,
			"page":              page,
			"limit":             limit,
			"last_page":         (total + int64(limit) - 1) / int64(limit),
			"sort_by":           sortBy,
			"sort_direction":    sortDirection,
			"valid_sort_fields": getKeys(validSortFields),
		},
	})
}

// GetFunnelsByProfessionID retrieves all funnels for products associated with a given profession_id
func (h *ProductHandler) GetFunnelsByProfessionID(c *fiber.Ctx) error {
	// Parse profession_id from URL parameter
	professionID, err := strconv.Atoi(c.Params("profession_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid profession ID format",
		})
	}

	// Get funnels for the profession
	funnels, err := h.productUseCase.GetFunnelsByProfessionID(professionID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// If no funnels found, return an empty array instead of null
	if funnels == nil {
		funnels = []entities.Funnel{}
	}

	// Create simplified funnel objects
	var simplifiedFunnels []fiber.Map
	for _, funnel := range funnels {
		simplifiedFunnels = append(simplifiedFunnels, fiber.Map{
			"funnel_id":   funnel.FunnelID,
			"funnel_name": funnel.FunnelName,
			"funnel_tag":  funnel.FunnelTag,
			"is_active":   funnel.IsActive,
		})
	}

	return c.JSON(fiber.Map{
		"data": simplifiedFunnels,
	})
}
