package handlers

import (
	"strconv"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/gofiber/fiber/v2"
)

// SurveyHandler lida com requisições relacionadas a pesquisas
type SurveyHandler struct {
	surveyUseCase *usecases.SurveyUseCase
}

// NewSurveyHandler cria uma nova instância de SurveyHandler
func NewSurveyHandler(surveyUseCase *usecases.SurveyUseCase) *SurveyHandler {
	return &SurveyHandler{
		surveyUseCase: surveyUseCase,
	}
}

// GetSurveys retorna todas as pesquisas com opção de filtros
// @Summary Retorna todas as pesquisas
// @Description Retorna todas as pesquisas com opção de filtros por funil e paginação
// @Tags surveys
// @Accept json
// @Produce json
// @Param page query int false "Página atual" default(1)
// @Param limit query int false "Itens por página" default(10)
// @Param funnel_id query int false "ID do funil"
// @Param survey_id query int false "ID da pesquisa"
// @Param include_funnel query bool false "Incluir dados do funil" default(false)
// @Success 200 {object} map[string]interface{} "Lista de pesquisas"
// @Failure 400 {object} map[string]interface{} "Erro de parâmetros"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /surveys [get]
func (h *SurveyHandler) GetSurveys(c *fiber.Ctx) error {
	// Obter parâmetros de query
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Parâmetro 'page' inválido"})
	}

	limit, err := strconv.Atoi(c.Query("limit", "10"))
	if err != nil || limit < 1 {
		return c.Status(400).JSON(fiber.Map{"error": "Parâmetro 'limit' inválido"})
	}

	// Obter filtros opcionais
	funnelID, _ := strconv.Atoi(c.Query("funnel_id", "0"))
	surveyID, _ := strconv.Atoi(c.Query("survey_id", "0"))
	includeFunnel := c.Query("include_funnel", "false") == "true"

	// Buscar pesquisas
	surveys, total, err := h.surveyUseCase.GetSurveys(page, limit, funnelID, surveyID, includeFunnel)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao buscar pesquisas: " + err.Error()})
	}

	// Retornar resposta
	return c.JSON(fiber.Map{
		"data":  surveys,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetSurveyMetrics retorna métricas agregadas de pesquisas com base nos filtros fornecidos
// @Summary Retorna métricas de pesquisas
// @Description Retorna métricas agregadas de todas as pesquisas disponíveis ou filtradas pelos parâmetros fornecidos
// @Tags surveys
// @Accept json
// @Produce json
// @Param data_inicio query string false "Data de início geral (ISO8601 com timezone)"
// @Param data_fim query string false "Data de fim geral (ISO8601 com timezone)"
// @Param lead_inicio query string false "Data de início para leads (ISO8601 com timezone)"
// @Param lead_fim query string false "Data de fim para leads (ISO8601 com timezone)"
// @Param pesquisa_inicio query string false "Data de início para respostas de pesquisa (ISO8601 com timezone)"
// @Param pesquisa_fim query string false "Data de fim para respostas de pesquisa (ISO8601 com timezone)"
// @Param venda_inicio query string false "Data de início para vendas (ISO8601 com timezone)"
// @Param venda_fim query string false "Data de fim para vendas (ISO8601 com timezone)"
// @Param profissao query int false "Filtrar por profissão"
// @Param funil query int false "Filtrar por funil"
// @Param pesquisa_id query int false "Filtrar por ID da pesquisa"
// @Success 200 {object} []map[string]interface{} "Métricas de pesquisas"
// @Failure 400 {object} map[string]interface{} "Erro de parâmetros"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /metrics/surveys [get]
func (h *SurveyHandler) GetSurveyMetrics(c *fiber.Ctx) error {
	// Obter parâmetros e filtros
	params := make(map[string]interface{})

	// Parâmetros de filtro de data
	dataInicioStr := c.Query("data_inicio", "")
	dataFimStr := c.Query("data_fim", "")
	leadInicioStr := c.Query("lead_inicio", "")
	leadFimStr := c.Query("lead_fim", "")
	pesquisaInicioStr := c.Query("pesquisa_inicio", "")
	pesquisaFimStr := c.Query("pesquisa_fim", "")
	vendaInicioStr := c.Query("venda_inicio", "")
	vendaFimStr := c.Query("venda_fim", "")

	// Converter datas para time.Time
	if dataInicioStr != "" {
		dataInicio, err := h.surveyUseCase.ParseDateParam(dataInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'data_inicio'. Use ISO8601."})
		}
		params["data_inicio"] = dataInicio
	}

	if dataFimStr != "" {
		dataFim, err := h.surveyUseCase.ParseDateParam(dataFimStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'data_fim'. Use ISO8601."})
		}
		// Ajustar para o final do dia, se for uma data simples
		if len(dataFimStr) <= 10 {
			dataFim = time.Date(dataFim.Year(), dataFim.Month(), dataFim.Day(), 23, 59, 59, 999999999, dataFim.Location())
		}
		params["data_fim"] = dataFim
	}

	if leadInicioStr != "" {
		leadInicio, err := h.surveyUseCase.ParseDateParam(leadInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'lead_inicio'. Use ISO8601."})
		}
		params["lead_inicio"] = leadInicio
	}

	if leadFimStr != "" {
		leadFim, err := h.surveyUseCase.ParseDateParam(leadFimStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'lead_fim'. Use ISO8601."})
		}
		// Ajustar para o final do dia, se for uma data simples
		if len(leadFimStr) <= 10 {
			leadFim = time.Date(leadFim.Year(), leadFim.Month(), leadFim.Day(), 23, 59, 59, 999999999, leadFim.Location())
		}
		params["lead_fim"] = leadFim
	}

	if pesquisaInicioStr != "" {
		pesquisaInicio, err := h.surveyUseCase.ParseDateParam(pesquisaInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'pesquisa_inicio'. Use ISO8601."})
		}
		params["pesquisa_inicio"] = pesquisaInicio
	}

	if pesquisaFimStr != "" {
		pesquisaFim, err := h.surveyUseCase.ParseDateParam(pesquisaFimStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'pesquisa_fim'. Use ISO8601."})
		}
		// Ajustar para o final do dia, se for uma data simples
		if len(pesquisaFimStr) <= 10 {
			pesquisaFim = time.Date(pesquisaFim.Year(), pesquisaFim.Month(), pesquisaFim.Day(), 23, 59, 59, 999999999, pesquisaFim.Location())
		}
		params["pesquisa_fim"] = pesquisaFim
	}

	if vendaInicioStr != "" {
		vendaInicio, err := h.surveyUseCase.ParseDateParam(vendaInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'venda_inicio'. Use ISO8601."})
		}
		params["venda_inicio"] = vendaInicio
	}

	if vendaFimStr != "" {
		vendaFim, err := h.surveyUseCase.ParseDateParam(vendaFimStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'venda_fim'. Use ISO8601."})
		}
		// Ajustar para o final do dia, se for uma data simples
		if len(vendaFimStr) <= 10 {
			vendaFim = time.Date(vendaFim.Year(), vendaFim.Month(), vendaFim.Day(), 23, 59, 59, 999999999, vendaFim.Location())
		}
		params["venda_fim"] = vendaFim
	}

	// Outros filtros
	profissaoID, err := strconv.Atoi(c.Query("profissao", "0"))
	if err == nil && profissaoID > 0 {
		params["profissao"] = profissaoID
	}

	funilID, err := strconv.Atoi(c.Query("funil", "0"))
	if err == nil && funilID > 0 {
		params["funil"] = funilID
	}

	pesquisaID, err := strconv.ParseInt(c.Query("pesquisa_id", "0"), 10, 64)
	if err == nil && pesquisaID > 0 {
		params["pesquisa_id"] = pesquisaID
	}

	// Buscar métricas de pesquisa
	metrics, err := h.surveyUseCase.GetSurveyMetrics(params)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao buscar métricas de pesquisa: " + err.Error()})
	}

	// Retornar resposta
	return c.JSON(metrics)
}

// GetSurveyDetails retorna detalhes de uma pesquisa específica, incluindo análise por pergunta e resposta
// @Summary Retorna detalhes de uma pesquisa
// @Description Retorna análise detalhada de uma pesquisa específica, com drill-down por pergunta e opção de resposta
// @Tags surveys
// @Accept json
// @Produce json
// @Param id path int true "ID da pesquisa"
// @Param data_inicio query string false "Data de início geral (ISO8601 com timezone)"
// @Param data_fim query string false "Data de fim geral (ISO8601 com timezone)"
// @Param pesquisa_inicio query string false "Data de início para respostas de pesquisa (ISO8601 com timezone)"
// @Param pesquisa_fim query string false "Data de fim para respostas de pesquisa (ISO8601 com timezone)"
// @Param venda_inicio query string false "Data de início para vendas (ISO8601 com timezone)"
// @Param venda_fim query string false "Data de fim para vendas (ISO8601 com timezone)"
// @Success 200 {object} []map[string]interface{} "Detalhes da pesquisa"
// @Failure 400 {object} map[string]interface{} "Erro de parâmetros"
// @Failure 404 {object} map[string]interface{} "Pesquisa não encontrada"
// @Failure 500 {object} map[string]interface{} "Erro interno do servidor"
// @Router /metrics/surveys/{id} [get]
func (h *SurveyHandler) GetSurveyDetails(c *fiber.Ctx) error {
	// Obter ID da pesquisa
	surveyID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || surveyID <= 0 {
		return c.Status(400).JSON(fiber.Map{"error": "ID de pesquisa inválido"})
	}

	// Obter parâmetros e filtros
	params := make(map[string]interface{})

	// Parâmetros de filtro de data
	dataInicioStr := c.Query("data_inicio", "")
	dataFimStr := c.Query("data_fim", "")
	pesquisaInicioStr := c.Query("pesquisa_inicio", "")
	pesquisaFimStr := c.Query("pesquisa_fim", "")
	vendaInicioStr := c.Query("venda_inicio", "")
	vendaFimStr := c.Query("venda_fim", "")

	// Converter datas para time.Time
	if dataInicioStr != "" {
		dataInicio, err := h.surveyUseCase.ParseDateParam(dataInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'data_inicio'. Use ISO8601."})
		}
		params["data_inicio"] = dataInicio
	}

	if dataFimStr != "" {
		dataFim, err := h.surveyUseCase.ParseDateParam(dataFimStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'data_fim'. Use ISO8601."})
		}
		// Ajustar para o final do dia, se for uma data simples
		if len(dataFimStr) <= 10 {
			dataFim = time.Date(dataFim.Year(), dataFim.Month(), dataFim.Day(), 23, 59, 59, 999999999, dataFim.Location())
		}
		params["data_fim"] = dataFim
	}

	if pesquisaInicioStr != "" {
		pesquisaInicio, err := h.surveyUseCase.ParseDateParam(pesquisaInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'pesquisa_inicio'. Use ISO8601."})
		}
		params["pesquisa_inicio"] = pesquisaInicio
	}

	if pesquisaFimStr != "" {
		pesquisaFim, err := h.surveyUseCase.ParseDateParam(pesquisaFimStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'pesquisa_fim'. Use ISO8601."})
		}
		// Ajustar para o final do dia, se for uma data simples
		if len(pesquisaFimStr) <= 10 {
			pesquisaFim = time.Date(pesquisaFim.Year(), pesquisaFim.Month(), pesquisaFim.Day(), 23, 59, 59, 999999999, pesquisaFim.Location())
		}
		params["pesquisa_fim"] = pesquisaFim
	}

	if vendaInicioStr != "" {
		vendaInicio, err := h.surveyUseCase.ParseDateParam(vendaInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'venda_inicio'. Use ISO8601."})
		}
		params["venda_inicio"] = vendaInicio
	}

	if vendaFimStr != "" {
		vendaFim, err := h.surveyUseCase.ParseDateParam(vendaFimStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Formato de data inválido para 'venda_fim'. Use ISO8601."})
		}
		// Ajustar para o final do dia, se for uma data simples
		if len(vendaFimStr) <= 10 {
			vendaFim = time.Date(vendaFim.Year(), vendaFim.Month(), vendaFim.Day(), 23, 59, 59, 999999999, vendaFim.Location())
		}
		params["venda_fim"] = vendaFim
	}

	// Buscar detalhes da pesquisa
	details, err := h.surveyUseCase.GetSurveyDetails(surveyID, params)
	if err != nil {
		// Verificar se é erro de "não encontrado"
		if err.Error() == "pesquisa não encontrada: record not found" {
			return c.Status(404).JSON(fiber.Map{"error": "Pesquisa não encontrada"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "Erro ao buscar detalhes da pesquisa: " + err.Error()})
	}

	// Retornar resposta
	return c.JSON(details)
}
