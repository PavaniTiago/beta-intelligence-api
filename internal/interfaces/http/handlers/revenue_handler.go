package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/gofiber/fiber/v2"
)

type RevenueHandler struct {
	revenueUseCase usecases.RevenueUseCase
}

func NewRevenueHandler(revenueUseCase usecases.RevenueUseCase) *RevenueHandler {
	return &RevenueHandler{
		revenueUseCase: revenueUseCase,
	}
}

// GetUnifiedDataGeneral retorna dados gerais unificados de leads e faturamento com comparação
func (h *RevenueHandler) GetUnifiedDataGeneral(c *fiber.Ctx) error {
	// Parse dos parâmetros de data
	currentFrom, currentTo, previousFrom, previousTo, err := h.parseDateParamsWithComparison(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Verificar se é dia único para incluir dados por hora
	isSingleDay := h.isSingleDay(currentFrom, currentTo)

	// Buscar dados de comparação
	data, err := h.revenueUseCase.GetRevenueComparisonGeneral(currentFrom, currentTo, previousFrom, previousTo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Erro ao buscar dados de comparação gerais: " + err.Error(),
		})
	}

	// Se for dia único, buscar dados por hora
	if isSingleDay {
		hourlyData, err := h.revenueUseCase.GetHourlyRevenueData(currentFrom, nil)
		if err == nil {
			data.HourlyData = hourlyData
		}

		// Buscar dados por hora do período anterior
		previousHourlyData, err := h.revenueUseCase.GetHourlyRevenueData(previousFrom, nil)
		if err == nil {
			data.PreviousPeriodData.HourlyData = previousHourlyData
		}
	}

	// Preparar filtros aplicados para resposta
	appliedFilters := fiber.Map{
		"current_period": fiber.Map{
			"from": currentFrom.Format("2006-01-02"),
			"to":   currentTo.Format("2006-01-02"),
		},
		"previous_period": fiber.Map{
			"from": previousFrom.Format("2006-01-02"),
			"to":   previousTo.Format("2006-01-02"),
		},
		"is_single_day": isSingleDay,
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"data":            data,
		"applied_filters": appliedFilters,
	})
}

// GetUnifiedDataByProfession retorna dados unificados de leads e faturamento por profissão com comparação
func (h *RevenueHandler) GetUnifiedDataByProfession(c *fiber.Ctx) error {
	// Parse dos parâmetros de data
	currentFrom, currentTo, previousFrom, previousTo, err := h.parseDateParamsWithComparison(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Parse dos IDs de profissão
	professionIDs, err := h.parseProfessionIDs(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Debug: log dos filtros recebidos
	fmt.Printf("GetUnifiedDataByProfession - ProfessionIDs recebidos: %v\n", professionIDs)
	fmt.Printf("GetUnifiedDataByProfession - Query profession_ids: %s\n", c.Query("profession_ids", ""))

	// Verificar se é dia único para incluir dados por hora
	isSingleDay := h.isSingleDay(currentFrom, currentTo)

	// Buscar dados de comparação por profissão
	data, err := h.revenueUseCase.GetRevenueComparisonByProfession(currentFrom, currentTo, previousFrom, previousTo, professionIDs)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Erro ao buscar dados de comparação por profissão: " + err.Error(),
		})
	}

	// Se for dia único, buscar dados por hora UMA VEZ para todas as profissões
	if isSingleDay && len(data) > 0 {
		// Buscar dados por hora do período atual
		hourlyData, err := h.revenueUseCase.GetHourlyRevenueData(currentFrom, professionIDs)
		if err == nil {
			// Aplicar os mesmos dados por hora para todas as profissões
			for i := range data {
				data[i].HourlyData = hourlyData
			}
		}

		// Buscar dados por hora do período anterior
		previousHourlyData, err := h.revenueUseCase.GetHourlyRevenueData(previousFrom, professionIDs)
		if err == nil {
			// Aplicar os mesmos dados por hora para todas as profissões
			for i := range data {
				data[i].PreviousPeriodData.HourlyData = previousHourlyData
			}
		}
	}

	// Preparar filtros aplicados para resposta
	appliedFilters := fiber.Map{
		"current_period": fiber.Map{
			"from": currentFrom.Format("2006-01-02"),
			"to":   currentTo.Format("2006-01-02"),
		},
		"previous_period": fiber.Map{
			"from": previousFrom.Format("2006-01-02"),
			"to":   previousTo.Format("2006-01-02"),
		},
		"is_single_day": isSingleDay,
	}
	if len(professionIDs) > 0 {
		appliedFilters["profession_ids"] = professionIDs
	}

	return c.JSON(fiber.Map{
		"success":         true,
		"data":            data,
		"applied_filters": appliedFilters,
	})
}

