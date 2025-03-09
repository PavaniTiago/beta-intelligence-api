package handlers

import (
	"fmt"
	"strconv"

	"github.com/PavaniTiago/beta-intelligence/internal/application/usecases"
	"github.com/gofiber/fiber/v2"
)

type FunnelHandler struct {
	funnelUseCase usecases.FunnelUseCase
}

func NewFunnelHandler(funnelUseCase usecases.FunnelUseCase) *FunnelHandler {
	return &FunnelHandler{funnelUseCase}
}

func (h *FunnelHandler) GetFunnels(c *fiber.Ctx) error {
	// Get query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))

	// Definir um limite muito alto por padrão para retornar todos os registros
	// Mas ainda permitir que o cliente defina um limite menor se desejar
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
		"funnel_id":    "funnel_id",
		"funnel_name":  "funnel_name",
		"funnel_tag":   "funnel_tag",
		"created_at":   "created_at",
		"product_id":   "product_id",
		"global":       "global",
		"product_name": "products.product_name",
	}

	orderBy := "created_at desc" // default ordering
	if field, ok := validSortFields[sortBy]; ok {
		orderBy = field + " " + sortDirection
	}

	funnels, total, err := h.funnelUseCase.GetFunnels(page, limit, orderBy)
	if err != nil {
		// Log do erro para debug
		fmt.Printf("Error getting funnels: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": funnels,
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
