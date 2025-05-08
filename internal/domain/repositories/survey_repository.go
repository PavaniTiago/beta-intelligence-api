package repositories

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/utils"
	"gorm.io/gorm"
)

// SurveyRepository implementa métodos para acesso a dados de pesquisas
type SurveyRepository struct {
	db *gorm.DB
}

// NewSurveyRepository cria uma nova instância de SurveyRepository
func NewSurveyRepository(db *gorm.DB) *SurveyRepository {
	return &SurveyRepository{
		db: db,
	}
}

// GetSurveys retorna todas as pesquisas com opção de filtros
func (r *SurveyRepository) GetSurveys(params map[string]interface{}) ([]entities.Survey, int64, error) {
	var surveys []entities.Survey
	var total int64

	// Obtendo localização para conversão para Brasília
	brazilLocation := utils.GetBrasilLocation()

	// Construindo a consulta base
	query := r.db.Model(&entities.Survey{})

	// Adiciona JOIN com funil se necessário
	if params["include_funnel"] == true {
		query = query.Preload("Funnel")
	}

	// Aplicando filtros
	if funnelID, ok := params["funnel_id"].(int); ok && funnelID > 0 {
		query = query.Where("funnel_id = ?", funnelID)
	}

	if surveyID, ok := params["survey_id"].(int64); ok && surveyID > 0 {
		query = query.Where("survey_id = ?", surveyID)
	}

	// Aplicando paginação
	page, _ := params["page"].(int)
	limit, _ := params["limit"].(int)

	if page <= 0 {
		page = 1
	}

	if limit <= 0 {
		limit = 10
	}

	// Contar total de registros antes da paginação
	query.Count(&total)

	// Aplicar ordenação
	sortBy, _ := params["sort_by"].(string)
	sortDirection, _ := params["sort_direction"].(string)

	if sortBy == "" {
		sortBy = "created_at"
	}

	if sortDirection == "" {
		sortDirection = "desc"
	}

	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortDirection))

	// Aplicar paginação
	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit)

	// Executando a consulta
	if err := query.Find(&surveys).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao buscar pesquisas: %w", err)
	}

	// Converter timestamps para horário de Brasília
	for i := range surveys {
		surveys[i].CreatedAt = surveys[i].CreatedAt.In(brazilLocation)
		surveys[i].UpdatedAt = surveys[i].UpdatedAt.In(brazilLocation)
	}

	return surveys, total, nil
}

