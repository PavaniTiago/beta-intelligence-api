package repositories

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/utils"
	"gorm.io/gorm"
)

// UnifiedData representa os dados unificados de leads e faturamento
type UnifiedData struct {
	ProfessionID   int       `json:"profession_id"`
	ProfessionName string    `json:"profession_name"`
	ProductID      int       `json:"product_id"`
	ProductName    string    `json:"product_name"`
	FunnelID       int       `json:"funnel_id"`
	FunnelName     string    `json:"funnel_name"`
	LeadCount      int64     `json:"lead_count"`
	PurchaseCount  int64     `json:"purchase_count"`
	TotalRevenue   float64   `json:"total_revenue"`
	EventTime      time.Time `json:"event_time"`
}

// RevenueMetricResult representa um resultado numérico com comparação ao período anterior
type RevenueMetricResult struct {
	Current      int64   `json:"current"`
	Previous     int64   `json:"previous"`
	Percentage   float64 `json:"percentage"`
	IsIncreasing bool    `json:"is_increasing"`
}

// RevenueMetricResultFloat representa um resultado float com comparação ao período anterior
type RevenueMetricResultFloat struct {
	Current      float64 `json:"current"`
	Previous     float64 `json:"previous"`
	Percentage   float64 `json:"percentage"`
	IsIncreasing bool    `json:"is_increasing"`
}

// HourlyRevenueMetrics representa dados por hora para revenue
type HourlyRevenueMetrics struct {
	LeadsByHour     map[string]int64   `json:"leads_by_hour"`
	PurchasesByHour map[string]int64   `json:"purchases_by_hour"`
	RevenueByHour   map[string]float64 `json:"revenue_by_hour"`
}

// RevenueComparisonData representa dados de comparação de revenue
type RevenueComparisonData struct {
	// Informações da profissão (para dados por profissão)
	ProfessionID   int    `json:"profession_id,omitempty"`
	ProfessionName string `json:"profession_name,omitempty"`

	// Métricas principais
	Leads     RevenueMetricResult      `json:"leads"`
	Purchases RevenueMetricResult      `json:"purchases"`
	Revenue   RevenueMetricResultFloat `json:"revenue"`

	// Dados por dia
	LeadsByDay     map[string]int64   `json:"leads_by_day"`
	PurchasesByDay map[string]int64   `json:"purchases_by_day"`
	RevenueByDay   map[string]float64 `json:"revenue_by_day"`

	// Dados do período anterior
	PreviousPeriodData *PreviousRevenueData `json:"previous_period_data,omitempty"`

	// Dados por hora (opcional, apenas para dia único)
	HourlyData *HourlyRevenueMetrics `json:"hourly_data,omitempty"`

	// NOVO: Resumo por profissão (apenas para dados gerais)
	ProfessionSummary []ProfessionSummary `json:"profession_summary,omitempty"`
}

// PreviousRevenueData representa dados detalhados do período anterior
type PreviousRevenueData struct {
	LeadsByDay     map[string]int64      `json:"leads_by_day"`
	PurchasesByDay map[string]int64      `json:"purchases_by_day"`
	RevenueByDay   map[string]float64    `json:"revenue_by_day"`
	HourlyData     *HourlyRevenueMetrics `json:"hourly_data,omitempty"`
}

// ProfessionResult representa dados de uma profissão para comparação
type ProfessionResult struct {
	ProfessionID   int     `gorm:"column:profession_id"`
	ProfessionName string  `gorm:"column:profession_name"`
	TotalLeads     int64   `gorm:"column:total_leads"`
	TotalPurchases int64   `gorm:"column:total_purchases"`
	TotalRevenue   float64 `gorm:"column:total_revenue"`
}

// ProfessionSummary representa um resumo dos dados de uma profissão
type ProfessionSummary struct {
	ProfessionID   int                      `json:"profession_id"`
	ProfessionName string                   `json:"profession_name"`
	Leads          RevenueMetricResult      `json:"leads"`
	Purchases      RevenueMetricResult      `json:"purchases"`
	Revenue        RevenueMetricResultFloat `json:"revenue"`
}

// RevenueRepository interface para operações de faturamento
type RevenueRepository interface {
	GetUnifiedDataByProfession(from, to time.Time, professionIDs []int) ([]UnifiedData, error)
	GetUnifiedDataGeneral(from, to time.Time) (UnifiedData, error)

	// Novos métodos para dados comparativos
	GetRevenueComparisonGeneral(currentFrom, currentTo, previousFrom, previousTo time.Time) (RevenueComparisonData, error)
	GetRevenueComparisonByProfession(currentFrom, currentTo, previousFrom, previousTo time.Time, professionIDs []int) ([]RevenueComparisonData, error)

	// Método para dados por hora
	GetHourlyRevenueData(date time.Time, professionIDs []int) (*HourlyRevenueMetrics, error)
}

