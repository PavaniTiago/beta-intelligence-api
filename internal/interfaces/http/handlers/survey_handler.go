package handlers

import (
	"fmt"
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

// calculateDateParams calcula todos os parâmetros de data a partir de vendas_inicio
func (h *SurveyHandler) calculateDateParams(vendasInicioStr string) (map[string]time.Time, error) {
	dateParams := make(map[string]time.Time)

	// Parse vendas_inicio
	vendasInicio, err := h.surveyUseCase.ParseDateParam(vendasInicioStr)
	if err != nil {
		return nil, fmt.Errorf("formato de data inválido para vendas: %v", err)
	}

	// Verificar se o horário é 20:30 para compatibilidade
	if vendasInicio.Hour() != 20 || vendasInicio.Minute() != 30 {
		// Talvez seja a URL codificada - verificar se a data existe mas o horário está errado
		fmt.Printf("Horário inválido recebido: %s (%d:%d). Normalizando para 20:30.\n",
			vendasInicioStr, vendasInicio.Hour(), vendasInicio.Minute())
	}

	// Normalizar para garantir que estamos usando o horário 20:30:00, INDEPENDENTE do que foi enviado
	dataBase := time.Date(vendasInicio.Year(), vendasInicio.Month(), vendasInicio.Day(), 0, 0, 0, 0, vendasInicio.Location())

	// 1. vendas_inicio = dataBase às 20:30
	vendasInicioNorm := time.Date(dataBase.Year(), dataBase.Month(), dataBase.Day(), 20, 30, 0, 0, dataBase.Location())
	dateParams["venda_inicio"] = vendasInicioNorm

	// 2. vendas_fim = dataBase às 23:59:59
	vendasFim := time.Date(dataBase.Year(), dataBase.Month(), dataBase.Day(), 23, 59, 59, 999999999, dataBase.Location())
	dateParams["venda_fim"] = vendasFim

	// 3. pesquisa_fim = dataBase às 20:00
	pesquisaFim := time.Date(dataBase.Year(), dataBase.Month(), dataBase.Day(), 20, 0, 0, 0, dataBase.Location())
	dateParams["pesquisa_fim"] = pesquisaFim

	// 4. pesquisa_inicio = dataBase - 7 dias, às 20:00
	pesquisaInicio := dataBase.AddDate(0, 0, -7)
	pesquisaInicio = time.Date(pesquisaInicio.Year(), pesquisaInicio.Month(), pesquisaInicio.Day(), 20, 0, 0, 0, pesquisaInicio.Location())
	dateParams["pesquisa_inicio"] = pesquisaInicio

	return dateParams, nil
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
// @Param venda_inicio query string false "Data de início para vendas (ISO8601 com timezone)"
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
	vendaInicioStr := c.Query("venda_inicio", "")

	// Se data de venda_inicio fornecida, calcular todos os parâmetros de data
	if vendaInicioStr != "" {
		dateParams, err := h.calculateDateParams(vendaInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Horário de início de vendas inválido. As vendas sempre iniciam às 20:30."})
		}

		// Adicionar todos os parâmetros de data ao mapa de parâmetros
		for key, value := range dateParams {
			params[key] = value
		}
	} else {
		// Se nenhuma data foi fornecida, use a terça-feira atual ou a terça-feira anterior
		dataEscolhida := time.Now()
		// Ajustar para a terça-feira atual ou anterior
		for dataEscolhida.Weekday() != time.Tuesday {
			dataEscolhida = dataEscolhida.AddDate(0, 0, -1)
		}

		// Formatar como string ISO8601
		vendaInicioStr = time.Date(dataEscolhida.Year(), dataEscolhida.Month(), dataEscolhida.Day(), 20, 30, 0, 0, dataEscolhida.Location()).Format(time.RFC3339)

		// Calcular parâmetros de data
		dateParams, err := h.calculateDateParams(vendaInicioStr)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao calcular datas padrão"})
		}

		// Adicionar todos os parâmetros de data ao mapa de parâmetros
		for key, value := range dateParams {
			params[key] = value
		}
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
// @Param venda_inicio query string false "Data de início para vendas (ISO8601 com timezone)"
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

	// Adicionar o surveyID aos parâmetros
	params["pesquisa_id"] = surveyID

	// Parâmetros de filtro de data - usar a mesma lógica que GetSurveyMetrics
	vendaInicioStr := c.Query("venda_inicio", "")

	// Se data de venda_inicio fornecida, calcular todos os parâmetros de data
	if vendaInicioStr != "" {
		dateParams, err := h.calculateDateParams(vendaInicioStr)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Horário de início de vendas inválido. As vendas sempre iniciam às 20:30."})
		}

		// Adicionar todos os parâmetros de data ao mapa de parâmetros
		for key, value := range dateParams {
			params[key] = value
		}
	} else {
		// Se nenhuma data foi fornecida, use a terça-feira atual ou a terça-feira anterior
		dataEscolhida := time.Now()
		// Ajustar para a terça-feira atual ou anterior
		for dataEscolhida.Weekday() != time.Tuesday {
			dataEscolhida = dataEscolhida.AddDate(0, 0, -1)
		}

		// Formatar como string ISO8601
		vendaInicioStr = time.Date(dataEscolhida.Year(), dataEscolhida.Month(), dataEscolhida.Day(), 20, 30, 0, 0, dataEscolhida.Location()).Format(time.RFC3339)

		// Calcular parâmetros de data
		dateParams, err := h.calculateDateParams(vendaInicioStr)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Erro ao calcular datas padrão"})
		}

		// Adicionar todos os parâmetros de data ao mapa de parâmetros
		for key, value := range dateParams {
			params[key] = value
		}
	}

	// Log dos parâmetros para debug
	fmt.Printf("GetSurveyDetails - Survey ID: %d, Parameters: %+v\n", surveyID, params)

	// Buscar detalhes da pesquisa
	details, err := h.surveyUseCase.GetSurveyDetails(surveyID, params)
	if err != nil {
		// Verificar se é erro de "não encontrado"
		if err.Error() == "pesquisa não encontrada: record not found" {
			return c.Status(404).JSON(fiber.Map{"error": "Pesquisa não encontrada"})
		}

		// Registrar o erro no log para investigação
		fmt.Printf("ERRO ao buscar detalhes da pesquisa ID %d: %v\n", surveyID, err)

		// Formatar uma mensagem de erro mais amigável para o usuário
		errorMessage := fmt.Sprintf("Erro ao buscar detalhes da pesquisa: %s", err.Error())
		return c.Status(500).JSON(fiber.Map{
			"error":     errorMessage,
			"survey_id": surveyID,
		})
	}

	// Verificar se recebemos resultados
	detailsArray, ok := details.([]map[string]interface{})
	if !ok || len(detailsArray) == 0 {
		// Retornar uma resposta vazia, mas não um erro
		return c.JSON([]map[string]interface{}{
			{
				"pergunta_id":    "no_data",
				"texto_pergunta": "Nenhuma resposta encontrada para o período selecionado",
				"respostas":      []map[string]interface{}{},
			},
		})
	}

	// Retornar resposta
	return c.JSON(details)
}
