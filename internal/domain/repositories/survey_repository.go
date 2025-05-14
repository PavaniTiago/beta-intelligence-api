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
	pesquisaInicio, pesquisaInicioOk := params["pesquisa_inicio"].(time.Time)
	pesquisaFim, pesquisaFimOk := params["pesquisa_fim"].(time.Time)

	vendaInicio, vendaInicioOk := params["venda_inicio"].(time.Time)
	vendaFim, vendaFimOk := params["venda_fim"].(time.Time)

	// Construir a query SQL
	query := `
WITH pesquisa_users AS (
  SELECT user_id
  FROM events
  WHERE event_type = 'PESQUISA_LEAD'`

	args := []interface{}{}

	// Adicionar filtros de data para pesquisa_users
	if pesquisaInicioOk {
		query += `
  AND event_time >= ?`
		args = append(args, pesquisaInicio.Format(time.RFC3339))
	}

	if pesquisaFimOk {
		query += `
  AND event_time <= ?`
		args = append(args, pesquisaFim.Format(time.RFC3339))
	}

	query += `
),

eventos_filtrados AS (
  SELECT e.*
  FROM events e
  WHERE e.event_type IN ('LEAD', 'PESQUISA_LEAD', 'PURCHASE')
  AND (`

	var conditions []string

	// Filtro para LEAD
	if pesquisaInicioOk && pesquisaFimOk {
		conditions = append(conditions, `(e.event_type = 'LEAD' AND (e.event_time >= ? AND e.event_time <= ?))`)
		args = append(args, pesquisaInicio.Format(time.RFC3339), pesquisaFim.Format(time.RFC3339))
	} else if pesquisaInicioOk {
		conditions = append(conditions, `(e.event_type = 'LEAD' AND e.event_time >= ?)`)
		args = append(args, pesquisaInicio.Format(time.RFC3339))
	} else if pesquisaFimOk {
		conditions = append(conditions, `(e.event_type = 'LEAD' AND e.event_time <= ?)`)
		args = append(args, pesquisaFim.Format(time.RFC3339))
	}

	// Filtro para PESQUISA_LEAD
	if pesquisaInicioOk && pesquisaFimOk {
		conditions = append(conditions, `(e.event_type = 'PESQUISA_LEAD' AND (e.event_time >= ? AND e.event_time <= ?))`)
		args = append(args, pesquisaInicio.Format(time.RFC3339), pesquisaFim.Format(time.RFC3339))
	} else if pesquisaInicioOk {
		conditions = append(conditions, `(e.event_type = 'PESQUISA_LEAD' AND e.event_time >= ?)`)
		args = append(args, pesquisaInicio.Format(time.RFC3339))
	} else if pesquisaFimOk {
		conditions = append(conditions, `(e.event_type = 'PESQUISA_LEAD' AND e.event_time <= ?)`)
		args = append(args, pesquisaFim.Format(time.RFC3339))
	}

	// Filtro para PURCHASE
	if vendaInicioOk && vendaFimOk {
		conditions = append(conditions, `(e.event_type = 'PURCHASE' AND (e.event_time >= ? AND e.event_time <= ?))`)
		args = append(args, vendaInicio.Format(time.RFC3339), vendaFim.Format(time.RFC3339))
	} else if vendaInicioOk {
		conditions = append(conditions, `(e.event_type = 'PURCHASE' AND e.event_time >= ?)`)
		args = append(args, vendaInicio.Format(time.RFC3339))
	} else if vendaFimOk {
		conditions = append(conditions, `(e.event_type = 'PURCHASE' AND e.event_time <= ?)`)
		args = append(args, vendaFim.Format(time.RFC3339))
	}

	// Se não houver condições específicas, adicione uma condição que seja sempre verdadeira
	if len(conditions) == 0 {
		query += "1=1"
	} else {
		query += strings.Join(conditions, " OR ")
	}

	query += `
  )
),

base AS (
  SELECT 
    s.survey_id,
    s.survey_name,
    f.funnel_name,
    p.profession_name,
    e.user_id,
    e.event_type
  FROM surveys s
  JOIN funnels f ON s.funnel_id = f.funnel_id
  JOIN products pr ON f.product_id = pr.product_id
  JOIN professions p ON pr.profession_id = p.profession_id
  LEFT JOIN eventos_filtrados e ON f.funnel_id = e.funnel_id`

	// Adicionar WHERE para pesquisa específica
	if pesquisaID, ok := params["pesquisa_id"].(int64); ok && pesquisaID > 0 {
		query += `
  WHERE s.survey_id = ?`
		args = append(args, pesquisaID)
	}

	query += `
)

SELECT 
  b.survey_id AS pesquisa_id,
  b.survey_name AS survey_name,
  b.funnel_name,
  b.profession_name AS profissao,

  COUNT(*) FILTER (WHERE b.event_type = 'LEAD') AS total_leads,
  COUNT(*) FILTER (WHERE b.event_type = 'PESQUISA_LEAD') AS total_respostas,

  COUNT(*) FILTER (
    WHERE b.event_type = 'PURCHASE' 
      AND b.user_id IN (SELECT user_id FROM pesquisa_users)
  ) AS total_vendas_com_pesquisa,

  ROUND(
    (COUNT(*) FILTER (WHERE b.event_type = 'PESQUISA_LEAD')::numeric /
     NULLIF(COUNT(*) FILTER (WHERE b.event_type = 'LEAD'), 0)::numeric) * 100, 
    2
  ) / 100 AS taxa_resposta_calculada,

  ROUND(
    (COUNT(*) FILTER (
       WHERE b.event_type = 'PURCHASE' 
         AND b.user_id IN (SELECT user_id FROM pesquisa_users)
     )::numeric /
     NULLIF(COUNT(*) FILTER (WHERE b.event_type = 'PESQUISA_LEAD'), 0)::numeric) * 100,
    2
  ) / 100 AS conversao_vendas_calculada

FROM base b
GROUP BY 
  b.survey_id,
  b.survey_name,
  b.funnel_name,
  b.profession_name;`

	// Imprimir a query e os argumentos para debug
	fmt.Printf("FINAL SQL QUERY: %s\n", query)
	fmt.Printf("FINAL SQL ARGS: %v\n", args)

	// Executar a consulta
	if err := r.db.Raw(query, args...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar métricas de pesquisa: %w", err)
	}

	return results, nil
}