type revenueRepository struct {
	db *gorm.DB
}

func NewRevenueRepository(db *gorm.DB) RevenueRepository {
	return &revenueRepository{db}
}

func (r *revenueRepository) GetUnifiedDataByProfession(from, to time.Time, professionIDs []int) ([]UnifiedData, error) {
	var results []UnifiedData

	// Obter localização de Brasília
	brazilLocation := utils.GetBrasilLocation()

	// Converter timestamps para horário de Brasília se necessário
	if !from.IsZero() {
		from = from.In(brazilLocation)
	}
	if !to.IsZero() {
		to = to.In(brazilLocation)
	}

	// Construir filtros como strings para evitar duplicação de argumentos
	var dateFilter string
	var professionFilter string
	args := []interface{}{}

	// Filtro de data
	if !from.IsZero() && !to.IsZero() {
		dateFilter = fmt.Sprintf(" AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'",
			from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"))
	} else if !from.IsZero() {
		dateFilter = fmt.Sprintf(" AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') >= '%s'",
			from.Format("2006-01-02 15:04:05"))
	} else if !to.IsZero() {
		dateFilter = fmt.Sprintf(" AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') <= '%s'",
			to.Format("2006-01-02 15:04:05"))
	}

	// Filtro de profissão
	if len(professionIDs) > 0 {
		professionIDsStr := make([]string, len(professionIDs))
		for i, profID := range professionIDs {
			professionIDsStr[i] = fmt.Sprintf("%d", profID)
		}
		professionFilter = fmt.Sprintf(" AND e.profession_id IN (%s)", strings.Join(professionIDsStr, ","))
	}

	// Query SQL unificada para obter dados de leads e faturamento agrupados por profissão
	query := fmt.Sprintf(`
	WITH lead_counts AS (
		SELECT 
			e.profession_id,
			e.product_id,
			e.funnel_id,
			COUNT(*) as lead_count,
			MAX(e.event_time AT TIME ZONE 'America/Sao_Paulo') as last_lead_time
		FROM events e
		WHERE e.event_type = 'LEAD'%s%s
		GROUP BY e.profession_id, e.product_id, e.funnel_id
	),
	purchase_data AS (
		SELECT 
			e.profession_id,
			e.product_id,
			e.funnel_id,
			COUNT(*) as purchase_count,
			SUM(CAST(e.event_propeties->>'value' AS DECIMAL(10,2))) as total_revenue,
			MAX(e.event_time AT TIME ZONE 'America/Sao_Paulo') as last_purchase_time
		FROM events e
		WHERE e.event_type = 'PURCHASE'
		AND e.event_propeties->>'value' IS NOT NULL
		AND e.event_propeties->>'value' != ''
		AND e.event_propeties->>'value' ~ '^[0-9]+\.?[0-9]*$'%s%s
		GROUP BY e.profession_id, e.product_id, e.funnel_id
	),
	all_combinations AS (
		SELECT profession_id, product_id, funnel_id FROM lead_counts
		UNION
		SELECT profession_id, product_id, funnel_id FROM purchase_data
	)
	SELECT 
		ac.profession_id,
		prof.profession_name,
		ac.product_id,
		prod.product_name,
		ac.funnel_id,
		f.funnel_name,
		COALESCE(lc.lead_count, 0) as lead_count,
		COALESCE(pd.purchase_count, 0) as purchase_count,
		COALESCE(pd.total_revenue, 0) as total_revenue,
		GREATEST(
			COALESCE(lc.last_lead_time, '1900-01-01'::timestamp),
			COALESCE(pd.last_purchase_time, '1900-01-01'::timestamp)
		) as event_time
	FROM all_combinations ac
	LEFT JOIN professions prof ON ac.profession_id = prof.profession_id
	LEFT JOIN products prod ON ac.product_id = prod.product_id
	LEFT JOIN funnels f ON ac.funnel_id = f.funnel_id
	LEFT JOIN lead_counts lc ON (ac.profession_id = lc.profession_id 
		AND ac.product_id = lc.product_id 
		AND ac.funnel_id = lc.funnel_id)
	LEFT JOIN purchase_data pd ON (ac.profession_id = pd.profession_id 
		AND ac.product_id = pd.product_id 
		AND ac.funnel_id = pd.funnel_id)
	ORDER BY prof.profession_name, total_revenue DESC, lead_count DESC
	`, dateFilter, professionFilter, dateFilter, professionFilter)

	fmt.Printf("Unified data by profession query: %s\n", query)
	fmt.Printf("Unified data by profession args: %v\n", args)

	if err := r.db.Raw(query, args...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar dados unificados por profissão: %w", err)
	}

	return results, nil
}

func (r *revenueRepository) GetUnifiedDataGeneral(from, to time.Time) (UnifiedData, error) {
	var result UnifiedData

	// Obter localização de Brasília
	brazilLocation := utils.GetBrasilLocation()

	// Converter timestamps para horário de Brasília se necessário
	if !from.IsZero() {
		from = from.In(brazilLocation)
	}
	if !to.IsZero() {
		to = to.In(brazilLocation)
	}

	// Construir filtros de data como strings para evitar duplicação de argumentos
	var dateFilter string
	args := []interface{}{}

	if !from.IsZero() && !to.IsZero() {
		dateFilter = fmt.Sprintf(" AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'",
			from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"))
	} else if !from.IsZero() {
		dateFilter = fmt.Sprintf(" AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') >= '%s'",
			from.Format("2006-01-02 15:04:05"))
	} else if !to.IsZero() {
		dateFilter = fmt.Sprintf(" AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') <= '%s'",
			to.Format("2006-01-02 15:04:05"))
	}

	// Query SQL unificada para obter dados gerais
	query := fmt.Sprintf(`
	WITH lead_data AS (
		SELECT 
			COUNT(*) as lead_count,
			MAX(e.event_time AT TIME ZONE 'America/Sao_Paulo') as last_lead_time
		FROM events e
		WHERE e.event_type = 'LEAD'%s
	),
	purchase_data AS (
		SELECT 
			COUNT(*) as purchase_count,
			SUM(CAST(e.event_propeties->>'value' AS DECIMAL(10,2))) as total_revenue,
			MAX(e.event_time AT TIME ZONE 'America/Sao_Paulo') as last_purchase_time
		FROM events e
		WHERE e.event_type = 'PURCHASE'
		AND e.event_propeties->>'value' IS NOT NULL
		AND e.event_propeties->>'value' != ''
		AND e.event_propeties->>'value' ~ '^[0-9]+\.?[0-9]*$'%s
	)
	SELECT 
		0 as profession_id,
		'Geral' as profession_name,
		0 as product_id,
		'Todos os Produtos' as product_name,
		0 as funnel_id,
		'Todos os Funis' as funnel_name,
		ld.lead_count,
		COALESCE(pd.purchase_count, 0) as purchase_count,
		COALESCE(pd.total_revenue, 0) as total_revenue,
		GREATEST(
			COALESCE(ld.last_lead_time, '1900-01-01'::timestamp),
			COALESCE(pd.last_purchase_time, '1900-01-01'::timestamp)
		) as event_time
	FROM lead_data ld
	CROSS JOIN purchase_data pd
	`, dateFilter, dateFilter)

	fmt.Printf("General unified data query: %s\n", query)
	fmt.Printf("General unified data args: %v\n", args)

	if err := r.db.Raw(query, args...).Scan(&result).Error; err != nil {
		return result, fmt.Errorf("erro ao buscar dados unificados gerais: %w", err)
	}

	return result, nil
}

func (r *revenueRepository) GetRevenueComparisonGeneral(currentFrom, currentTo, previousFrom, previousTo time.Time) (RevenueComparisonData, error) {
	result := RevenueComparisonData{
		LeadsByDay:     make(map[string]int64),
		PurchasesByDay: make(map[string]int64),
		RevenueByDay:   make(map[string]float64),
		PreviousPeriodData: &PreviousRevenueData{
			LeadsByDay:     make(map[string]int64),
			PurchasesByDay: make(map[string]int64),
			RevenueByDay:   make(map[string]float64),
		},
	}

	// Obter localização de Brasília
	brazilLocation := utils.GetBrasilLocation()

	// Converter timestamps para horário de Brasília
	currentFrom = currentFrom.In(brazilLocation)
	currentTo = currentTo.In(brazilLocation)
	previousFrom = previousFrom.In(brazilLocation)
	previousTo = previousTo.In(brazilLocation)

	// Query SUPER OTIMIZADA: tudo em uma única consulta
	query := fmt.Sprintf(`
	WITH 
	-- Dados do período atual
	current_data AS (
		SELECT 
			event_type,
			to_char(date_trunc('day', event_time AT TIME ZONE 'America/Sao_Paulo'), 'YYYY-MM-DD') as dia,
			profession_id,
			COUNT(*) as count,
			CASE 
				WHEN event_type = 'PURCHASE' THEN SUM(CAST(COALESCE(event_propeties->>'value', '0') AS DECIMAL(10,2)))
				ELSE 0 
			END as revenue
		FROM events 
		WHERE event_type IN ('LEAD', 'PURCHASE')
		AND (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'
		AND (event_type = 'LEAD' OR (
			event_type = 'PURCHASE' 
			AND event_propeties->>'value' IS NOT NULL 
			AND event_propeties->>'value' != '' 
			AND event_propeties->>'value' ~ '^[0-9]+\.?[0-9]*$'
		))
		GROUP BY event_type, dia, profession_id
	),
	-- Dados do período anterior
	previous_data AS (
		SELECT 
			event_type,
			to_char(date_trunc('day', event_time AT TIME ZONE 'America/Sao_Paulo'), 'YYYY-MM-DD') as dia,
			profession_id,
			COUNT(*) as count,
			CASE 
				WHEN event_type = 'PURCHASE' THEN SUM(CAST(COALESCE(event_propeties->>'value', '0') AS DECIMAL(10,2)))
				ELSE 0 
			END as revenue
		FROM events 
		WHERE event_type IN ('LEAD', 'PURCHASE')
		AND (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'
		AND (event_type = 'LEAD' OR (
			event_type = 'PURCHASE' 
			AND event_propeties->>'value' IS NOT NULL 
			AND event_propeties->>'value' != '' 
			AND event_propeties->>'value' ~ '^[0-9]+\.?[0-9]*$'
		))
		GROUP BY event_type, dia, profession_id
	),
	-- Totais consolidados
	totals AS (
		SELECT 
			'current' as period,
			'LEAD' as event_type,
			SUM(count) as total_count,
			0 as total_revenue
		FROM current_data WHERE event_type = 'LEAD'
		UNION ALL
		SELECT 
			'current' as period,
			'PURCHASE' as event_type,
			SUM(count) as total_count,
			SUM(revenue) as total_revenue
		FROM current_data WHERE event_type = 'PURCHASE'
		UNION ALL
		SELECT 
			'previous' as period,
			'LEAD' as event_type,
			SUM(count) as total_count,
			0 as total_revenue
		FROM previous_data WHERE event_type = 'LEAD'
		UNION ALL
		SELECT 
			'previous' as period,
			'PURCHASE' as event_type,
			SUM(count) as total_count,
			SUM(revenue) as total_revenue
		FROM previous_data WHERE event_type = 'PURCHASE'
	),
	-- Resumo por profissão
	profession_summary AS (
		SELECT 
			COALESCE(c.profession_id, p.profession_id) as profession_id,
			prof.profession_name,
			SUM(CASE WHEN c.event_type = 'LEAD' THEN c.count ELSE 0 END) as current_leads,
			SUM(CASE WHEN c.event_type = 'PURCHASE' THEN c.count ELSE 0 END) as current_purchases,
			SUM(CASE WHEN c.event_type = 'PURCHASE' THEN c.revenue ELSE 0 END) as current_revenue,
			SUM(CASE WHEN p.event_type = 'LEAD' THEN p.count ELSE 0 END) as previous_leads,
			SUM(CASE WHEN p.event_type = 'PURCHASE' THEN p.count ELSE 0 END) as previous_purchases,
			SUM(CASE WHEN p.event_type = 'PURCHASE' THEN p.revenue ELSE 0 END) as previous_revenue
		FROM current_data c
		FULL OUTER JOIN previous_data p ON c.profession_id = p.profession_id AND c.event_type = p.event_type
		LEFT JOIN professions prof ON COALESCE(c.profession_id, p.profession_id) = prof.profession_id
		GROUP BY COALESCE(c.profession_id, p.profession_id), prof.profession_name
	)
	-- Resultado final unificado
	SELECT 'daily_current' as type, event_type::text, dia as key, count::BIGINT as value, revenue::DECIMAL(10,2) FROM current_data
	UNION ALL
	SELECT 'daily_previous' as type, event_type::text, dia as key, count::BIGINT as value, revenue::DECIMAL(10,2) FROM previous_data
	UNION ALL
	SELECT 'totals' as type, event_type::text, period as key, total_count::BIGINT as value, total_revenue::DECIMAL(10,2) as revenue FROM totals
	UNION ALL
	SELECT 'profession' as type, 'SUMMARY'::text as event_type, 
		profession_id::text as key, 
		0::BIGINT as value,
		0::DECIMAL(10,2) as revenue
	FROM profession_summary
	ORDER BY type, event_type, key
	`,
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"))

	type UnifiedResult struct {
		Type      string  `gorm:"column:type"`
		EventType string  `gorm:"column:event_type"`
		Key       string  `gorm:"column:key"`
		Value     int64   `gorm:"column:value"`
		Revenue   float64 `gorm:"column:revenue"`
	}

	var results []UnifiedResult
	if err := r.db.Raw(query).Scan(&results).Error; err != nil {
		return result, fmt.Errorf("erro na consulta unificada de revenue: %w", err)
	}

	var currentLeads, previousLeads, currentPurchases, previousPurchases int64
	var currentRevenue, previousRevenue float64

	// Processar resultados de forma otimizada
	for _, row := range results {
		switch row.Type {
		case "daily_current":
			if row.EventType == "LEAD" {
				result.LeadsByDay[row.Key] = row.Value
			} else if row.EventType == "PURCHASE" {
				result.PurchasesByDay[row.Key] = row.Value
				result.RevenueByDay[row.Key] = math.Round(row.Revenue*100) / 100
			}
		case "daily_previous":
			if row.EventType == "LEAD" {
				result.PreviousPeriodData.LeadsByDay[row.Key] = row.Value
			} else if row.EventType == "PURCHASE" {
				result.PreviousPeriodData.PurchasesByDay[row.Key] = row.Value
				result.PreviousPeriodData.RevenueByDay[row.Key] = math.Round(row.Revenue*100) / 100
			}
		case "totals":
			if row.Key == "current" && row.EventType == "LEAD" {
				currentLeads = row.Value
			} else if row.Key == "current" && row.EventType == "PURCHASE" {
				currentPurchases = row.Value
				currentRevenue = row.Revenue
			} else if row.Key == "previous" && row.EventType == "LEAD" {
				previousLeads = row.Value
			} else if row.Key == "previous" && row.EventType == "PURCHASE" {
				previousPurchases = row.Value
				previousRevenue = row.Revenue
			}
		}
	}

	// Calcular métricas de comparação
	result.Leads = r.calculateMetricComparison(currentLeads, previousLeads)
	result.Purchases = r.calculateMetricComparison(currentPurchases, previousPurchases)
	result.Revenue = r.calculateFloatMetricComparison(currentRevenue, previousRevenue)

	// Buscar resumo por profissão de forma otimizada
	professionSummary, err := r.getProfessionSummaryOptimized(currentFrom, currentTo, previousFrom, previousTo)
	if err == nil {
		result.ProfessionSummary = professionSummary
	}

	return result, nil
}

func (r *revenueRepository) GetRevenueComparisonByProfession(currentFrom, currentTo, previousFrom, previousTo time.Time, professionIDs []int) ([]RevenueComparisonData, error) {
	// Obter localização de Brasília
	brazilLocation := utils.GetBrasilLocation()

	// Converter timestamps para horário de Brasília
	currentFrom = currentFrom.In(brazilLocation)
	currentTo = currentTo.In(brazilLocation)
	previousFrom = previousFrom.In(brazilLocation)
	previousTo = previousTo.In(brazilLocation)

	// Construir filtro de profissão
	var professionFilter string
	if len(professionIDs) > 0 {
		professionIDsStr := make([]string, len(professionIDs))
		for i, profID := range professionIDs {
			professionIDsStr[i] = fmt.Sprintf("%d", profID)
		}
		professionFilter = fmt.Sprintf(" AND profession_id IN (%s)", strings.Join(professionIDsStr, ","))
	}

	// Query SUPER OTIMIZADA: tudo em uma única consulta por profissão
	query := fmt.Sprintf(`
	WITH profession_data AS (
		SELECT 
			profession_id,
			event_type,
			to_char(date_trunc('day', event_time AT TIME ZONE 'America/Sao_Paulo'), 'YYYY-MM-DD') as dia,
			CASE 
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'current'
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'previous'
				ELSE NULL
			END as period,
			COUNT(*) as count,
			CASE 
				WHEN event_type = 'PURCHASE' THEN SUM(CAST(COALESCE(event_propeties->>'value', '0') AS DECIMAL(10,2)))
				ELSE 0 
			END as revenue
		FROM events 
		WHERE event_type IN ('LEAD', 'PURCHASE')
		AND (
			(event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'
			OR (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'
		)%s
		AND (event_type = 'LEAD' OR (
			event_type = 'PURCHASE' 
			AND event_propeties->>'value' IS NOT NULL 
			AND event_propeties->>'value' != '' 
			AND event_propeties->>'value' ~ '^[0-9]+\.?[0-9]*$'
		))
		GROUP BY profession_id, event_type, dia, 
			CASE 
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'current'
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'previous'
				ELSE NULL
			END
		HAVING CASE 
			WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'current'
			WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'previous'
			ELSE NULL
		END IS NOT NULL
	)
	SELECT 
		pd.profession_id,
		COALESCE(prof.profession_name, 'Profissão ' || pd.profession_id) as profession_name,
		pd.period,
		pd.event_type,
		pd.dia,
		pd.count,
		pd.revenue,
		-- Totais por profissão e período
		SUM(pd.count) OVER (PARTITION BY pd.profession_id, pd.period, pd.event_type) as total_count,
		SUM(pd.revenue) OVER (PARTITION BY pd.profession_id, pd.period, pd.event_type) as total_revenue
	FROM profession_data pd
	LEFT JOIN professions prof ON pd.profession_id = prof.profession_id
	ORDER BY pd.profession_id, pd.period, pd.event_type, pd.dia
	`,
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"),
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"),
		professionFilter,
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"),
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"))

	type ProfessionDataResult struct {
		ProfessionID   int     `gorm:"column:profession_id"`
		ProfessionName string  `gorm:"column:profession_name"`
		Period         string  `gorm:"column:period"`
		EventType      string  `gorm:"column:event_type"`
		Dia            string  `gorm:"column:dia"`
		Count          int64   `gorm:"column:count"`
		Revenue        float64 `gorm:"column:revenue"`
		TotalCount     int64   `gorm:"column:total_count"`
		TotalRevenue   float64 `gorm:"column:total_revenue"`
	}

	var queryResults []ProfessionDataResult
	if err := r.db.Raw(query).Scan(&queryResults).Error; err != nil {
		return nil, fmt.Errorf("erro na consulta otimizada por profissão: %w", err)
	}

	// Debug: log da query e filtros aplicados
	fmt.Printf("GetRevenueComparisonByProfession - Filtro aplicado: %s\n", professionFilter)
	fmt.Printf("GetRevenueComparisonByProfession - Resultados encontrados: %d\n", len(queryResults))
	if len(queryResults) > 0 {
		fmt.Printf("GetRevenueComparisonByProfession - Primeira profissão: ID=%d, Nome=%s\n",
			queryResults[0].ProfessionID, queryResults[0].ProfessionName)
	}

	// Agrupar resultados por profissão
	professionMap := make(map[int]*RevenueComparisonData)
	professionTotals := make(map[int]map[string]map[string]int64)
	professionRevenueTotals := make(map[int]map[string]float64)
	professionNames := make(map[int]string)

	for _, row := range queryResults {
		// Armazenar nome da profissão
		professionNames[row.ProfessionID] = row.ProfessionName

		// Inicializar estruturas se necessário
		if _, exists := professionMap[row.ProfessionID]; !exists {
			professionMap[row.ProfessionID] = &RevenueComparisonData{
				ProfessionID:   row.ProfessionID,
				ProfessionName: row.ProfessionName,
				LeadsByDay:     make(map[string]int64),
				PurchasesByDay: make(map[string]int64),
				RevenueByDay:   make(map[string]float64),
				PreviousPeriodData: &PreviousRevenueData{
					LeadsByDay:     make(map[string]int64),
					PurchasesByDay: make(map[string]int64),
					RevenueByDay:   make(map[string]float64),
				},
			}
			professionTotals[row.ProfessionID] = make(map[string]map[string]int64)
			professionTotals[row.ProfessionID]["current"] = make(map[string]int64)
			professionTotals[row.ProfessionID]["previous"] = make(map[string]int64)
			professionRevenueTotals[row.ProfessionID] = make(map[string]float64)
		}

		data := professionMap[row.ProfessionID]

		// Preencher dados diários
		if row.Period == "current" {
			if row.EventType == "LEAD" {
				data.LeadsByDay[row.Dia] = row.Count
			} else if row.EventType == "PURCHASE" {
				data.PurchasesByDay[row.Dia] = row.Count
				data.RevenueByDay[row.Dia] = math.Round(row.Revenue*100) / 100
			}
		} else if row.Period == "previous" {
			if row.EventType == "LEAD" {
				data.PreviousPeriodData.LeadsByDay[row.Dia] = row.Count
			} else if row.EventType == "PURCHASE" {
				data.PreviousPeriodData.PurchasesByDay[row.Dia] = row.Count
				data.PreviousPeriodData.RevenueByDay[row.Dia] = math.Round(row.Revenue*100) / 100
			}
		}

		// Armazenar totais únicos (evitar duplicação devido ao window function)
		key := fmt.Sprintf("%s_%s", row.Period, row.EventType)
		if _, exists := professionTotals[row.ProfessionID][row.Period][row.EventType]; !exists {
			professionTotals[row.ProfessionID][row.Period][row.EventType] = row.TotalCount
			if row.EventType == "PURCHASE" {
				professionRevenueTotals[row.ProfessionID][key] = row.TotalRevenue
			}
		}
	}

	// Converter para slice e calcular comparações
	type ProfessionWithRevenue struct {
		ProfessionID int
		Data         *RevenueComparisonData
		Revenue      float64
	}

	professionList := make([]ProfessionWithRevenue, 0, len(professionMap))

	for professionID, data := range professionMap {
		// Calcular métricas de comparação
		currentLeads := professionTotals[professionID]["current"]["LEAD"]
		previousLeads := professionTotals[professionID]["previous"]["LEAD"]
		currentPurchases := professionTotals[professionID]["current"]["PURCHASE"]
		previousPurchases := professionTotals[professionID]["previous"]["PURCHASE"]
		currentRevenue := professionRevenueTotals[professionID]["current_PURCHASE"]
		previousRevenue := professionRevenueTotals[professionID]["previous_PURCHASE"]

		data.Leads = r.calculateMetricComparison(currentLeads, previousLeads)
		data.Purchases = r.calculateMetricComparison(currentPurchases, previousPurchases)
		data.Revenue = r.calculateFloatMetricComparison(currentRevenue, previousRevenue)

		professionList = append(professionList, ProfessionWithRevenue{
			ProfessionID: professionID,
			Data:         data,
			Revenue:      currentRevenue,
		})
	}

	// Ordenar por revenue (maior para menor)
	for i := 0; i < len(professionList)-1; i++ {
		for j := i + 1; j < len(professionList); j++ {
			if professionList[i].Revenue < professionList[j].Revenue {
				professionList[i], professionList[j] = professionList[j], professionList[i]
			}
		}
	}

	// Converter para slice final
	results := make([]RevenueComparisonData, 0, len(professionList))
	for _, prof := range professionList {
		results = append(results, *prof.Data)
	}

	return results, nil
}

func (r *revenueRepository) GetHourlyRevenueData(date time.Time, professionIDs []int) (*HourlyRevenueMetrics, error) {
	result := &HourlyRevenueMetrics{
		LeadsByHour:     make(map[string]int64),
		PurchasesByHour: make(map[string]int64),
		RevenueByHour:   make(map[string]float64),
	}

	// Obter localização de Brasília
	brazilLocation := utils.GetBrasilLocation()
	date = date.In(brazilLocation)

	// Definir início e fim do dia
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, brazilLocation)
	endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, brazilLocation)

	// Verificar se é o dia atual
	now := time.Now().In(brazilLocation)
	isToday := date.Year() == now.Year() && date.Month() == now.Month() && date.Day() == now.Day()

	// Calcular horas máximas
	var maxHour int
	if isToday {
		maxHour = now.Hour() + 1
		if maxHour > 24 {
			maxHour = 24
		}
	} else {
		maxHour = 24
	}

	// Inicializar mapas com zeros de forma otimizada
	for hour := 0; hour < maxHour; hour++ {
		hourStr := fmt.Sprintf("%02d", hour)
		result.LeadsByHour[hourStr] = 0
		result.PurchasesByHour[hourStr] = 0
		result.RevenueByHour[hourStr] = 0
	}

	// Construir filtro de profissão
	var professionFilter string
	if len(professionIDs) > 0 {
		professionIDsStr := make([]string, len(professionIDs))
		for i, profID := range professionIDs {
			professionIDsStr[i] = fmt.Sprintf("%d", profID)
		}
		professionFilter = fmt.Sprintf(" AND profession_id IN (%s)", strings.Join(professionIDsStr, ","))
	}

	// Query SUPER OTIMIZADA: uma única consulta para leads e purchases por hora
	query := fmt.Sprintf(`
	SELECT 
		event_type,
		to_char(date_trunc('hour', event_time AT TIME ZONE 'America/Sao_Paulo'), 'HH24') AS hour_str,
		COUNT(*) AS count,
		CASE 
			WHEN event_type = 'PURCHASE' THEN SUM(CAST(COALESCE(event_propeties->>'value', '0') AS DECIMAL(10,2)))
			ELSE 0 
		END as revenue
	FROM events
	WHERE event_type IN ('LEAD', 'PURCHASE')
	AND (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'%s
	AND (event_type = 'LEAD' OR (
		event_type = 'PURCHASE' 
		AND event_propeties->>'value' IS NOT NULL 
		AND event_propeties->>'value' != '' 
		AND event_propeties->>'value' ~ '^[0-9]+\.?[0-9]*$'
	))
	GROUP BY event_type, hour_str
	ORDER BY hour_str, event_type
	`, startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05"), professionFilter)

	type HourlyData struct {
		EventType string  `gorm:"column:event_type"`
		HourStr   string  `gorm:"column:hour_str"`
		Count     int64   `gorm:"column:count"`
		Revenue   float64 `gorm:"column:revenue"`
	}

	var results []HourlyData
	if err := r.db.Raw(query).Scan(&results).Error; err != nil {
		return result, fmt.Errorf("erro ao buscar dados por hora: %w", err)
	}

	// Processar resultados de forma otimizada
	for _, hourData := range results {
		// Validar hora se for hoje
		if isToday {
			hourInt, err := strconv.Atoi(hourData.HourStr)
			if err != nil || hourInt > now.Hour() {
				continue
			}
		}

		// Preencher dados baseado no tipo de evento
		if hourData.EventType == "LEAD" {
			result.LeadsByHour[hourData.HourStr] = hourData.Count
		} else if hourData.EventType == "PURCHASE" {
			result.PurchasesByHour[hourData.HourStr] = hourData.Count
			result.RevenueByHour[hourData.HourStr] = math.Round(hourData.Revenue*100) / 100
		}
	}

	return result, nil
}

// Métodos auxiliares
func (r *revenueRepository) calculateMetricComparison(current, previous int64) RevenueMetricResult {
	var percentage float64
	if previous > 0 {
		percentage = float64(current-previous) / float64(previous) * 100
		percentage = math.Round(percentage*100) / 100
	}

	return RevenueMetricResult{
		Current:      current,
		Previous:     previous,
		Percentage:   percentage,
		IsIncreasing: current > previous,
	}
}

func (r *revenueRepository) calculateFloatMetricComparison(current, previous float64) RevenueMetricResultFloat {
	var percentage float64
	if previous > 0 {
		percentage = (current - previous) / previous * 100
		percentage = math.Round(percentage*100) / 100
	}

	return RevenueMetricResultFloat{
		Current:      math.Round(current*100) / 100,
		Previous:     math.Round(previous*100) / 100,
		Percentage:   percentage,
		IsIncreasing: current > previous,
	}
}

func (r *revenueRepository) getProfessionSummaryOptimized(currentFrom, currentTo, previousFrom, previousTo time.Time) ([]ProfessionSummary, error) {
	// Query SUPER OTIMIZADA: uma única consulta para todos os dados por profissão
	query := fmt.Sprintf(`
	WITH profession_data AS (
		SELECT 
			profession_id,
			event_type,
			CASE 
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'current'
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'previous'
				ELSE NULL
			END as period,
			COUNT(*) as count,
			CASE 
				WHEN event_type = 'PURCHASE' THEN SUM(CAST(COALESCE(event_propeties->>'value', '0') AS DECIMAL(10,2)))
				ELSE 0 
			END as revenue
		FROM events 
		WHERE event_type IN ('LEAD', 'PURCHASE')
		AND (
			(event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'
			OR (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s'
		)
		AND (event_type = 'LEAD' OR (
			event_type = 'PURCHASE' 
			AND event_propeties->>'value' IS NOT NULL 
			AND event_propeties->>'value' != '' 
			AND event_propeties->>'value' ~ '^[0-9]+\.?[0-9]*$'
		))
		GROUP BY profession_id, event_type, 
			CASE 
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'current'
				WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'previous'
				ELSE NULL
			END
		HAVING CASE 
			WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'current'
			WHEN (event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN '%s' AND '%s' THEN 'previous'
			ELSE NULL
		END IS NOT NULL
	)
	SELECT 
		pd.profession_id,
		COALESCE(prof.profession_name, 'Profissão ' || pd.profession_id) as profession_name,
		SUM(CASE WHEN pd.period = 'current' AND pd.event_type = 'LEAD' THEN pd.count ELSE 0 END) as current_leads,
		SUM(CASE WHEN pd.period = 'current' AND pd.event_type = 'PURCHASE' THEN pd.count ELSE 0 END) as current_purchases,
		SUM(CASE WHEN pd.period = 'current' AND pd.event_type = 'PURCHASE' THEN pd.revenue ELSE 0 END) as current_revenue,
		SUM(CASE WHEN pd.period = 'previous' AND pd.event_type = 'LEAD' THEN pd.count ELSE 0 END) as previous_leads,
		SUM(CASE WHEN pd.period = 'previous' AND pd.event_type = 'PURCHASE' THEN pd.count ELSE 0 END) as previous_purchases,
		SUM(CASE WHEN pd.period = 'previous' AND pd.event_type = 'PURCHASE' THEN pd.revenue ELSE 0 END) as previous_revenue
	FROM profession_data pd
	LEFT JOIN professions prof ON pd.profession_id = prof.profession_id
	GROUP BY pd.profession_id, prof.profession_name
	HAVING SUM(CASE WHEN pd.period = 'current' THEN pd.count ELSE 0 END) > 0
	ORDER BY current_revenue DESC, current_leads DESC
	`,
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"),
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"),
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"),
		currentFrom.Format("2006-01-02 15:04:05"), currentTo.Format("2006-01-02 15:04:05"),
		previousFrom.Format("2006-01-02 15:04:05"), previousTo.Format("2006-01-02 15:04:05"))

	type ProfessionSummaryResult struct {
		ProfessionID      int     `gorm:"column:profession_id"`
		ProfessionName    string  `gorm:"column:profession_name"`
		CurrentLeads      int64   `gorm:"column:current_leads"`
		CurrentPurchases  int64   `gorm:"column:current_purchases"`
		CurrentRevenue    float64 `gorm:"column:current_revenue"`
		PreviousLeads     int64   `gorm:"column:previous_leads"`
		PreviousPurchases int64   `gorm:"column:previous_purchases"`
		PreviousRevenue   float64 `gorm:"column:previous_revenue"`
	}

	var results []ProfessionSummaryResult
	if err := r.db.Raw(query).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar resumo otimizado por profissão: %w", err)
	}

	// Pré-alocar slice para melhor performance
	summary := make([]ProfessionSummary, 0, len(results))

	// Converter para ProfessionSummary de forma otimizada
	for _, result := range results {
		summary = append(summary, ProfessionSummary{
			ProfessionID:   result.ProfessionID,
			ProfessionName: result.ProfessionName,
			Leads:          r.calculateMetricComparison(result.CurrentLeads, result.PreviousLeads),
			Purchases:      r.calculateMetricComparison(result.CurrentPurchases, result.PreviousPurchases),
			Revenue:        r.calculateFloatMetricComparison(result.CurrentRevenue, result.PreviousRevenue),
		})
	}

	return summary, nil
}
