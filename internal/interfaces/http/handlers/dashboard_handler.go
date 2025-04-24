package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"crypto/md5"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/gofiber/fiber/v2"
)

// DashboardHandler lida com requisições relacionadas a dashboards
type DashboardHandler struct {
	dashboardUseCase usecases.DashboardUseCase
}

// NewDashboardHandler cria uma nova instância de DashboardHandler
func NewDashboardHandler(dashboardUseCase usecases.DashboardUseCase) *DashboardHandler {
	return &DashboardHandler{
		dashboardUseCase: dashboardUseCase,
	}
}

// GetUnifiedDashboard retorna um dashboard unificado com métricas de diferentes fontes
// @Summary Retorna dados consolidados para o dashboard
// @Description Retorna métricas de sessões, leads e clientes, com comparação entre períodos e séries temporais
// @Tags dashboard
// @Accept json
// @Produce json
// @Param profession_id query string false "ID da profissão"
// @Param funnel_id query string false "ID do funil"
// @Param from query string false "Data inicial (formato: 2006-01-02)"
// @Param to query string false "Data final (formato: 2006-01-02)"
// @Param time_from query string false "Hora inicial (formato: 00:00)"
// @Param time_to query string false "Hora final (formato: 23:59)"
// @Param time_frame query string false "Granularidade da série temporal (hourly, daily, weekly, monthly)" default(daily)
// @Param product_id query string false "ID do produto"
// @Param user_id query string false "ID do usuário"
// @Param landingPage query string false "URL da página de destino"
// @Success 200 {object} map[string]interface{} "Dados consolidados do dashboard"
// @Failure 400 {object} map[string]interface{} "Erro de parâmetros"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /dashboard/unified [get]
func (h *DashboardHandler) GetUnifiedDashboard(c *fiber.Ctx) error {
	// Registrar momento inicial para métricas de performance
	startTime := time.Now()

	// Extrair parâmetros da query
	params := make(map[string]string)

	// Parâmetros de filtro básicos
	params["profession_id"] = c.Query("profession_id", "")
	params["funnel_id"] = c.Query("funnel_id", "")
	params["landingPage"] = c.Query("landingPage", "")
	if params["landingPage"] == "" {
		params["landing_page"] = c.Query("landing_page", "")
	}
	params["user_id"] = c.Query("user_id", "")
	params["product_id"] = c.Query("product_id", "")
	params["time_frame"] = c.Query("time_frame", "daily") // daily, weekly, monthly

	// Extrair parâmetros de data
	from := c.Query("from", "")
	to := c.Query("to", "")
	timeFrom := c.Query("time_from", "00:00")
	timeTo := c.Query("time_to", "23:59")

	// Adicionar datas ao mapa de parâmetros
	params["from"] = from
	params["to"] = to
	params["time_from"] = timeFrom
	params["time_to"] = timeTo

	// Validar datas
	if from == "" || to == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Os parâmetros 'from' e 'to' são obrigatórios",
		})
	}

	// Processar período atual
	currentFromDate, err := time.Parse("2006-01-02", from)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Formato de data inválido para 'from': %s", err.Error()),
		})
	}

	currentToDate, err := time.Parse("2006-01-02", to)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Formato de data inválido para 'to': %s", err.Error()),
		})
	}

	// Configurar período atual
	currentPeriod := usecases.DatePeriod{
		From:     currentFromDate,
		To:       currentToDate,
		TimeFrom: timeFrom,
		TimeTo:   timeTo,
	}

	// Calcular período anterior (mesmo número de dias, imediatamente anterior)
	daysDiff := currentToDate.Sub(currentFromDate).Hours() / 24
	previousToDate := currentFromDate.AddDate(0, 0, -1)
	previousFromDate := previousToDate.AddDate(0, 0, -int(daysDiff))

	// Configurar período anterior
	previousPeriod := usecases.DatePeriod{
		From:     previousFromDate,
		To:       previousToDate,
		TimeFrom: timeFrom,
		TimeTo:   timeTo,
	}

	// Obter dados otimizados do dashboard
	result, err := h.dashboardUseCase.GetUnifiedDashboard(params, currentPeriod, previousPeriod)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Erro ao obter dados do dashboard: %s", err.Error()),
		})
	}

	// Criar ETag baseado no conteúdo
	etagContent := fmt.Sprintf("%v", result)
	etag := fmt.Sprintf(`W/"%x"`, md5.Sum([]byte(etagContent)))

	// Verificar se o cliente já tem a versão mais recente
	if c.Get("If-None-Match") == etag {
		return c.SendStatus(fiber.StatusNotModified)
	}

	// Adicionar ETag no cabeçalho de resposta
	c.Set("ETag", etag)

	// Calcular tempo de execução
	executionTime := time.Since(startTime).Milliseconds()

	// Retornar resposta com métricas de performance
	return c.JSON(fiber.Map{
		"data": result,
		"performance": fiber.Map{
			"execution_time_ms": executionTime,
		},
	})
}