// parseDateParamsWithComparison extrai e calcula períodos atual e anterior
func (h *RevenueHandler) parseDateParamsWithComparison(c *fiber.Ctx) (time.Time, time.Time, time.Time, time.Time, error) {
	var currentFrom, currentTo, previousFrom, previousTo time.Time
	var err error

	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	// Se não forem fornecidas datas, usar hoje vs ontem
	if fromStr == "" || toStr == "" {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

		currentFrom = today
		currentTo = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

		// Período anterior: ontem
		previousFrom = today.AddDate(0, 0, -1)
		previousTo = time.Date(previousFrom.Year(), previousFrom.Month(), previousFrom.Day(), 23, 59, 59, 999999999, previousFrom.Location())
	} else {
		// Parse das datas fornecidas
		currentFrom, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			currentFrom, err = time.Parse("2006-01-02T15:04:05Z07:00", fromStr)
			if err != nil {
				return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
			}
		}

		currentTo, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			currentTo, err = time.Parse("2006-01-02T15:04:05Z07:00", toStr)
			if err != nil {
				return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
			}
		}

		// Se apenas a data foi fornecida, definir para o final do dia
		if len(toStr) == 10 { // formato YYYY-MM-DD
			currentTo = time.Date(currentTo.Year(), currentTo.Month(), currentTo.Day(), 23, 59, 59, 999999999, currentTo.Location())
		}

		// Calcular período anterior (mesmo número de dias, imediatamente anterior)
		daysDiff := int(currentTo.Sub(currentFrom).Hours() / 24)
		previousTo = currentFrom.AddDate(0, 0, -1)
		previousFrom = previousTo.AddDate(0, 0, -daysDiff)

		// Ajustar previousTo para incluir o dia completo
		previousTo = time.Date(previousTo.Year(), previousTo.Month(), previousTo.Day(), 23, 59, 59, 999999999, previousTo.Location())
	}

	return currentFrom, currentTo, previousFrom, previousTo, nil
}

// isSingleDay verifica se o período é de um único dia
func (h *RevenueHandler) isSingleDay(from, to time.Time) bool {
	return from.Year() == to.Year() && from.Month() == to.Month() && from.Day() == to.Day()
}

// parseDateParams extrai e valida os parâmetros de data da query string (método original mantido para compatibilidade)
func (h *RevenueHandler) parseDateParams(c *fiber.Ctx) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error

	fromStr := c.Query("from", "")
	toStr := c.Query("to", "")

	if fromStr != "" {
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			from, err = time.Parse("2006-01-02T15:04:05Z07:00", fromStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
		}
	}

	if toStr != "" {
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			to, err = time.Parse("2006-01-02T15:04:05Z07:00", toStr)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
		}
		// Se apenas a data foi fornecida, definir para o final do dia
		if len(toStr) == 10 { // formato YYYY-MM-DD
			to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())
		}
	}

	return from, to, nil
}

// parseProfessionIDs extrai e valida os IDs de profissão da query string
func (h *RevenueHandler) parseProfessionIDs(c *fiber.Ctx) ([]int, error) {
	professionIDsStr := c.Query("profession_ids", "")
	if professionIDsStr == "" {
		return nil, nil
	}

	professionIDsStrSlice := strings.Split(professionIDsStr, ",")
	professionIDs := make([]int, 0, len(professionIDsStrSlice))

	for _, idStr := range professionIDsStrSlice {
		id, err := strconv.Atoi(strings.TrimSpace(idStr))
		if err != nil {
			return nil, err
		}
		professionIDs = append(professionIDs, id)
	}

	return professionIDs, nil
}
