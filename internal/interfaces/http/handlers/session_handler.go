package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SessionHandler struct {
	sessionUseCase *usecases.SessionUseCase
}

func NewSessionHandler(sessionUseCase *usecases.SessionUseCase) *SessionHandler {
	return &SessionHandler{
		sessionUseCase: sessionUseCase,
	}
}

// GetSessions retorna todas as sessões com paginação, ordenação e filtros
func (h *SessionHandler) GetSessions(c *fiber.Ctx) error {
	// Parse de parâmetros básicos
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'page' parameter"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'limit' parameter"})
	}

	// Verificar se é para retornar apenas a contagem
	countOnly := c.Query("count_only", "false") == "true"

	// Parse de ordenação
	sortBy := c.Query("sortBy", "sessionStart")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	// Parse de filtros de data
	from := time.Time{}
	to := time.Now()

	// Parse from e to das query params
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	if fromStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			// Tentar formato alternativo YYYY-MM-DD
			fromTime, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid 'from' date format. Use ISO format (e.g., 2023-01-01T00:00:00Z) or YYYY-MM-DD"})
			}
			// Definir para início do dia
			fromTime = time.Date(fromTime.Year(), fromTime.Month(), fromTime.Day(), 0, 0, 0, 0, fromTime.Location())
		}
		from = fromTime
	}

	if toStr != "" {
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			// Tentar formato alternativo YYYY-MM-DD
			toTime, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid 'to' date format. Use ISO format (e.g., 2023-01-31T23:59:59Z) or YYYY-MM-DD"})
			}
			// Definir para fim do dia
			toTime = time.Date(toTime.Year(), toTime.Month(), toTime.Day(), 23, 59, 59, 999999999, toTime.Location())
		}
		to = toTime
	}

	// Suporte a startDate/endDate alternativo para from/to
	startDateStr := c.Query("startDate", "")
	endDateStr := c.Query("endDate", "")

	if fromStr == "" && startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			// Definir para início do dia
			from = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
			fromStr = startDateStr + "T00:00:00Z"
		}
	}

	if toStr == "" && endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			// Definir para fim do dia
			to = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
			toStr = endDateStr + "T23:59:59Z"
		}
	}

	// Parse de filtros de tempo
	timeFrom := c.Query("time_from", "")
	timeTo := c.Query("time_to", "")

	// Parse de outros filtros
	userID := c.Query("user_id", "")
	professionID := c.Query("profession_id", "")
	productID := c.Query("product_id", "")
	funnelID := c.Query("funnel_id", "")

	// Parse do parâmetro isActive
	var isActive *bool
	isActiveStr := c.Query("is_active", "")
	if isActiveStr != "" {
		isActiveVal := isActiveStr == "true"
		isActive = &isActiveVal
	}

	// Verificar se há filtro de período
	hasDateFilter := !from.IsZero() && !to.IsZero()

	// Se count_only for true, apenas obter a contagem
	if countOnly {
		// Tratamento específico para dashboard com períodos múltiplos
		periodsParam := c.Query("periods", "")
		period := c.Query("period", "false") == "true"
		allData := c.Query("all_data", "false") == "true"

		// Verificar se é para buscar todos os dados históricos
		if allData {
			// Buscar intervalo completo de datas das sessões
			firstDate, lastDate, err := h.sessionUseCase.GetSessionsDateRange()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao obter intervalo de datas de sessões: %v", err),
				})
			}

			// Normalizar as datas para formato de data apenas (sem horas)
			firstDateOnly := time.Date(firstDate.Year(), firstDate.Month(), firstDate.Day(), 0, 0, 0, 0, firstDate.Location())
			lastDateOnly := time.Date(lastDate.Year(), lastDate.Month(), lastDate.Day(), 0, 0, 0, 0, lastDate.Location())

			// Gerar array de todas as datas no intervalo
			dateRange := GenerateDateRange(firstDateOnly, lastDateOnly)
			result, err := h.sessionUseCase.CountSessionsByPeriods(dateRange)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Error counting sessions by periods: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods":    result,
				"start_date": firstDateOnly.Format("2006-01-02"),
				"end_date":   lastDateOnly.Format("2006-01-02"),
				"all_data":   true,
			})
		} else if period && hasDateFilter {
			// Gerar array de datas no intervalo from-to
			dateRange := GenerateDateRange(from, to)
			result, err := h.sessionUseCase.CountSessionsByPeriods(dateRange)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Error counting sessions by periods: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods": result,
				"from":    fromStr,
				"to":      toStr,
			})
		} else if periodsParam != "" {
			periods := strings.Split(periodsParam, ",")
			result, err := h.sessionUseCase.CountSessionsByPeriods(periods)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Error counting sessions by periods: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods": result,
			})
		}

		// Contagem normal
		count, err := h.sessionUseCase.CountSessions(from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Error counting sessions: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"count":     count,
			"from":      fromStr,
			"to":        toStr,
			"time_from": timeFrom,
			"time_to":   timeTo,
		})
	}

	// Buscar sessões com filtros
	sessions, total, err := h.sessionUseCase.GetSessions(
		c.Context(),
		page,
		limit,
		orderBy,
		from,
		to,
		timeFrom,
		timeTo,
		userID,
		professionID,
		productID,
		funnelID,
		isActive,
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error retrieving sessions: %v", err),
		})
	}

	// Garantir um array vazio em vez de null
	if sessions == nil {
		sessions = []entities.Session{}
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"sessions":      sessions,
		"page":          page,
		"limit":         limit,
		"total":         total,
		"totalPages":    totalPages,
		"sortBy":        sortBy,
		"sortDirection": sortDirection,
		"from":          fromStr,
		"to":            toStr,
		"time_from":     timeFrom,
		"time_to":       timeTo,
		"limitApplied":  hasDateFilter, // Indica se o limite foi aplicado (apenas com filtro de data)
	})
}

// GetActiveSessions retorna todas as sessões ativas
func (h *SessionHandler) GetActiveSessions(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'page' parameter"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'limit' parameter"})
	}

	// Parse de ordenação
	sortBy := c.Query("sortBy", "lastActivity")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	// Verificar se é para retornar apenas a contagem
	countOnly := c.Query("count_only", "false") == "true"

	if countOnly {
		// Para count_only, usamos o método normal com isActive = true
		isActiveVal := true
		count, err := h.sessionUseCase.CountSessions(time.Time{}, time.Time{}, "", "", "", "", "", "", &isActiveVal)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Error counting active sessions: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"count": count,
		})
	}

	// Buscar sessões ativas
	sessions, total, err := h.sessionUseCase.FindActiveSessions(page, limit, orderBy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error retrieving active sessions: %v", err),
		})
	}

	// Garantir um array vazio em vez de null
	if sessions == nil {
		sessions = []entities.Session{}
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"sessions":      sessions,
		"page":          page,
		"limit":         limit,
		"total":         total,
		"totalPages":    totalPages,
		"sortBy":        sortBy,
		"sortDirection": sortDirection,
	})
}

// GetSessionByID retorna uma sessão específica pelo ID
func (h *SessionHandler) GetSessionByID(c *fiber.Ctx) error {
	id := c.Params("id")

	// Validar UUID
	_, err := uuid.Parse(id)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid session ID format"})
	}

	session, err := h.sessionUseCase.FindSessionByID(context.Background(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Session not found"})
	}

	return c.JSON(session)
}
