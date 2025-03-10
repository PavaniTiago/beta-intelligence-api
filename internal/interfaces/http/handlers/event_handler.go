package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/gofiber/fiber/v2"
)

type EventHandler struct {
	eventUseCase usecases.EventUseCase
}

func NewEventHandler(eventUseCase usecases.EventUseCase) *EventHandler {
	return &EventHandler{eventUseCase}
}

func (h *EventHandler) GetEvents(c *fiber.Ctx) error {
	// Get query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// Get sort parameters
	sortBy := c.Query("sortBy", "event_time")
	sortDirection := c.Query("sortDirection", "desc")

	// Obter todos os parâmetros da query usando QueryParams que suporta múltiplos valores
	queryParams := c.Context().QueryArgs()
	fmt.Printf("All query parameters: %v\n", queryParams)

	// Get profession filter - usando uma abordagem que captura todos os valores
	var professionIDs []int

	// Capturar todos os valores de profession_ids[]
	profIDsArray := queryParams.PeekMulti("profession_ids[]")
	fmt.Printf("profession_ids[] values: %v\n", profIDsArray)
	for _, idBytes := range profIDsArray {
		if id, err := strconv.Atoi(string(idBytes)); err == nil {
			professionIDs = append(professionIDs, id)
			fmt.Printf("Added profession_id from array format: %d\n", id)
		}
	}

	// Capturar valores de profession_ids (sem colchetes)
	if profIDsBytes := queryParams.Peek("profession_ids"); profIDsBytes != nil {
		profIDsStr := string(profIDsBytes)
		fmt.Printf("profession_ids value: %s\n", profIDsStr)

		// Decodificar manualmente o valor URL-encoded
		profIDsStr = strings.ReplaceAll(profIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		profIDsArr := strings.Split(profIDsStr, ",")
		for _, idStr := range profIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				professionIDs = append(professionIDs, id)
				fmt.Printf("Added profession_id from plural format: %d\n", id)
			}
		}
	}

	// Capturar valores de profession_id (singular)
	if profIDsBytes := queryParams.Peek("profession_id"); profIDsBytes != nil {
		profIDsStr := string(profIDsBytes)
		fmt.Printf("profession_id value: %s\n", profIDsStr)

		// Decodificar manualmente o valor URL-encoded
		profIDsStr = strings.ReplaceAll(profIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		profIDsArr := strings.Split(profIDsStr, ",")
		for _, idStr := range profIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				professionIDs = append(professionIDs, id)
				fmt.Printf("Added profession_id from singular format: %d\n", id)
			}
		}
	}

	fmt.Printf("Final profession_ids for filtering: %v\n", professionIDs)

	// Get funnel filter - usando a mesma abordagem
	var funnelIDs []int

	// Capturar todos os valores de funnel_ids[]
	funnelIDsArray := queryParams.PeekMulti("funnel_ids[]")
	fmt.Printf("funnel_ids[] values: %v\n", funnelIDsArray)
	for _, idBytes := range funnelIDsArray {
		if id, err := strconv.Atoi(string(idBytes)); err == nil {
			funnelIDs = append(funnelIDs, id)
			fmt.Printf("Added funnel_id from array format: %d\n", id)
		}
	}

	// Capturar valores de funnel_ids (sem colchetes)
	if funnelIDsBytes := queryParams.Peek("funnel_ids"); funnelIDsBytes != nil {
		funnelIDsStr := string(funnelIDsBytes)
		fmt.Printf("funnel_ids value: %s\n", funnelIDsStr)

		// Decodificar manualmente o valor URL-encoded
		funnelIDsStr = strings.ReplaceAll(funnelIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		funnelIDsArr := strings.Split(funnelIDsStr, ",")
		for _, idStr := range funnelIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				funnelIDs = append(funnelIDs, id)
				fmt.Printf("Added funnel_id from plural format: %d\n", id)
			}
		}
	}

	// Capturar valores de funnel_id (singular)
	if funnelIDsBytes := queryParams.Peek("funnel_id"); funnelIDsBytes != nil {
		funnelIDsStr := string(funnelIDsBytes)
		fmt.Printf("funnel_id value: %s\n", funnelIDsStr)

		// Decodificar manualmente o valor URL-encoded
		funnelIDsStr = strings.ReplaceAll(funnelIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		funnelIDsArr := strings.Split(funnelIDsStr, ",")
		for _, idStr := range funnelIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				funnelIDs = append(funnelIDs, id)
				fmt.Printf("Added funnel_id from singular format: %d\n", id)
			}
		}
	}

	fmt.Printf("Final funnel_ids for filtering: %v\n", funnelIDs)

	// Validate sort direction
	if sortDirection != "asc" && sortDirection != "desc" {
		sortDirection = "desc"
	}

	// Validate sortBy field and build orderBy
	validSortFields := map[string]string{
		// Event fields
		"event_id":     "events.event_id",
		"event_name":   "events.event_name",
		"pageview_id":  "events.pageview_id",
		"session_id":   "events.session_id",
		"event_time":   "events.event_time",
		"event_source": "events.event_source",
		"event_type":   "events.event_type",

		// User fields
		"fullname":  "users.fullname",
		"email":     "users.email",
		"phone":     "users.phone",
		"is_client": "users.isClient",

		// Session fields
		"utm_source":   "sessions.utm_source",
		"utm_medium":   "sessions.utm_medium",
		"utm_campaign": "sessions.utm_campaign",
		"utm_content":  "sessions.utm_content",
		"utm_term":     "sessions.utm_term",
		"country":      "sessions.country",
		"state":        "sessions.state",
		"city":         "sessions.city",

		// Profession fields
		"profession_name": "professions.profession_name",
		"meta_pixel":      "professions.meta_pixel",
		"meta_token":      "professions.meta_token",

		// Product fields
		"product_name": "products.product_name",

		// Funnel fields
		"funnel_name": "funnels.funnel_name",
		"funnel_tag":  "funnels.funnel_tag",
		"global":      "funnels.global",
	}

	orderBy := "events.event_time desc" // default ordering
	if field, ok := validSortFields[sortBy]; ok {
		orderBy = field + " " + sortDirection
	}

	// Parse date parameters
	from := c.Query("from", "")
	to := c.Query("to", "")

	var fromTime, toTime time.Time
	var err error

	// Tenta primeiro o formato com hora (ISO 8601)
	if from != "" {
		// Tenta primeiro o formato com hora
		fromTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			// Se falhar, tenta o formato apenas com data
			fromTime, err = time.Parse("2006-01-02", from)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid from date format. Use YYYY-MM-DD or YYYY-MM-DDThh:mm:ssZ",
				})
			}
			// Se for apenas data, define para o início do dia
			fromTime = time.Date(fromTime.Year(), fromTime.Month(), fromTime.Day(), 0, 0, 0, 0, fromTime.Location())
		}
	} else {
		// If no from date, use 30 days ago as default
		fromTime = time.Now().AddDate(0, 0, -30)
	}

	if to != "" {
		// Tenta primeiro o formato com hora
		toTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			// Se falhar, tenta o formato apenas com data
			toTime, err = time.Parse("2006-01-02", to)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid to date format. Use YYYY-MM-DD or YYYY-MM-DDThh:mm:ssZ",
				})
			}
			// Se for apenas data, define para o final do dia
			toTime = time.Date(toTime.Year(), toTime.Month(), toTime.Day(), 23, 59, 59, 999999999, toTime.Location())
		}
	} else {
		// If no to date, use current time
		toTime = time.Now()
	}

	// Após processar os parâmetros de data
	fmt.Printf("From time: %v (UTC: %v)\n", fromTime, fromTime.UTC())
	fmt.Printf("To time: %v (UTC: %v)\n", toTime, toTime.UTC())

	events, total, err := h.eventUseCase.GetEvents(page, limit, orderBy, fromTime, toTime, professionIDs, funnelIDs)
	if err != nil {
		fmt.Printf("Error fetching events: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"data": events,
		"meta": fiber.Map{
			"total":             total,
			"page":              page,
			"limit":             limit,
			"last_page":         (total + int64(limit) - 1) / int64(limit),
			"from":              fromTime.Format(time.RFC3339),
			"to":                toTime.Format(time.RFC3339),
			"sort_by":           sortBy,
			"sort_direction":    sortDirection,
			"profession_ids":    professionIDs,
			"funnel_ids":        funnelIDs,
			"valid_sort_fields": getKeys(validSortFields),
		},
	})
}

// Helper function to get map keys
func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