// GetSurveyMetrics retorna métricas agregadas de pesquisas com base nos filtros fornecidos
func (r *SurveyRepository) GetSurveyMetrics(params map[string]interface{}) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Definir os filtros de data baseados nos parâmetros
	dataInicio, dataInicioOk := params["data_inicio"].(time.Time)
	dataFim, dataFimOk := params["data_fim"].(time.Time)

	leadInicio, leadInicioOk := params["lead_inicio"].(time.Time)
	leadFim, leadFimOk := params["lead_fim"].(time.Time)

	pesquisaInicio, pesquisaInicioOk := params["pesquisa_inicio"].(time.Time)
	pesquisaFim, pesquisaFimOk := params["pesquisa_fim"].(time.Time)

	vendaInicio, vendaInicioOk := params["venda_inicio"].(time.Time)
	vendaFim, vendaFimOk := params["venda_fim"].(time.Time)

	// Usar filtros gerais se filtros específicos não foram fornecidos
	if !leadInicioOk && dataInicioOk {
		leadInicio = dataInicio
		leadInicioOk = true
	}

	if !leadFimOk && dataFimOk {
		leadFim = dataFim
		leadFimOk = true
	}

	if !pesquisaInicioOk && dataInicioOk {
		pesquisaInicio = dataInicio
		pesquisaInicioOk = true
	}

	if !pesquisaFimOk && dataFimOk {
		pesquisaFim = dataFim
		pesquisaFimOk = true
	}

	if !vendaInicioOk && dataInicioOk {
		vendaInicio = dataInicio
		vendaInicioOk = true
	}

	if !vendaFimOk && dataFimOk {
		vendaFim = dataFim
		vendaFimOk = true
	}

	// Construir condições de filtro de data para cada tipo de evento
	var leadTimeFilter, pesquisaTimeFilter, vendaTimeFilter string
	var leadTimeArgs, pesquisaTimeArgs, vendaTimeArgs []interface{}

	if leadInicioOk && leadFimOk {
		leadTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' BETWEEN ? AND ?"
		leadTimeArgs = []interface{}{leadInicio.Format(time.RFC3339), leadFim.Format(time.RFC3339)}
	} else if leadInicioOk {
		leadTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' >= ?"
		leadTimeArgs = []interface{}{leadInicio.Format(time.RFC3339)}
	} else if leadFimOk {
		leadTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' <= ?"
		leadTimeArgs = []interface{}{leadFim.Format(time.RFC3339)}
	}

	if pesquisaInicioOk && pesquisaFimOk {
		pesquisaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' BETWEEN ? AND ?"
		pesquisaTimeArgs = []interface{}{pesquisaInicio.Format(time.RFC3339), pesquisaFim.Format(time.RFC3339)}
	} else if pesquisaInicioOk {
		pesquisaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' >= ?"
		pesquisaTimeArgs = []interface{}{pesquisaInicio.Format(time.RFC3339)}
	} else if pesquisaFimOk {
		pesquisaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' <= ?"
		pesquisaTimeArgs = []interface{}{pesquisaFim.Format(time.RFC3339)}
	}

	if vendaInicioOk && vendaFimOk {
		vendaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' BETWEEN ? AND ?"
		vendaTimeArgs = []interface{}{vendaInicio.Format(time.RFC3339), vendaFim.Format(time.RFC3339)}
	} else if vendaInicioOk {
		vendaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' >= ?"
		vendaTimeArgs = []interface{}{vendaInicio.Format(time.RFC3339)}
	} else if vendaFimOk {
		vendaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' <= ?"
		vendaTimeArgs = []interface{}{vendaFim.Format(time.RFC3339)}
	}

	// Consulta para métricas agregadas
	sqlQuery := `
WITH pesquisa_users AS (
  SELECT user_id
  FROM events
  WHERE event_type = 'PESQUISA_LEAD'
  ` + func() string {
		if pesquisaTimeFilter != "" {
			return "AND " + pesquisaTimeFilter
		}
		return ""
	}() + `
),

eventos_filtrados AS (
  SELECT e.*
  FROM events e
  WHERE e.event_type IN ('LEAD', 'PESQUISA_LEAD', 'PURCHASE')
  ` + func() string {
		var conditions []string

		if leadTimeFilter != "" {
			conditions = append(conditions, "(e.event_type = 'LEAD' AND "+leadTimeFilter+")")
		}

		if pesquisaTimeFilter != "" {
			conditions = append(conditions, "(e.event_type = 'PESQUISA_LEAD' AND "+pesquisaTimeFilter+")")
		}

		if vendaTimeFilter != "" {
			conditions = append(conditions, "(e.event_type = 'PURCHASE' AND "+vendaTimeFilter+")")
		}

		if len(conditions) > 0 {
			return "AND (" + strings.Join(conditions, " OR ") + ")"
		}
		return ""
	}() + func() string {
		if profissaoID, ok := params["profissao"].(int); ok && profissaoID > 0 {
			return fmt.Sprintf(" AND e.profession_id = %d", profissaoID)
		}
		return ""
	}() + func() string {
		if funilID, ok := params["funil"].(int); ok && funilID > 0 {
			return fmt.Sprintf(" AND e.funnel_id = %d", funilID)
		}
		return ""
	}() + `
),

base AS (
  SELECT 
    s.survey_name,
    f.funnel_name,
    p.profession_name,
    e.user_id,
    e.event_type
  FROM surveys s
  JOIN funnels f ON s.funnel_id = f.funnel_id
  JOIN products pr ON f.product_id = pr.product_id
  JOIN professions p ON pr.profession_id = p.profession_id
  LEFT JOIN eventos_filtrados e ON f.funnel_id = e.funnel_id
  ` + func() string {
		if pesquisaID, ok := params["pesquisa_id"].(int64); ok && pesquisaID > 0 {
			return fmt.Sprintf("WHERE s.survey_id = %d", pesquisaID)
		}
		return ""
	}() + `
)

SELECT 
  s.survey_id AS survey_id,
  b.survey_name AS nome_pesquisa,
  b.funnel_name AS funil,
  b.profession_name AS profissao,

  COUNT(DISTINCT b.user_id) FILTER (WHERE b.event_type = 'LEAD') AS total_leads,
  COUNT(DISTINCT b.user_id) FILTER (WHERE b.event_type = 'PESQUISA_LEAD') AS total_respostas,

  COUNT(DISTINCT b.user_id) FILTER (
    WHERE b.event_type = 'PURCHASE' 
      AND b.user_id IN (SELECT user_id FROM pesquisa_users)
  ) AS total_vendas,

  ROUND(
    (COUNT(DISTINCT b.user_id) FILTER (WHERE b.event_type = 'PESQUISA_LEAD')::numeric /
     NULLIF(COUNT(DISTINCT b.user_id) FILTER (WHERE b.event_type = 'LEAD'), 0)::numeric) * 100, 
    2
  ) / 100 AS taxa_resposta,

  ROUND(
    (COUNT(DISTINCT b.user_id) FILTER (
       WHERE b.event_type = 'PURCHASE' 
         AND b.user_id IN (SELECT user_id FROM pesquisa_users)
     )::numeric /
     NULLIF(COUNT(DISTINCT b.user_id) FILTER (WHERE b.event_type = 'PESQUISA_LEAD'), 0)::numeric) * 100,
    2
  ) / 100 AS conversao_vendas

FROM base b
JOIN surveys s ON s.survey_name = b.survey_name
GROUP BY 
  s.survey_id,
  b.survey_name,
  b.funnel_name,
  b.profession_name;`

	// Preparar os argumentos da consulta
	args := []interface{}{}
	if leadTimeFilter != "" {
		args = append(args, leadTimeArgs...)
	}
	if pesquisaTimeFilter != "" {
		args = append(args, pesquisaTimeArgs...)
	}
	if vendaTimeFilter != "" {
		args = append(args, vendaTimeArgs...)
	}

	// Executar a consulta
	if err := r.db.Raw(sqlQuery, args...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar métricas de pesquisa: %w", err)
	}

	return results, nil
}

