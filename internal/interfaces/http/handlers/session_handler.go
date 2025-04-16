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
	countOnly := c.Query("count_only") == "true"

	// Obter todos os parâmetros uma única vez para evitar inconsistências
	landingPage := c.Query("landingPage", "")
	if landingPage == "" {
		landingPage = c.Query("landing_page", "")
	}

	// Extract funnel ID and profession ID
	funnelID := c.Query("funnel_id", "")
	professionID := c.Query("profession_id", "")

	// Obter localização de Brasília
	brazilLocation := GetBrasilLocation()

	// Parse de parâmetros básicos
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'page' parameter"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid 'limit' parameter"})
	}

	// Parse de ordenação
	sortBy := c.Query("sortBy", "sessionStart")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	// Parse period and periods parameters
	periodEnabled := c.Query("period", "false") == "true"
	periodsParam := c.Query("periods", "")

	// Parse de filtros de data (MOVIDO PARA ANTES DO BLOCO IF periodEnabled)
	fromTime := time.Time{}
	toTime := time.Now().In(brazilLocation)

	// Parse from e to das query params
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	if fromStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			// Tentar formato alternativo YYYY-MM-DD
			parsedTime, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid 'from' date format. Use ISO format (e.g., 2023-01-01T00:00:00Z) or YYYY-MM-DD"})
			}
			// Definir para início do dia no horário de Brasília
			fromTime = time.Date(parsedTime.Year(), parsedTime.Month(), parsedTime.Day(), 0, 0, 0, 0, brazilLocation)
		} else {
			// Converter para horário de Brasília
			fromTime = parsedTime.In(brazilLocation)
		}
	}

	if toStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			// Tentar formato alternativo YYYY-MM-DD
			parsedTime, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				return c.Status(400).JSON(fiber.Map{"error": "Invalid 'to' date format. Use ISO format (e.g., 2023-01-31T23:59:59Z) or YYYY-MM-DD"})
			}
			// Definir para fim do dia no horário de Brasília
			toTime = time.Date(parsedTime.Year(), parsedTime.Month(), parsedTime.Day(), 23, 59, 59, 999999999, brazilLocation)
		} else {
			// Converter para horário de Brasília
			toTime = parsedTime.In(brazilLocation)
		}
	}

	// Suporte a startDate/endDate alternativo para from/to
	startDateStr := c.Query("startDate", "")
	endDateStr := c.Query("endDate", "")

	if fromStr == "" && startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			// Definir para início do dia no horário de Brasília
			fromTime = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, brazilLocation)
		}
	}

	if toStr == "" && endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			// Definir para fim do dia no horário de Brasília
			toTime = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, brazilLocation)
		}
	}

	// Verificar se há filtro de período
	hasDateFilter := !fromTime.IsZero() && !toTime.IsZero()

	// Se period estiver habilitado, contar as sessões por períodos
	if periodEnabled {
		var periods []string
		var results map[string]int64
		var err error

		// Verificar se temos períodos específicos ou se devemos usar intervalo de datas
		if periodsParam != "" {
			// Usar split por vírgula para períodos específicos
			periods = strings.Split(periodsParam, ",")
		} else if hasDateFilter {
			// Gerar períodos a partir do intervalo de datas from-to
			periods = GenerateDateRange(fromTime, toTime)
		} else {
			// Sem períodos e sem datas, usar data atual
			today := time.Now().In(brazilLocation)
			todayStr := today.Format("2006-01-02")
			periods = []string{todayStr}
		}

		// Executar a consulta com os períodos determinados
		results, err = h.sessionUseCase.CountSessionsByPeriods(periods, landingPage, funnelID, professionID)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to count sessions by periods",
			})
		}

		return c.JSON(fiber.Map{
			"periods":       results,
			"landingPage":   landingPage,
			"funnel_id":     funnelID,
			"profession_id": professionID,
			"from":          fromStr,
			"to":            toStr,
		})
	}

	// Parse de filtros de tempo
	timeFrom := c.Query("time_from", "")
	timeTo := c.Query("time_to", "")

	// Parse de outros filtros
	userID := c.Query("user_id", "")
	productID := c.Query("product_id", "")

	// Parse do parâmetro isActive
	var isActive *bool
	isActiveStr := c.Query("is_active", "")
	if isActiveStr != "" {
		isActiveVal := isActiveStr == "true"
		isActive = &isActiveVal
	}

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

			result, err := h.sessionUseCase.CountSessionsByPeriods(dateRange, landingPage, funnelID, professionID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Error counting sessions by periods: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods":       result,
				"start_date":    firstDateOnly.Format("2006-01-02"),
				"end_date":      lastDateOnly.Format("2006-01-02"),
				"all_data":      true,
				"landingPage":   landingPage,
				"funnel_id":     funnelID,
				"profession_id": professionID,
			})
		} else if period && hasDateFilter {
			// Gerar array de datas no intervalo from-to
			dateRange := GenerateDateRange(fromTime, toTime)

			result, err := h.sessionUseCase.CountSessionsByPeriods(dateRange, landingPage, funnelID, professionID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Error counting sessions by periods: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods":       result,
				"from":          fromStr,
				"to":            toStr,
				"landingPage":   landingPage,
				"funnel_id":     funnelID,
				"profession_id": professionID,
			})
		} else if periodsParam != "" {
			periods := strings.Split(periodsParam, ",")
			result, err := h.sessionUseCase.CountSessionsByPeriods(periods, landingPage, funnelID, professionID)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Error counting sessions by periods: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods":       result,
				"landingPage":   landingPage,
				"funnel_id":     funnelID,
				"profession_id": professionID,
			})
		}

		// Contagem normal
		count, err := h.sessionUseCase.CountSessions(fromTime, toTime, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive, landingPage)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Error counting sessions: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"count":     count,
			"from":      fromTime.Format(time.RFC3339),
			"to":        toTime.Format(time.RFC3339),
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
		fromTime,
		toTime,
		timeFrom,
		timeTo,
		userID,
		professionID,
		productID,
		funnelID,
		isActive,
		landingPage,
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
		"from":          fromTime.Format(time.RFC3339),
		"to":            toTime.Format(time.RFC3339),
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

	// Obter parâmetro landingPage com suporte para ambos os formatos
	landingPage := c.Query("landingPage", "")
	if landingPage == "" {
		landingPage = c.Query("landing_page", "")
	}

	// Verificar se é para retornar apenas a contagem
	countOnly := c.Query("count_only", "false") == "true"

	if countOnly {
		// Para count_only, usamos o método normal com isActive = true
		isActiveVal := true
		count, err := h.sessionUseCase.CountSessions(time.Time{}, time.Time{}, "", "", "", "", "", "", &isActiveVal, landingPage)
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
	sessions, total, err := h.sessionUseCase.FindActiveSessions(page, limit, orderBy, landingPage)
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
		"landingPage":   landingPage,
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
