package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userUseCase *usecases.UserUseCase
	userRepo    *repositories.UserRepository
}

func NewUserHandler(userUseCase *usecases.UserUseCase, userRepo *repositories.UserRepository) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
		userRepo:    userRepo,
	}
}

func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
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

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	// Parse date filters
	from := time.Time{}
	to := time.Now()

	// Parse from and to dates from query params
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	if fromStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'from' date format. Use ISO format (e.g., 2023-01-01T00:00:00Z)"})
		}
		from = fromTime
	}

	if toStr != "" {
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'to' date format. Use ISO format (e.g., 2023-01-31T23:59:59Z)"})
		}
		to = toTime
	}

	// Get time filters
	timeFrom := c.Query("time_from", "")
	timeTo := c.Query("time_to", "")

	// Verificar se há filtro de período
	hasDateFilter := !from.IsZero() && !to.IsZero()

	// Se count_only for true, apenas obter a contagem
	if countOnly {
		count, err := h.userRepo.CountUsers(from, to, timeFrom, timeTo)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Erro ao contar usuários: %v", err),
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

	users, total, err := h.userUseCase.GetUsers(c.Context(), page, limit, orderBy, from, to, timeFrom, timeTo)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve users"})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"users":         users,
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

func (h *UserHandler) GetLeads(c *fiber.Ctx) error {
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

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	// Parse date filters
	from := time.Time{}
	to := time.Now()

	// Parse from and to dates from query params
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	if fromStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'from' date format. Use ISO format (e.g., 2023-01-01T00:00:00Z)"})
		}
		from = fromTime
	}

	if toStr != "" {
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'to' date format. Use ISO format (e.g., 2023-01-31T23:59:59Z)"})
		}
		to = toTime
	}

	// Get time filters
	timeFrom := c.Query("time_from", "")
	timeTo := c.Query("time_to", "")

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
			// Buscar intervalo completo de datas dos leads
			firstDate, lastDate, err := h.userRepo.GetLeadsDateRange()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao obter intervalo de datas de leads: %v", err),
				})
			}

			// Normalizar as datas para formato de data apenas (sem horas)
			firstDateOnly := time.Date(firstDate.Year(), firstDate.Month(), firstDate.Day(), 0, 0, 0, 0, firstDate.Location())
			lastDateOnly := time.Date(lastDate.Year(), lastDate.Month(), lastDate.Day(), 0, 0, 0, 0, lastDate.Location())

			// Gerar array de todas as datas no intervalo
			dateRange := GenerateDateRange(firstDateOnly, lastDateOnly)
			result, err := h.userRepo.CountLeadsByPeriods(dateRange)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao contar leads por períodos: %v", err),
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
			result, err := h.userRepo.CountLeadsByPeriods(dateRange)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao contar leads por períodos: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods": result,
				"from":    fromStr,
				"to":      toStr,
			})
		} else if periodsParam != "" {
			periods := strings.Split(periodsParam, ",")
			result, err := h.userRepo.CountLeadsByPeriods(periods)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao contar leads por períodos: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods": result,
			})
		}

		count, err := h.userRepo.CountLeads(from, to, timeFrom, timeTo)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Erro ao contar leads: %v", err),
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

	leads, total, err := h.userRepo.FindLeads(page, limit, orderBy, from, to, timeFrom, timeTo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Erro ao buscar leads: %v", err),
		})
	}

	if leads == nil {
		leads = []entities.User{}
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"leads":         leads,
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

func (h *UserHandler) GetClients(c *fiber.Ctx) error {
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

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	// Parse date filters
	from := time.Time{}
	to := time.Now()

	// Parse from and to dates from query params
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	if fromStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'from' date format. Use ISO format (e.g., 2023-01-01T00:00:00Z)"})
		}
		from = fromTime
	}

	if toStr != "" {
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'to' date format. Use ISO format (e.g., 2023-01-31T23:59:59Z)"})
		}
		to = toTime
	}

	// Get time filters
	timeFrom := c.Query("time_from", "")
	timeTo := c.Query("time_to", "")

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
			// Buscar intervalo completo de datas dos clientes
			firstDate, lastDate, err := h.userRepo.GetClientsDateRange()
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao obter intervalo de datas de clientes: %v", err),
				})
			}

			// Normalizar as datas para formato de data apenas (sem horas)
			firstDateOnly := time.Date(firstDate.Year(), firstDate.Month(), firstDate.Day(), 0, 0, 0, 0, firstDate.Location())
			lastDateOnly := time.Date(lastDate.Year(), lastDate.Month(), lastDate.Day(), 0, 0, 0, 0, lastDate.Location())

			// Gerar array de todas as datas no intervalo
			dateRange := GenerateDateRange(firstDateOnly, lastDateOnly)
			result, err := h.userRepo.CountClientsByPeriods(dateRange)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao contar clientes por períodos: %v", err),
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
			result, err := h.userRepo.CountClientsByPeriods(dateRange)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao contar clientes por períodos: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods": result,
				"from":    fromStr,
				"to":      toStr,
			})
		} else if periodsParam != "" {
			periods := strings.Split(periodsParam, ",")
			result, err := h.userRepo.CountClientsByPeriods(periods)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao contar clientes por períodos: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"periods": result,
			})
		}

		count, err := h.userRepo.CountClients(from, to, timeFrom, timeTo)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Erro ao contar clientes: %v", err),
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

	clients, total, err := h.userRepo.FindClients(page, limit, orderBy, from, to, timeFrom, timeTo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao buscar clientes",
		})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"clients":       clients,
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

func (h *UserHandler) GetAnonymous(c *fiber.Ctx) error {
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

	sortBy := c.Query("sortBy", "created_at")
	sortDirection := c.Query("sortDirection", "desc")
	orderBy := fmt.Sprintf("%s %s", sortBy, sortDirection)

	// Parse date filters
	from := time.Time{}
	to := time.Now()

	// Parse from and to dates from query params
	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	if fromStr != "" {
		fromTime, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'from' date format. Use ISO format (e.g., 2023-01-01T00:00:00Z)"})
		}
		from = fromTime
	}

	if toStr != "" {
		toTime, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid 'to' date format. Use ISO format (e.g., 2023-01-31T23:59:59Z)"})
		}
		to = toTime
	}

	// Get time filters
	timeFrom := c.Query("time_from", "")
	timeTo := c.Query("time_to", "")

	// Verificar se há filtro de período
	hasDateFilter := !from.IsZero() && !to.IsZero()

	// Se count_only for true, apenas obter a contagem
	if countOnly {
		count, err := h.userRepo.CountAnonymous(from, to, timeFrom, timeTo)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Erro ao contar usuários anônimos: %v", err),
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

	anonymous, total, err := h.userRepo.FindAnonymous(page, limit, orderBy, from, to, timeFrom, timeTo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Erro ao buscar usuários anônimos",
		})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(fiber.Map{
		"anonymous":     anonymous,
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