// GetSurveyDetails retorna detalhes de uma pesquisa específica, incluindo análise por pergunta e resposta
func (r *SurveyRepository) GetSurveyDetails(surveyID int64, params map[string]interface{}) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// Verificar se a pesquisa existe
	var survey entities.Survey
	if err := r.db.Where("survey_id = ?", surveyID).First(&survey).Error; err != nil {
		return nil, fmt.Errorf("pesquisa não encontrada: %w", err)
	}

	// Definir os filtros de data baseados nos parâmetros
	dataInicio, dataInicioOk := params["data_inicio"].(time.Time)
	dataFim, dataFimOk := params["data_fim"].(time.Time)

	pesquisaInicio, pesquisaInicioOk := params["pesquisa_inicio"].(time.Time)
	pesquisaFim, pesquisaFimOk := params["pesquisa_fim"].(time.Time)

	vendaInicio, vendaInicioOk := params["venda_inicio"].(time.Time)
	vendaFim, vendaFimOk := params["venda_fim"].(time.Time)

	// Usar filtros gerais se filtros específicos não foram fornecidos
	if !pesquisaInicioOk && dataInicioOk {
		pesquisaInicio = dataInicio
		pesquisaInicioOk = true
	}

	if !pesquisaFimOk && dataFimOk {
		pesquisaFim = dataFim
		pesquisaFimOk = true
	}

	if !vendaInicioOk && dataInicioOk {
		vendaInicio = dataInicio
		vendaInicioOk = true
	}

	if !vendaFimOk && dataFimOk {
		vendaFim = dataFim
		vendaFimOk = true
	}

	// Construir condições de filtro de data
	var pesquisaTimeFilter, vendaTimeFilter string
	var pesquisaTimeArgs, vendaTimeArgs []interface{}

	if pesquisaInicioOk && pesquisaFimOk {
		pesquisaTimeFilter = "sr.created_at AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' BETWEEN ? AND ?"
		pesquisaTimeArgs = []interface{}{pesquisaInicio.Format(time.RFC3339), pesquisaFim.Format(time.RFC3339)}
	} else if pesquisaInicioOk {
		pesquisaTimeFilter = "sr.created_at AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' >= ?"
		pesquisaTimeArgs = []interface{}{pesquisaInicio.Format(time.RFC3339)}
	} else if pesquisaFimOk {
		pesquisaTimeFilter = "sr.created_at AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' <= ?"
		pesquisaTimeArgs = []interface{}{pesquisaFim.Format(time.RFC3339)}
	}

	if vendaInicioOk && vendaFimOk {
		vendaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' BETWEEN ? AND ?"
		vendaTimeArgs = []interface{}{vendaInicio.Format(time.RFC3339), vendaFim.Format(time.RFC3339)}
	} else if vendaInicioOk {
		vendaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' >= ?"
		vendaTimeArgs = []interface{}{vendaInicio.Format(time.RFC3339)}
	} else if vendaFimOk {
		vendaTimeFilter = "e.event_time AT TIME ZONE 'UTC' AT TIME ZONE 'America/Sao_Paulo' <= ?"
		vendaTimeArgs = []interface{}{vendaFim.Format(time.RFC3339)}
	}

	// Consulta para detalhes por perguntas e opções de resposta
	sqlQuery := `
WITH responses AS (
  SELECT 
    sr.id AS response_id,
    sr.event_id,
    sr.total_score,
    sr.completed,
    sr.faixa,
    e.user_id,
    s.survey_id,
    s.survey_name,
    f.funnel_id,
    f.funnel_name,
    pr.product_id,
    p.profession_id,
    p.profession_name
  FROM survey_responses sr
  JOIN events e ON sr.event_id = e.event_id
  JOIN surveys s ON sr.survey_id = s.survey_id
  JOIN funnels f ON s.funnel_id = f.funnel_id
  JOIN products pr ON f.product_id = pr.product_id
  JOIN professions p ON pr.profession_id = p.profession_id
  WHERE sr.survey_id = ?
  ` + func() string {
		if pesquisaTimeFilter != "" {
			return "AND " + pesquisaTimeFilter
		}
		return ""
	}() + `
),

vendas AS (
  SELECT 
    e.user_id
  FROM events e
  WHERE e.event_type = 'PURCHASE'
  ` + func() string {
		if vendaTimeFilter != "" {
			return "AND " + vendaTimeFilter
		}
		return ""
	}() + `
  AND e.user_id IN (SELECT DISTINCT user_id FROM responses)
),

answers_grouped AS (
  SELECT
    sa.question_id,
    sa.question_text,
    sa.value,
    sa.score,
    r.profession_name,
    COUNT(DISTINCT sa.survey_response_id) AS num_respostas,
    COUNT(DISTINCT v.user_id) AS num_vendas
  FROM survey_answers sa
  JOIN responses r ON sa.survey_response_id = r.response_id
  LEFT JOIN vendas v ON r.user_id = v.user_id
  GROUP BY
    sa.question_id,
    sa.question_text,
    sa.value,
    sa.score,
    r.profession_name
),

base_stats AS (
  SELECT
    COUNT(DISTINCT responses.response_id) AS total_responses,
    COUNT(DISTINCT vendas.user_id) AS total_vendas,
    responses.profession_name
  FROM responses
  LEFT JOIN vendas ON responses.user_id = vendas.user_id
  GROUP BY responses.profession_name
)

SELECT
  ag.question_id AS pergunta_id,
  ag.question_text AS texto_pergunta,
  ag.value AS texto_opcao,
  ag.score AS score_peso,
  ag.profession_name AS profissao,
  ag.num_respostas,
  ROUND((ag.num_respostas::numeric / bs.total_responses::numeric) * 100, 2) AS percentual_participacao,
  ag.num_vendas,
  CASE WHEN ag.num_respostas > 0 
    THEN ROUND((ag.num_vendas::numeric / ag.num_respostas::numeric) * 100, 2)
    ELSE 0 
  END AS taxa_conversao_percentual,
  CASE WHEN bs.total_vendas > 0 
    THEN ROUND((ag.num_vendas::numeric / bs.total_vendas::numeric) * 100, 2) 
    ELSE 0 
  END AS percentual_vendas
FROM answers_grouped ag
JOIN base_stats bs ON ag.profession_name = bs.profession_name
ORDER BY 
  ag.question_id,
  ag.num_respostas DESC;`

	// Preparar os argumentos da consulta
	args := []interface{}{surveyID}
	if pesquisaTimeFilter != "" {
		args = append(args, pesquisaTimeArgs...)
	}
	if vendaTimeFilter != "" {
		args = append(args, vendaTimeArgs...)
	}

	// Executar a consulta
	if err := r.db.Raw(sqlQuery, args...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar detalhes da pesquisa: %w", err)
	}

	// Reorganizar os resultados por pergunta e suas opções
	questions := make(map[string]map[string]interface{})
	for _, row := range results {
		perguntaID, _ := row["pergunta_id"].(string)

		if _, exists := questions[perguntaID]; !exists {
			questions[perguntaID] = map[string]interface{}{
				"pergunta_id":    perguntaID,
				"texto_pergunta": row["texto_pergunta"],
				"profissao":      row["profissao"],
				"respostas":      []map[string]interface{}{},
			}
		}

		resposta := map[string]interface{}{
			"texto_opcao":               row["texto_opcao"],
			"score_peso":                row["score_peso"],
			"num_respostas":             row["num_respostas"],
			"percentual_participacao":   row["percentual_participacao"],
			"num_vendas":                row["num_vendas"],
			"taxa_conversao_percentual": row["taxa_conversao_percentual"],
			"percentual_vendas":         row["percentual_vendas"],
		}

		respostas, _ := questions[perguntaID]["respostas"].([]map[string]interface{})
		questions[perguntaID]["respostas"] = append(respostas, resposta)
	}

	// Converter o mapa para slice
	formattedResults := []map[string]interface{}{}
	for _, question := range questions {
		formattedResults = append(formattedResults, question)
	}

	return formattedResults, nil
}