// GetSurveyDetails retorna detalhes de uma pesquisa específica com métricas agregadas
func (r *SurveyRepository) GetSurveyDetails(surveyID int64, params map[string]interface{}) ([]map[string]interface{}, error) {
	// Verificar se a pesquisa existe
	var survey entities.Survey
	if err := r.db.Where("survey_id = ?", surveyID).First(&survey).Error; err != nil {
		return nil, fmt.Errorf("pesquisa não encontrada: %w", err)
	}

	// Definir os filtros de data baseados nos parâmetros
	pesquisaInicio, pesquisaInicioOk := params["pesquisa_inicio"].(time.Time)
	pesquisaFim, pesquisaFimOk := params["pesquisa_fim"].(time.Time)
	vendaInicio, vendaInicioOk := params["venda_inicio"].(time.Time)
	vendaFim, vendaFimOk := params["venda_fim"].(time.Time)

	// Log para debug
	fmt.Printf("Survey ID: %d, Params: %+v\n", surveyID, params)
	fmt.Printf("Date Params - pesquisa_inicio: %v, pesquisa_fim: %v, venda_inicio: %v, venda_fim: %v\n",
		pesquisaInicio, pesquisaFim, vendaInicio, vendaFim)

	// Formatar as datas para a consulta SQL
	pesquisaInicioStr := "(TIMESTAMP '1900-01-01 00:00:00-03:00')"
	if pesquisaInicioOk {
		pesquisaInicioStr = fmt.Sprintf("(TIMESTAMP '%s')",
			pesquisaInicio.Format("2006-01-02 15:04:05-07:00"))
	}

	pesquisaFimStr := "(TIMESTAMP '9999-12-31 23:59:59-03:00')"
	if pesquisaFimOk {
		pesquisaFimStr = fmt.Sprintf("(TIMESTAMP '%s')",
			pesquisaFim.Format("2006-01-02 15:04:05-07:00"))
	}

	vendaInicioStr := "(TIMESTAMP '1900-01-01 00:00:00-03:00')"
	if vendaInicioOk {
		vendaInicioStr = fmt.Sprintf("(TIMESTAMP '%s')",
			vendaInicio.Format("2006-01-02 15:04:05-07:00"))
	}

	vendaFimStr := "(TIMESTAMP '9999-12-31 23:59:59-03:00')"
	if vendaFimOk {
		vendaFimStr = fmt.Sprintf("(TIMESTAMP '%s')",
			vendaFim.Format("2006-01-02 15:04:05-07:00"))
	}

	// Consulta principal exatamente como fornecida
	mainQuery := fmt.Sprintf(`
WITH parametros AS (
    SELECT 
        %s AT TIME ZONE 'America/Sao_Paulo' AS pesquisa_inicio,
        %s AT TIME ZONE 'America/Sao_Paulo' AS pesquisa_fim,
        %s AT TIME ZONE 'America/Sao_Paulo' AS venda_inicio,
        %s AT TIME ZONE 'America/Sao_Paulo' AS venda_fim
),

eventos_pesquisa AS (
  SELECT
    funnel_id,
    user_id,
    event_type
  FROM events e
  WHERE event_time BETWEEN 
        (SELECT pesquisa_inicio FROM parametros) AND (SELECT pesquisa_fim FROM parametros)
    AND event_type IN ('LEAD', 'PESQUISA_LEAD')
),

eventos_venda AS (
  SELECT
    funnel_id,
    user_id
  FROM events e
  WHERE event_time BETWEEN 
        (SELECT venda_inicio FROM parametros) AND (SELECT venda_fim FROM parametros)
    AND event_type = 'PURCHASE'
),

eventos_por_funnel AS (
  SELECT
    ep.funnel_id,
    COUNT(*) FILTER (WHERE ep.event_type = 'LEAD') AS total_leads,
    COUNT(*) FILTER (WHERE ep.event_type = 'PESQUISA_LEAD') AS total_respostas,
    COUNT(*) FILTER (
      WHERE ep.event_type = 'PESQUISA_LEAD'
        AND ep.user_id IN (
          SELECT ev.user_id FROM eventos_venda ev WHERE ev.funnel_id = ep.funnel_id
        )
    ) AS total_vendas
  FROM eventos_pesquisa ep
  GROUP BY ep.funnel_id
),

tempo_resposta_por_survey AS (
  SELECT
    sr.survey_id,
    ROUND(SUM(sa.time_to_answer)::numeric / NULLIF(COUNT(sa.time_to_answer), 0), 2) AS tempo_medio_resposta
  FROM survey_answers sa
  JOIN survey_responses sr ON sa.survey_response_id = sr.id
  WHERE sr.created_at BETWEEN 
    (SELECT pesquisa_inicio FROM parametros) AND 
    (SELECT pesquisa_fim FROM parametros)
  GROUP BY sr.survey_id
),

tempo_medio_por_usuario AS (
  SELECT
    respostas_agrupadas.survey_id,
    ROUND(AVG(respostas_agrupadas.total_pesquisa)::numeric, 2) AS tempo_medio_resposta_por_usuario
  FROM (
    SELECT
      sr.survey_id,
      sr.id AS survey_response_id,
      SUM(sa.time_to_answer) AS total_pesquisa
    FROM survey_responses sr
    JOIN survey_answers sa ON sa.survey_response_id = sr.id
    WHERE sr.created_at BETWEEN 
      (SELECT pesquisa_inicio FROM parametros) AND 
      (SELECT pesquisa_fim FROM parametros)
    GROUP BY sr.survey_id, sr.id
  ) respostas_agrupadas
  GROUP BY respostas_agrupadas.survey_id
)

SELECT
  s.survey_id AS pesquisa_id,
  s.survey_name AS nome_pesquisa,
  p.profession_name AS profissao,
  f.funnel_name AS funil,

  -- Totais
  COALESCE(epf.total_leads, 0) AS total_leads,
  COALESCE(epf.total_respostas, 0) AS total_respostas,
  COALESCE(epf.total_vendas, 0) AS total_vendas,

  -- Taxas
  ROUND(100.0 * COALESCE(epf.total_respostas, 0) / NULLIF(epf.total_leads, 0), 2) AS taxa_resposta_percentual,
  ROUND(100.0 * COALESCE(epf.total_vendas, 0) / NULLIF(epf.total_respostas, 0), 2) AS taxa_conversao_percentual,

  -- Tempos
  COALESCE(trps.tempo_medio_resposta, 0) AS tempo_medio_resposta,
  COALESCE(tmpu.tempo_medio_resposta_por_usuario, 0) AS tempo_medio_resposta_por_usuario

FROM surveys s
JOIN funnels f ON s.funnel_id = f.funnel_id
JOIN products pr ON f.product_id = pr.product_id
JOIN professions p ON pr.profession_id = p.profession_id
LEFT JOIN eventos_por_funnel epf ON epf.funnel_id = f.funnel_id
LEFT JOIN tempo_resposta_por_survey trps ON trps.survey_id = s.survey_id
LEFT JOIN tempo_medio_por_usuario tmpu ON tmpu.survey_id = s.survey_id
WHERE s.survey_id = %d
ORDER BY s.survey_name
`,
		pesquisaInicioStr, pesquisaFimStr,
		vendaInicioStr, vendaFimStr,
		surveyID)

	// Imprimir a query para debug
	fmt.Printf("FINAL DETAILS SQL QUERY: %s\n", mainQuery)

	// Executar a consulta principal
	var mainResults []map[string]interface{}
	if err := r.db.Raw(mainQuery).Scan(&mainResults).Error; err != nil {
		return nil, fmt.Errorf("erro ao executar consulta de detalhes da pesquisa: %w", err)
	}

	// Verificar se retornou resultados
	if len(mainResults) == 0 {
		// Retornar uma estrutura vazia com mensagem em vez de erro
		return []map[string]interface{}{
			{
				"pesquisa_id":                      surveyID,
				"nome_pesquisa":                    "Pesquisa não encontrada ou sem dados para o período",
				"profissao":                        "",
				"funil":                            "",
				"total_leads":                      0,
				"total_respostas":                  0,
				"total_vendas":                     0,
				"taxa_resposta_percentual":         0,
				"taxa_conversao_percentual":        0,
				"tempo_medio_resposta":             0,
				"tempo_medio_resposta_por_usuario": 0,
				"questoes":                         []interface{}{},
			},
		}, nil
	}

	// Consulta para análise detalhada por questão/resposta
	detailQuery := fmt.Sprintf(`
WITH parametros AS (
    SELECT 
        %s AT TIME ZONE 'America/Sao_Paulo' AS pesquisa_inicio,
        %s AT TIME ZONE 'America/Sao_Paulo' AS pesquisa_fim,
        %s AT TIME ZONE 'America/Sao_Paulo' AS venda_inicio,
        %s AT TIME ZONE 'America/Sao_Paulo' AS venda_fim
),

-- Identificar usuários que compraram no período especificado
compradores AS (
    SELECT DISTINCT
        v.user_id
    FROM
        events v
    WHERE
        v.funnel_id = (SELECT funnel_id FROM surveys WHERE survey_id = %d)
        AND v.event_type = 'PURCHASE'
        AND v.event_time BETWEEN (SELECT venda_inicio FROM parametros) AND (SELECT venda_fim FROM parametros)
        AND EXISTS (
            SELECT 1
            FROM events pesq
            WHERE pesq.user_id = v.user_id
            AND pesq.event_type = 'PESQUISA_LEAD'
        )
),

-- Mapeamento direto de cada comprador para sua resposta na pesquisa
respostas_compradores AS (
    SELECT
        c.user_id,
        sa.question_id,
        sa.question_text,
        sa.value AS resposta,
        sa.score
    FROM
        compradores c
        CROSS JOIN LATERAL (
            SELECT 
                sa.question_id,
                sa.question_text,
                sa.value,
                sa.score,
                e.event_time
            FROM 
                events e
                JOIN survey_responses sr ON sr.event_id = e.event_id
                JOIN survey_answers sa ON sa.survey_response_id = sr.id
            WHERE 
                e.user_id = c.user_id
                AND e.event_type = 'PESQUISA_LEAD'
                AND sr.survey_id = %d
            ORDER BY 
                e.event_time DESC
            LIMIT 1
        ) sa
),

-- Todas as respostas de pesquisa no período especificado
todas_respostas AS (
    SELECT
        sa.question_id,
        sa.question_text,
        sa.value AS resposta,
        sa.score,
        e.user_id
    FROM
        survey_answers sa
        JOIN survey_responses sr ON sa.survey_response_id = sr.id
        JOIN events e ON sr.event_id = e.event_id
    WHERE
        e.event_type = 'PESQUISA_LEAD'
        AND sr.survey_id = %d
        AND e.event_time BETWEEN (SELECT pesquisa_inicio FROM parametros) AND (SELECT pesquisa_fim FROM parametros)
),

-- Agrupamento de respostas por pergunta e alternativa
respostas_agrupadas AS (
    SELECT
        question_id,
        question_text,
        resposta,
        score,
        COUNT(DISTINCT user_id) AS total_respondentes
    FROM
        todas_respostas
    GROUP BY
        question_id, question_text, resposta, score
),

-- Contagem de vendas por resposta (usando o mapeamento direto)
vendas_por_resposta AS (
    SELECT
        question_id,
        question_text,
        resposta,
        score,
        COUNT(DISTINCT user_id) AS total_compradores
    FROM
        respostas_compradores
    GROUP BY
        question_id, question_text, resposta, score
),

-- Totais por pergunta para cálculo de percentuais
totais_pergunta AS (
    SELECT
        question_id,
        SUM(total_respondentes) AS total_respostas_pergunta
    FROM
        respostas_agrupadas
    GROUP BY
        question_id
),

-- Total geral de vendas para referência
total_vendas AS (
    SELECT COUNT(DISTINCT user_id) AS total FROM compradores
)

-- Resultado final
SELECT
    ra.question_id,
    ra.question_text,
    ra.score AS score_peso,
    ra.resposta,
    ra.total_respondentes AS num_respostas,
    ROUND((ra.total_respondentes * 100.0 / tp.total_respostas_pergunta)::numeric, 2) AS participacao_percentual,
    COALESCE(vpr.total_compradores, 0) AS num_vendas,
    CASE 
        WHEN ra.total_respondentes > 0 THEN 
            ROUND((COALESCE(vpr.total_compradores, 0) * 100.0 / ra.total_respondentes)::numeric, 2)
        ELSE 0 
    END AS taxa_conversao_percentual,
    -- Percentual em relação ao total de vendas
    CASE 
        WHEN (SELECT total FROM total_vendas) > 0 THEN
            ROUND((COALESCE(vpr.total_compradores, 0) * 100.0 / (SELECT total FROM total_vendas))::numeric, 2) 
        ELSE 0
    END AS percentual_do_total_vendas
FROM
    respostas_agrupadas ra
    JOIN totais_pergunta tp ON ra.question_id = tp.question_id
    LEFT JOIN vendas_por_resposta vpr ON 
        ra.question_id = vpr.question_id AND 
        ra.resposta = vpr.resposta
    CROSS JOIN total_vendas
ORDER BY
    ra.question_id, 
    num_vendas DESC, 
    participacao_percentual DESC;
`,
		pesquisaInicioStr, pesquisaFimStr,
		vendaInicioStr, vendaFimStr,
		survey.FunnelID, surveyID, surveyID)

	fmt.Printf("DETAIL ANALYTICS QUERY: %s\n", detailQuery)

	// Executar a consulta de detalhes por questão/resposta
	var detailResults []map[string]interface{}
	if err := r.db.Raw(detailQuery).Scan(&detailResults).Error; err != nil {
		return nil, fmt.Errorf("erro ao executar consulta de análise por questão/resposta: %w", err)
	}

	// Processar os resultados para organizar as respostas por questão
	questionMap := make(map[string]map[string]interface{})

	for _, row := range detailResults {
		questionID, _ := row["question_id"].(string)
		questionText, _ := row["question_text"].(string)

		// Criar a entrada para a questão se não existir
		if _, exists := questionMap[questionID]; !exists {
			questionMap[questionID] = map[string]interface{}{
				"question_id":   questionID,
				"question_text": questionText,
				"respostas":     []map[string]interface{}{},
			}
		}

		// Adicionar a resposta atual à lista de respostas da questão
		respostas := questionMap[questionID]["respostas"].([]map[string]interface{})

		resposta := map[string]interface{}{
			"resposta":                   row["resposta"],
			"score_peso":                 row["score_peso"],
			"num_respostas":              row["num_respostas"],
			"participacao_percentual":    row["participacao_percentual"],
			"num_vendas":                 row["num_vendas"],
			"taxa_conversao_percentual":  row["taxa_conversao_percentual"],
			"percentual_do_total_vendas": row["percentual_do_total_vendas"],
		}

		questionMap[questionID]["respostas"] = append(respostas, resposta)
	}

	// Converter o mapa de questões para um slice
	questoes := make([]map[string]interface{}, 0, len(questionMap))
	for _, q := range questionMap {
		questoes = append(questoes, q)
	}

	// Adicionar as questões ao resultado principal
	mainResults[0]["questoes"] = questoes

	return mainResults, nil
}
