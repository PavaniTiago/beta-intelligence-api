package handlers

import (
	"strconv"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/gofiber/fiber/v2"
)

type ProfessionHandler struct {
	professionUseCase usecases.ProfessionUseCase
}

func NewProfessionHandler(professionUseCase usecases.ProfessionUseCase) *ProfessionHandler {
	return &ProfessionHandler{professionUseCase}
}

func (h *ProfessionHandler) GetProfessions(c *fiber.Ctx) error {
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
		"profession_id":   "profession_id",
		"created_at":      "created_at",
		"profession_name": "profession_name",
		"meta_pixel":      "meta_pixel",
		"meta_token":      "meta_token",
	}

	orderBy := "created_at desc" // default ordering
	if field, ok := validSortFields[sortBy]; ok {
		orderBy = field + " " + sortDirection
	}

	professions, total, err := h.professionUseCase.GetProfessions(page, limit, orderBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Create simplified response objects with only id and name
	var simplifiedProfessions []fiber.Map
	for _, p := range professions {
		simplifiedProfessions = append(simplifiedProfessions, fiber.Map{
			"profession_id":   p.ProfessionID,
			"profession_name": p.ProfessionName,
		})
	}

	return c.JSON(fiber.Map{
		"data": simplifiedProfessions,
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