// Função auxiliar para criar metadados
func createMetadata(page, limit, total int) map[string]interface{} {
	return map[string]interface{}{
		"page":          page,
		"limit":         limit,
		"total":         total,
		"total_pages":   1,
		"has_next_page": false,
	}
}

// calculatePeriods calcula os períodos atual e anterior com base nos parâmetros da requisição
func calculatePeriods(c *fiber.Ctx) (usecases.DatePeriod, usecases.DatePeriod, error) {
	var currentPeriod, previousPeriod usecases.DatePeriod

	// Configurar valores padrão
	to := time.Now()
	from := to.AddDate(0, 0, -7) // Padrão: últimos 7 dias
	timeFrom := "00:00"
	timeTo := "23:59"

	// Obter parâmetros de data/hora da requisição
	if fromStr := c.Query("from"); fromStr != "" {
		parsedFrom, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return currentPeriod, previousPeriod, fmt.Errorf("formato de data inválido para 'from': %w", err)
		}
		from = parsedFrom
	}

	if toStr := c.Query("to"); toStr != "" {
		parsedTo, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return currentPeriod, previousPeriod, fmt.Errorf("formato de data inválido para 'to': %w", err)
		}
		to = parsedTo.Add(24*time.Hour - time.Second) // Final do dia
	}

	if tfStr := c.Query("time_from"); tfStr != "" {
		timeFrom = tfStr
	}

	if ttStr := c.Query("time_to"); ttStr != "" {
		timeTo = ttStr
	}

	// Calcular a duração do período atual
	duration := to.Sub(from)

	// Configurar período atual
	currentPeriod = usecases.DatePeriod{
		From:     from,
		To:       to,
		TimeFrom: timeFrom,
		TimeTo:   timeTo,
	}

	// Configurar período anterior (mesmo intervalo, imediatamente antes do período atual)
	previousPeriod = usecases.DatePeriod{
		From:     from.Add(-duration),
		To:       from.Add(-time.Second), // 1 segundo antes do início do período atual
		TimeFrom: timeFrom,
		TimeTo:   timeTo,
	}

	return currentPeriod, previousPeriod, nil
}

// TimeComponents é uma estrutura para armazenar componentes de tempo parseados
type TimeComponents struct {
	Hour   int
	Minute int
}

// ParseTimeString converte uma string de tempo (HH:MM) para componentes
func ParseTimeString(timeStr string) *TimeComponents {
	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return nil
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return nil
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return nil
	}

	return &TimeComponents{
		Hour:   hour,
		Minute: minute,
	}
}

// calculatePreviousPeriod determina o período anterior para comparação
func calculatePreviousPeriod(from, to time.Time, timeFrame string) (time.Time, time.Time) {
	duration := to.Sub(from)

	switch timeFrame {
	case "Daily":
		return from.AddDate(0, 0, -1), to.AddDate(0, 0, -1)
	case "Weekly":
		return from.AddDate(0, 0, -7), to.AddDate(0, 0, -7)
	case "Monthly":
		return from.AddDate(0, -1, 0), to.AddDate(0, -1, 0)
	case "Yearly":
		return from.AddDate(-1, 0, 0), to.AddDate(-1, 0, 0)
	default:
		// Usar a duração exata do período atual
		return from.Add(-duration), to.Add(-duration)
	}
}
