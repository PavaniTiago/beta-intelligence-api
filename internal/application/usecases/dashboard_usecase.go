package usecases

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	// Necessário para acessar as estruturas de Session e Event
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	// Necessário para usar AdvancedFilter
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
	"gorm.io/gorm"
)

// DatePeriod representa um intervalo de datas com hora inicial e final
type DatePeriod struct {
	From     time.Time
	To       time.Time
	TimeFrom string
	TimeTo   string
}

// MetricResult representa um resultado numérico com comparação ao período anterior
type MetricResult struct {
	Current      int64   `json:"current"`
	Previous     int64   `json:"previous"`
	Percentage   float64 `json:"percentage"`
	IsIncreasing bool    `json:"is_increasing"`
}

// TimeSeriesPoint representa um ponto em uma série temporal
type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

// TimeSeriesData representa dados de séries temporais com comparação
type TimeSeriesData struct {
	Current  []TimeSeriesPoint `json:"current"`
	Previous []TimeSeriesPoint `json:"previous"`
}

// Adicionar estrutura para dados por hora
type HourlyMetrics struct {
	SessionsByHour       map[string]int64   `json:"sessions_by_hour"`
	LeadsByHour          map[string]int64   `json:"leads_by_hour"`
	ConversionRateByHour map[string]float64 `json:"conversion_rate_by_hour"`
}

// Métricas consolidadas para dashboard
type Metrics struct {
	Sessions             int64   `json:"sessions"`
	Leads                int64   `json:"leads"`
	ConversionRate       float64 `json:"conversion_rate"`
	PrevSessions         int64   `json:"prev_sessions"`
	PrevLeads            int64   `json:"prev_leads"`
	PrevConversionRate   float64 `json:"prev_conversion_rate"`
	SessionChange        float64 `json:"session_change"`
	LeadChange           float64 `json:"lead_change"`
	ConversionRateChange float64 `json:"conversion_rate_change"`
}

// Contagem por dia para dashboard
type DayCount struct {
	Day   string `json:"day"`
	Count int64  `json:"count"`
}

// Atualizando a estrutura DashboardResult para o novo formato
type DashboardResult struct {
	// Métricas principais
	Metrics       *Metrics         `json:"metrics"`
	SessionsByDay map[string]int64 `json:"sessions_by_day"`
	LeadsByDay    map[string]int64 `json:"leads_by_day"`

	// Dados por hora (opcional)
	HourlyData *HourlyMetrics `json:"hourly_data,omitempty"`

	// Mantendo campos originais para compatibilidade
	Sessions            MetricResult       `json:"sessions"`
	Leads               MetricResult       `json:"leads"`
	ConversionRate      MetricResult       `json:"conversion_rate"`
	PeriodCounts        map[string]int64   `json:"period_counts,omitempty"`
	Filters             map[string]string  `json:"filters"`
	ConversionRateByDay map[string]float64 `json:"conversion_rate_by_day,omitempty"`
}

// DashboardUseCase define a interface para operações do dashboard unificado
type DashboardUseCase interface {
	GetUnifiedDashboard(params map[string]string, currentPeriod DatePeriod, previousPeriod DatePeriod) (DashboardResult, error)
	GetProfessionConversionRates(currentPeriod DatePeriod, previousPeriod DatePeriod) (map[string]interface{}, error)
}

// ISessionRepository adiciona a interface do repositório de sessão necessária para otimização
type ISessionRepository interface {
	CountSessionsByDateRange(from, to time.Time, timeFrom, timeTo, userID, professionID, productID, funnelID string, landingPage string) (int64, error)
	GetSessionsCountByDays(from, to time.Time, timeFrom, timeTo, userID, professionID, productID, funnelID string, landingPage string) (map[string]int64, error)
	GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) ([]entities.Session, int64, error)
}

// IEventRepository adiciona a interface do repositório de eventos necessária para otimização
type IEventRepository interface {
	CountEventsByDateRange(from, to time.Time, timeFrom, timeTo string, eventType string, professionIDs, funnelIDs []int, logicalOperator string) (int64, error)
	GetEvents(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []repositories.AdvancedFilter, filterCondition string) ([]entities.Event, int64, error)
}

// Atualizar struct dashboardUseCase sem o campo etag
type dashboardUseCase struct {
	sessionRepository ISessionRepository
	eventRepository   IEventRepository
	db                *gorm.DB // Campo db para consultas diretas
}

// Atualizar o construtor sem o etag
func NewDashboardUseCase(sessionRepo ISessionRepository, eventRepo IEventRepository, db *gorm.DB) *dashboardUseCase {
	return &dashboardUseCase{
		sessionRepository: sessionRepo,
		eventRepository:   eventRepo,
		db:                db,
	}
}

// GetUnifiedDashboard implementa a lógica para obter todos os dados do dashboard
func (uc *dashboardUseCase) GetUnifiedDashboard(
	params map[string]string,
	currentPeriod DatePeriod,
	previousPeriod DatePeriod,
) (DashboardResult, error) {
	// Iniciar timer para medir tempo de processamento interno
	startTime := time.Now()

	// Verificar se o período é muito grande (limitar a 90 dias)
	maxDaysAllowed := 90
	if currentPeriod.To.Sub(currentPeriod.From).Hours() > float64(24*maxDaysAllowed) {
		return DashboardResult{}, fmt.Errorf("período muito longo (máximo %d dias). considere reduzir o intervalo ou usar agregação mensal", maxDaysAllowed)
	}

	// Verificar se é dia único
	isSingleDayQuery := currentPeriod.From.Year() == currentPeriod.To.Year() &&
		currentPeriod.From.Month() == currentPeriod.To.Month() &&
		currentPeriod.From.Day() == currentPeriod.To.Day()

	// Inicializar o resultado
	result := DashboardResult{
		Filters:             params,
		PeriodCounts:        make(map[string]int64),
		SessionsByDay:       make(map[string]int64),
		LeadsByDay:          make(map[string]int64),
		ConversionRateByDay: make(map[string]float64),
		Metrics:             &Metrics{}, // Inicializar o objeto Metrics
	}

	// Extrair parâmetros para as consultas
	professionID := params["profession_id"]
	funnelID := params["funnel_id"]
	landingPage := params["landingPage"]
	productID := params["product_id"]
	timeFrame := params["time_frame"]
	if timeFrame == "" {
		timeFrame = "daily"
	}

	// Para dias únicos, garantir que temos dados de comparação
	if isSingleDayQuery {
		// Verificar se temos período anterior definido
		if previousPeriod.From.IsZero() || previousPeriod.To.IsZero() {
			// Usar ontem como período de comparação
			ontem := currentPeriod.From.AddDate(0, 0, -1)
			previousPeriod.From = ontem
			previousPeriod.To = ontem
			log.Printf("Dia único sem período de comparação. Usando ontem: %s", ontem.Format("2006-01-02"))
		}
	}

	// Usar método otimizado para obter TODAS as contagens em uma única operação
	dashboardData, err := uc.getOptimizedDashboardData(
		currentPeriod,
		previousPeriod,
		professionID,
		funnelID,
		landingPage,
		productID,
	)

	if err != nil {
		return result, fmt.Errorf("erro ao obter dados otimizados: %w", err)
	}

	// Transferir dados otimizados para o resultado
	result.Sessions = dashboardData.Sessions
	result.Leads = dashboardData.Leads
	result.ConversionRate = dashboardData.ConversionRate
	result.PeriodCounts = dashboardData.PeriodCounts
	result.SessionsByDay = dashboardData.SessionsByDay
	result.LeadsByDay = dashboardData.LeadsByDay
	result.ConversionRateByDay = dashboardData.ConversionRateByDay
	result.HourlyData = dashboardData.HourlyData // Transferir dados por hora, se existirem

	// Preencher o objeto Metrics com os dados da resposta
	result.Metrics.Sessions = dashboardData.Sessions.Current
	result.Metrics.PrevSessions = dashboardData.Sessions.Previous
	result.Metrics.SessionChange = math.Round(dashboardData.Sessions.Percentage*100) / 100

	result.Metrics.Leads = dashboardData.Leads.Current
	result.Metrics.PrevLeads = dashboardData.Leads.Previous
	result.Metrics.LeadChange = math.Round(dashboardData.Leads.Percentage*100) / 100

	result.Metrics.ConversionRate = math.Round(float64(dashboardData.ConversionRate.Current)*100) / 100
	result.Metrics.PrevConversionRate = math.Round(float64(dashboardData.ConversionRate.Previous)*100) / 100
	result.Metrics.ConversionRateChange = math.Round(dashboardData.ConversionRate.Percentage*100) / 100

	// Se for um único dia, verificar se os totais batem com a soma das horas
	if isSingleDayQuery && result.HourlyData != nil {
		// Calcular totais dos dados horários
		var totalSessions, totalLeads int64
		for _, sessions := range result.HourlyData.SessionsByHour {
			totalSessions += sessions
		}
		for _, leads := range result.HourlyData.LeadsByHour {
			totalLeads += leads
		}

		// Buscar dados do dia anterior para comparação PRECISA
		ontem := currentPeriod.From.AddDate(0, 0, -1)
		log.Printf("Dia único detectado. Buscando dados do dia anterior: %s", ontem.Format("2006-01-02"))

		dadosOntem, err := uc.obterDadosComparacao(ontem, ontem, professionID, funnelID, landingPage, productID)

		var prevSessions, prevLeads int64
		var prevConvRate float64

		if err == nil && dadosOntem.Sessions > 0 {
			// Usar dados de ontem como valores de comparação
			prevSessions = dadosOntem.Sessions
			prevLeads = dadosOntem.Leads

			// Calcular taxa de conversão do dia anterior
			if prevSessions > 0 {
				prevConvRate = math.Round(float64(prevLeads)/float64(prevSessions)*10000) / 100
			}

			log.Printf("Dados do dia anterior encontrados: %d sessões, %d leads, taxa %.2f%%",
				prevSessions, prevLeads, prevConvRate)
		} else {
			// Tentar dados da semana anterior se não encontrou dados de ontem
			log.Printf("Não encontrou dados do dia anterior. Tentando semana anterior.")

			semanaAnterior := currentPeriod.From.AddDate(0, 0, -7)
			dadosSemanaAnterior, err := uc.obterDadosComparacao(semanaAnterior, semanaAnterior, professionID, funnelID, landingPage, productID)

			if err == nil && dadosSemanaAnterior.Sessions > 0 {
				log.Printf("Usando dados da semana anterior como comparação.")

				prevSessions = dadosSemanaAnterior.Sessions
				prevLeads = dadosSemanaAnterior.Leads

				if prevSessions > 0 {
					prevConvRate = math.Round(float64(prevLeads)/float64(prevSessions)*10000) / 100
				}

				log.Printf("Dados da semana anterior: %d sessões, %d leads, taxa %.2f%%",
					prevSessions, prevLeads, prevConvRate)
			} else {
				// Se não encontrou nenhum dado, usar zeros mesmo
				log.Printf("Não encontrou dados para comparação. Usando zeros.")
			}
		}

		// Se os totais forem maiores que os dados de métricas atuais, usar os totais
		if totalSessions > 0 && (result.Metrics.Sessions == 0 || totalSessions > result.Metrics.Sessions) {
			result.Metrics.Sessions = totalSessions
			result.Sessions.Current = totalSessions
		}
		if totalLeads > 0 && (result.Metrics.Leads == 0 || totalLeads > result.Metrics.Leads) {
			result.Metrics.Leads = totalLeads
			result.Leads.Current = totalLeads
		}

		// Recalcular taxa de conversão atual
		if result.Metrics.Sessions > 0 {
			convRate := float64(result.Metrics.Leads) / float64(result.Metrics.Sessions) * 100
			result.Metrics.ConversionRate = math.Round(convRate*100) / 100
			result.ConversionRate.Current = int64(math.Round(convRate))
		}

		// Definir dados de comparação
		result.Metrics.PrevSessions = prevSessions
		result.Metrics.PrevLeads = prevLeads
		result.Metrics.PrevConversionRate = prevConvRate

		// Atualizar campos de comparação em todos os lugares
		result.Sessions.Previous = prevSessions
		result.Leads.Previous = prevLeads
		result.ConversionRate.Previous = int64(math.Round(prevConvRate))

		// Recalcular percentagens de alteração apenas se tiver dados anteriores
		if prevSessions > 0 {
			sessionChange := float64(result.Metrics.Sessions-prevSessions) / float64(prevSessions) * 100
			result.Metrics.SessionChange = math.Round(sessionChange*100) / 100
			result.Sessions.Percentage = math.Round(sessionChange*100) / 100
			result.Sessions.IsIncreasing = result.Metrics.Sessions > prevSessions
		}

		if prevLeads > 0 {
			leadChange := float64(result.Metrics.Leads-prevLeads) / float64(prevLeads) * 100
			result.Metrics.LeadChange = math.Round(leadChange*100) / 100
			result.Leads.Percentage = math.Round(leadChange*100) / 100
			result.Leads.IsIncreasing = result.Metrics.Leads > prevLeads
		}

		if prevConvRate > 0 {
			convChange := (result.Metrics.ConversionRate - prevConvRate) / prevConvRate * 100
			result.Metrics.ConversionRateChange = math.Round(convChange*100) / 100
			result.ConversionRate.Percentage = math.Round(convChange*100) / 100
			result.ConversionRate.IsIncreasing = result.Metrics.ConversionRate > prevConvRate
		}
	}

	// Atualizar também o sessions_by_day
	if isSingleDayQuery {
		dayKey := currentPeriod.From.Format("2006-01-02")
		// Usar o valor calculado com os dados horários se estiver disponível
		if result.HourlyData != nil {
			result.SessionsByDay[dayKey] = result.Metrics.Sessions
			result.LeadsByDay[dayKey] = result.Metrics.Leads

			// Recalcular a taxa de conversão para o dia
			if result.Metrics.Sessions > 0 {
				convRate := float64(result.Metrics.Leads) / float64(result.Metrics.Sessions) * 100
				result.ConversionRateByDay[dayKey] = math.Round(convRate*100) / 100
			}
		}
	}

	// Garantir que todos os dias no intervalo tenham valores
	days := generateDateRange(currentPeriod.From, currentPeriod.To)
	for _, day := range days {
		if _, exists := result.SessionsByDay[day]; !exists {
			result.SessionsByDay[day] = 0
		}
		if _, exists := result.LeadsByDay[day]; !exists {
			result.LeadsByDay[day] = 0
		}

		// Calcular taxa de conversão para o dia
		sessions := result.SessionsByDay[day]
		leads := result.LeadsByDay[day]
		if sessions > 0 {
			convRate := float64(leads) / float64(sessions) * 100
			result.ConversionRateByDay[day] = math.Round(convRate*100) / 100
		} else {
			result.ConversionRateByDay[day] = 0
		}
	}

	// Adicionar tempo de processamento
	processingTime := time.Since(startTime).Milliseconds()
	result.Filters["processing_time_ms"] = fmt.Sprintf("%d", processingTime)

	return result, nil
}

// OptimizedDashboardData contém dados otimizados para o dashboard
type OptimizedDashboardData struct {
	Sessions            MetricResult
	Leads               MetricResult
	ConversionRate      MetricResult
	SessionsByDay       map[string]int64   // Adicionado para corresponder ao DashboardResult
	PeriodCounts        map[string]int64   // Alterado de []DayCount para map
	LeadsByDay          map[string]int64   // Alterado de []DayCount para map
	ConversionRateByDay map[string]float64 // Alterado para usar map
	HourlyData          *HourlyMetrics     // Alterado para usar HourlyMetrics
}

// getOptimizedDashboardData obtém todos os dados do dashboard em uma operação otimizada
func (uc *dashboardUseCase) getOptimizedDashboardData(
	currentPeriod DatePeriod,
	previousPeriod DatePeriod,
	professionID string,
	funnelID string,
	landingPage string,
	productID string,
) (OptimizedDashboardData, error) {
	// Iniciar timer para medir desempenho
	startTime := time.Now()

	result := OptimizedDashboardData{
		PeriodCounts:        make(map[string]int64),
		SessionsByDay:       make(map[string]int64),
		LeadsByDay:          make(map[string]int64),
		ConversionRateByDay: make(map[string]float64),
	}

	// Verificar se estamos consultando um único dia
	isSingleDayQuery := currentPeriod.From.Year() == currentPeriod.To.Year() &&
		currentPeriod.From.Month() == currentPeriod.To.Month() &&
		currentPeriod.From.Day() == currentPeriod.To.Day()

	// Para dias únicos, verificar se temos um período anterior adequado
	if isSingleDayQuery && (previousPeriod.From.IsZero() || previousPeriod.To.IsZero()) {
		// Se não foi fornecido um período anterior válido, usar dados de 7 dias atrás (semana anterior)
		previousPeriod.From = currentPeriod.From.AddDate(0, 0, -7)
		previousPeriod.To = currentPeriod.To.AddDate(0, 0, -7)
		log.Printf("Utilizando período anterior automático para dia único: %s até %s",
			previousPeriod.From.Format("2006-01-02"),
			previousPeriod.To.Format("2006-01-02"))
	}

	// Data formatada para registro
	currentPeriodStr := fmt.Sprintf("%s até %s",
		currentPeriod.From.Format("2006-01-02"),
		currentPeriod.To.Format("2006-01-02"))

	previousPeriodStr := fmt.Sprintf("%s até %s",
		previousPeriod.From.Format("2006-01-02"),
		previousPeriod.To.Format("2006-01-02"))

	fmt.Printf("Processando dashboard para período: %s (isSingleDay: %v), anterior: %s\n",
		currentPeriodStr, isSingleDayQuery, previousPeriodStr)

	// OTIMIZAÇÃO PRINCIPAL: Consulta unificada para sessões e leads
	// Isso elimina múltiplas chamadas ao banco de dados
	unifiedQuery := `
		WITH 
		-- Contagem de sessões atuais e anteriores
		session_counts AS (
			SELECT
				CASE
					WHEN "sessionStart" BETWEEN ? AND ? THEN 'current'
					WHEN "sessionStart" BETWEEN ? AND ? THEN 'previous'
				END AS periodo,
				COUNT(*) AS total
			FROM sessions
			WHERE ("sessionStart" BETWEEN ? AND ? OR "sessionStart" BETWEEN ? AND ?)
	`
	queryArgs := []interface{}{
		currentPeriod.From, currentPeriod.To, // current
		previousPeriod.From, previousPeriod.To, // previous
		currentPeriod.From, currentPeriod.To, // primeiro período WHERE
		previousPeriod.From, previousPeriod.To, // segundo período WHERE
	}

	// Adicionar filtros para sessões
	if landingPage != "" {
		unifiedQuery += ` AND "landingPage" = ?`
		queryArgs = append(queryArgs, landingPage)
	}
	if professionID != "" {
		unifiedQuery += ` AND profession_id = ?`
		queryArgs = append(queryArgs, professionID)
	}
	if productID != "" {
		unifiedQuery += ` AND product_id = ?`
		queryArgs = append(queryArgs, productID)
	}
	if funnelID != "" {
		unifiedQuery += ` AND funnel_id = ?`
		queryArgs = append(queryArgs, funnelID)
	}

	unifiedQuery += `
			GROUP BY periodo
		),
		-- Contagem de leads atuais e anteriores
		lead_counts AS (
			SELECT
				CASE
					WHEN event_time BETWEEN ? AND ? THEN 'current'
					WHEN event_time BETWEEN ? AND ? THEN 'previous'
				END AS periodo,
				COUNT(*) AS total
			FROM events
			WHERE (event_time BETWEEN ? AND ? OR event_time BETWEEN ? AND ?)
			AND event_type = 'LEAD'
	`
	queryArgs = append(queryArgs,
		currentPeriod.From, currentPeriod.To, // current
		previousPeriod.From, previousPeriod.To, // previous
		currentPeriod.From, currentPeriod.To, // primeiro período WHERE
		previousPeriod.From, previousPeriod.To, // segundo período WHERE
	)

	// Adicionar filtros para leads
	if professionID != "" {
		unifiedQuery += ` AND profession_id = ?`
		queryArgs = append(queryArgs, professionID)
	}
	if productID != "" {
		unifiedQuery += ` AND product_id = ?`
		queryArgs = append(queryArgs, productID)
	}
	if funnelID != "" {
		unifiedQuery += ` AND funnel_id = ?`
		queryArgs = append(queryArgs, funnelID)
	}

	unifiedQuery += `
			GROUP BY periodo
		),
		-- Contagem diária para o período atual
		daily_session_data AS (
			SELECT 
				to_char(date_trunc('day', "sessionStart" AT TIME ZONE 'America/Sao_Paulo'), 'YYYY-MM-DD') as dia,
				COUNT(*) as sessoes
			FROM sessions 
			WHERE "sessionStart" BETWEEN ? AND ?
	`
	queryArgs = append(queryArgs,
		currentPeriod.From, currentPeriod.To, // para sessões WHERE
	)

	// Adicionar filtros para sessões diárias
	if landingPage != "" {
		unifiedQuery += ` AND "landingPage" = ?`
		queryArgs = append(queryArgs, landingPage)
	}
	if professionID != "" {
		unifiedQuery += ` AND profession_id = ?`
		queryArgs = append(queryArgs, professionID)
	}
	if productID != "" {
		unifiedQuery += ` AND product_id = ?`
		queryArgs = append(queryArgs, productID)
	}
	if funnelID != "" {
		unifiedQuery += ` AND funnel_id = ?`
		queryArgs = append(queryArgs, funnelID)
	}

	unifiedQuery += `
			GROUP BY dia
		),
		daily_lead_data AS (
			SELECT 
				to_char(date_trunc('day', event_time AT TIME ZONE 'America/Sao_Paulo'), 'YYYY-MM-DD') as dia,
				COUNT(*) as leads
			FROM events 
			WHERE event_time BETWEEN ? AND ?
			AND event_type = 'LEAD'
	`
	queryArgs = append(queryArgs,
		currentPeriod.From, currentPeriod.To, // para leads WHERE
	)

	// Adicionar filtros para leads diários
	if professionID != "" {
		unifiedQuery += ` AND profession_id = ?`
		queryArgs = append(queryArgs, professionID)
	}
	if productID != "" {
		unifiedQuery += ` AND product_id = ?`
		queryArgs = append(queryArgs, productID)
	}
	if funnelID != "" {
		unifiedQuery += ` AND funnel_id = ?`
		queryArgs = append(queryArgs, funnelID)
	}

	unifiedQuery += `
			GROUP BY dia
		)
		SELECT 'counts' as type, periodo, total FROM session_counts
		UNION ALL
		SELECT 'leads' as type, periodo, total FROM lead_counts
		UNION ALL
		SELECT 'daily' as type, dia, sessoes FROM daily_session_data
		UNION ALL
		SELECT 'daily_leads' as type, dia, leads FROM daily_lead_data
	`

	// Estrutura para receber os resultados unificados
	type ResultRow struct {
		Type    string `gorm:"column:type"`
		Periodo string `gorm:"column:periodo"`
		Total   int64  `gorm:"column:total"`
	}

	var results []ResultRow
	if err := uc.db.Raw(unifiedQuery, queryArgs...).Scan(&results).Error; err != nil {
		return result, fmt.Errorf("erro na consulta unificada: %w", err)
	}

	// Processar resultados
	var currentSessions, previousSessions int64
	var currentLeads, previousLeads int64

	for _, row := range results {
		switch row.Type {
		case "counts":
			if row.Periodo == "current" {
				currentSessions = row.Total
			} else if row.Periodo == "previous" {
				previousSessions = row.Total
			}
		case "leads":
			if row.Periodo == "current" {
				currentLeads = row.Total
			} else if row.Periodo == "previous" {
				previousLeads = row.Total
			}
		case "daily":
			result.SessionsByDay[row.Periodo] = row.Total
			result.PeriodCounts[row.Periodo] = row.Total
		case "daily_leads":
			result.LeadsByDay[row.Periodo] = row.Total
		}
	}

	// Calcular taxas de conversão por dia
	for day, sessions := range result.PeriodCounts {
		leads := result.LeadsByDay[day]
		var conversionRate float64
		if sessions > 0 {
			conversionRate = float64(leads) / float64(sessions) * 100
			// Arredondar para duas casas decimais
			conversionRate = math.Round(conversionRate*100) / 100
		}
		result.ConversionRateByDay[day] = conversionRate
	}

	// Métricas de sessões
	var percentSessionChange float64
	if previousSessions > 0 {
		percentSessionChange = float64(currentSessions-previousSessions) / float64(previousSessions) * 100
		// Arredondar para duas casas decimais
		percentSessionChange = math.Round(percentSessionChange*100) / 100
	}
	result.Sessions = MetricResult{
		Current:      currentSessions,
		Previous:     previousSessions,
		Percentage:   percentSessionChange,
		IsIncreasing: currentSessions > previousSessions,
	}

	// Métricas de leads
	var percentLeadChange float64
	if previousLeads > 0 {
		percentLeadChange = float64(currentLeads-previousLeads) / float64(previousLeads) * 100
		// Arredondar para duas casas decimais
		percentLeadChange = math.Round(percentLeadChange*100) / 100
	}
	result.Leads = MetricResult{
		Current:      currentLeads,
		Previous:     previousLeads,
		Percentage:   percentLeadChange,
		IsIncreasing: currentLeads > previousLeads,
	}

	// Métricas de taxa de conversão
	var currentConversionRate, previousConversionRate float64
	if currentSessions > 0 {
		currentConversionRate = float64(currentLeads) / float64(currentSessions) * 100
		// Arredondar para duas casas decimais
		currentConversionRate = math.Round(currentConversionRate*100) / 100
	}
	if previousSessions > 0 {
		previousConversionRate = float64(previousLeads) / float64(previousSessions) * 100
		// Arredondar para duas casas decimais
		previousConversionRate = math.Round(previousConversionRate*100) / 100
	}

	var percentConversionChange float64
	if previousConversionRate > 0 {
		percentConversionChange = (currentConversionRate - previousConversionRate) / previousConversionRate * 100
		// Arredondar para duas casas decimais
		percentConversionChange = math.Round(percentConversionChange*100) / 100
	}

	result.ConversionRate = MetricResult{
		Current:      int64(math.Round(currentConversionRate)),
		Previous:     int64(math.Round(previousConversionRate)),
		Percentage:   percentConversionChange,
		IsIncreasing: currentConversionRate > previousConversionRate,
	}

	// Buscar dados por hora se for um único dia
	if isSingleDayQuery {
		hourlyData, err := uc.GetHourlyData(
			currentPeriod.From,
			currentPeriod.To,
			professionID,
			productID,
			funnelID,
			landingPage,
		)
		if err == nil {
			result.HourlyData = hourlyData
		} else {
			log.Printf("Erro ao buscar dados por hora: %v", err)
		}
	}

	// Loggar tempo de execução
	executionTime := time.Since(startTime)
	log.Printf("Consulta unificada do dashboard executada em %v", executionTime)

	return result, nil
}

// generateDateRange gera um array de strings de datas entre from e to
func generateDateRange(from, to time.Time) []string {
	var dates []string

	// Normalizar para início do dia
	current := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())

	// Incluir cada dia no intervalo
	for !current.After(end) {
		dates = append(dates, current.Format("2006-01-02"))
		current = current.AddDate(0, 0, 1) // Adicionar um dia
	}

	return dates
}

// GetHourlyData obtém dados agrupados por hora para um único dia
func (u *dashboardUseCase) GetHourlyData(
	fromDate time.Time,
	toDate time.Time,
	professionID,
	productID,
	funnelID string,
	landingPage string,
) (*HourlyMetrics, error) {
	startTime := time.Now()
	dateStr := fromDate.Format("2006-01-02")

	// Logging para debug
	log.Printf("Obtendo dados por hora para a data %s com landingPage=%s", dateStr, landingPage)

	result := &HourlyMetrics{
		SessionsByHour:       make(map[string]int64),
		LeadsByHour:          make(map[string]int64),
		ConversionRateByHour: make(map[string]float64),
	}

	// Verificar se é o dia atual
	now := time.Now()
	isToday := fromDate.Year() == now.Year() &&
		fromDate.Month() == now.Month() &&
		fromDate.Day() == now.Day()

	// Inicializar mapas com zeros para todas as horas
	var maxHour int
	if isToday {
		// Se for hoje, só inicializar até a hora atual
		maxHour = now.Hour() + 1
		if maxHour > 24 {
			maxHour = 24
		}
	} else {
		// Se não for hoje, inicializar todas as 24 horas
		maxHour = 24
	}

	for hour := 0; hour < maxHour; hour++ {
		hourStr := fmt.Sprintf("%02d", hour)
		result.SessionsByHour[hourStr] = 0
		result.LeadsByHour[hourStr] = 0
		result.ConversionRateByHour[hourStr] = 0
	}

	// OTIMIZAÇÃO: Usar queries diretas sem generate_series e LEFT JOIN
	startOfDay := time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, fromDate.Location())

	// Se for hoje, limitar a consulta até a hora atual
	var endOfDay time.Time
	if isToday {
		// Para o dia atual, limitar à hora atual
		endOfDay = time.Now()
		log.Printf("Dia atual detectado. Limitando consulta até: %v", endOfDay.Format("2006-01-02 15:04:05"))
	} else {
		// Para dias anteriores, usar o final do dia
		endOfDay = time.Date(toDate.Year(), toDate.Month(), toDate.Day(), 23, 59, 59, 999999999, toDate.Location())
	}

	// Query otimizada para sessions por hora (sem LEFT JOIN com generate_series)
	sessionHourlyQuery := `
		SELECT 
			to_char(date_trunc('hour', "sessionStart" AT TIME ZONE 'America/Sao_Paulo'), 'HH24') AS hour_str,
			COUNT(*) AS session_count
		FROM sessions
		WHERE "sessionStart" BETWEEN ? AND ?
	`

	// Parâmetros para a query
	sessionQueryArgs := []interface{}{
		startOfDay, endOfDay,
	}

	// Adicionar filtros para sessões
	if landingPage != "" {
		sessionHourlyQuery += ` AND "landingPage" = ?`
		sessionQueryArgs = append(sessionQueryArgs, landingPage)
	}
	if professionID != "" {
		sessionHourlyQuery += ` AND profession_id = ?`
		sessionQueryArgs = append(sessionQueryArgs, professionID)
	}
	if productID != "" {
		sessionHourlyQuery += ` AND product_id = ?`
		sessionQueryArgs = append(sessionQueryArgs, productID)
	}
	if funnelID != "" {
		sessionHourlyQuery += ` AND funnel_id = ?`
		sessionQueryArgs = append(sessionQueryArgs, funnelID)
	}

	// Agrupar e ordenar
	sessionHourlyQuery += `
		GROUP BY hour_str
		ORDER BY hour_str
	`

	// Debug do SQL gerado
	log.Printf("Query para sessões por hora: %s, params: %v", sessionHourlyQuery, sessionQueryArgs)

	// Estrutura para receber os resultados de sessões
	type HourlySessionData struct {
		HourStr      string `gorm:"column:hour_str"`
		SessionCount int64  `gorm:"column:session_count"`
	}

	// Executar a consulta para sessões
	var sessionResults []HourlySessionData
	if err := u.db.Raw(sessionHourlyQuery, sessionQueryArgs...).Scan(&sessionResults).Error; err != nil {
		log.Printf("Erro ao consultar sessões por hora: %v", err)
		return result, err
	}

	// Atualizar os dados de sessões
	for _, hourData := range sessionResults {
		hourInt, err := strconv.Atoi(hourData.HourStr)
		if err != nil {
			log.Printf("Erro ao converter hora %s: %v", hourData.HourStr, err)
			continue
		}

		// Ignorar horas futuras se for o dia atual
		if isToday && hourInt > now.Hour() {
			log.Printf("Ignorando dados futuros para hora %s (atual: %d)", hourData.HourStr, now.Hour())
			continue
		}

		result.SessionsByHour[hourData.HourStr] = hourData.SessionCount
	}

	// Query otimizada para leads por hora
	leadHourlyQuery := `
		SELECT 
			to_char(date_trunc('hour', event_time AT TIME ZONE 'America/Sao_Paulo'), 'HH24') AS hour_str,
			COUNT(*) AS lead_count
		FROM events
		WHERE event_time BETWEEN ? AND ?
		AND event_type = 'LEAD'
	`

	// Parâmetros para a query de leads
	leadQueryArgs := []interface{}{
		startOfDay, endOfDay,
	}

	// Adicionar filtros para leads
	if professionID != "" {
		leadHourlyQuery += ` AND profession_id = ?`
		leadQueryArgs = append(leadQueryArgs, professionID)
	}
	if productID != "" {
		leadHourlyQuery += ` AND product_id = ?`
		leadQueryArgs = append(leadQueryArgs, productID)
	}
	if funnelID != "" {
		leadHourlyQuery += ` AND funnel_id = ?`
		leadQueryArgs = append(leadQueryArgs, funnelID)
	}

	// Agrupar e ordenar
	leadHourlyQuery += `
		GROUP BY hour_str
		ORDER BY hour_str
	`

	// Debug do SQL gerado
	log.Printf("Query para leads por hora: %s, params: %v", leadHourlyQuery, leadQueryArgs)

	// Estrutura para receber os resultados de leads
	type HourlyLeadData struct {
		HourStr   string `gorm:"column:hour_str"`
		LeadCount int64  `gorm:"column:lead_count"`
	}

	// Executar a consulta para leads
	var leadResults []HourlyLeadData
	if err := u.db.Raw(leadHourlyQuery, leadQueryArgs...).Scan(&leadResults).Error; err != nil {
		log.Printf("Erro ao consultar leads por hora: %v", err)
		return result, err
	}

	// Atualizar os dados de leads
	for _, hourData := range leadResults {
		hourInt, err := strconv.Atoi(hourData.HourStr)
		if err != nil {
			log.Printf("Erro ao converter hora %s: %v", hourData.HourStr, err)
			continue
		}

		// Ignorar horas futuras se for o dia atual
		if isToday && hourInt > now.Hour() {
			log.Printf("Ignorando dados futuros para hora %s (atual: %d)", hourData.HourStr, now.Hour())
			continue
		}

		result.LeadsByHour[hourData.HourStr] = hourData.LeadCount
	}

	// Calcular taxas de conversão para cada hora
	for hour := 0; hour < maxHour; hour++ {
		hourStr := fmt.Sprintf("%02d", hour)
		sessions := result.SessionsByHour[hourStr]
		leads := result.LeadsByHour[hourStr]

		// Calcular taxa de conversão
		if sessions > 0 {
			convRate := float64(leads) / float64(sessions) * 100
			result.ConversionRateByHour[hourStr] = math.Round(convRate*100) / 100
		}
	}

	// Registrar tempo total para fins de performance
	duration := time.Since(startTime)
	log.Printf("Dados por hora calculados em %v: %d sessões, %d leads no total",
		duration, countMapValues(result.SessionsByHour), countMapValues(result.LeadsByHour))

	return result, nil
}

// Função auxiliar para contar valores em um mapa
func countMapValues(data map[string]int64) int64 {
	var total int64
	for _, value := range data {
		total += value
	}
	return total
}

// obterDadosComparacao obtém os dados de um dia específico para comparação
func (uc *dashboardUseCase) obterDadosComparacao(
	from time.Time,
	to time.Time,
	professionID string,
	funnelID string,
	landingPage string,
	productID string,
) (struct{ Sessions, Leads int64 }, error) {
	result := struct{ Sessions, Leads int64 }{}

	// Ajustar para dia completo (início e fim do dia)
	startOfDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	endOfDay := time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())

	log.Printf("Buscando dados de comparação para o período: %s até %s",
		startOfDay.Format("2006-01-02 15:04:05"),
		endOfDay.Format("2006-01-02 15:04:05"))

	// Query simplificada para contar sessões
	sessionQuery := `
		SELECT COUNT(*) as total
		FROM sessions
		WHERE "sessionStart" BETWEEN ? AND ?
	`
	sessionQueryArgs := []interface{}{
		startOfDay, endOfDay,
	}

	// Adicionar filtros para sessões
	if landingPage != "" {
		sessionQuery += ` AND "landingPage" = ?`
		sessionQueryArgs = append(sessionQueryArgs, landingPage)
	}
	if professionID != "" {
		sessionQuery += ` AND profession_id = ?`
		sessionQueryArgs = append(sessionQueryArgs, professionID)
	}
	if productID != "" {
		sessionQuery += ` AND product_id = ?`
		sessionQueryArgs = append(sessionQueryArgs, productID)
	}
	if funnelID != "" {
		sessionQuery += ` AND funnel_id = ?`
		sessionQueryArgs = append(sessionQueryArgs, funnelID)
	}

	// Executar query de sessões
	if err := uc.db.Raw(sessionQuery, sessionQueryArgs...).Scan(&result.Sessions).Error; err != nil {
		return result, err
	}

	// Query simplificada para contar leads
	leadQuery := `
		SELECT COUNT(*) as total
		FROM events
		WHERE event_time BETWEEN ? AND ?
		AND event_type = 'LEAD'
	`
	leadQueryArgs := []interface{}{
		startOfDay, endOfDay,
	}

	// Adicionar filtros para leads
	if professionID != "" {
		leadQuery += ` AND profession_id = ?`
		leadQueryArgs = append(leadQueryArgs, professionID)
	}
	if productID != "" {
		leadQuery += ` AND product_id = ?`
		leadQueryArgs = append(leadQueryArgs, productID)
	}
	if funnelID != "" {
		leadQuery += ` AND funnel_id = ?`
		leadQueryArgs = append(leadQueryArgs, funnelID)
	}

	// Executar query de leads
	if err := uc.db.Raw(leadQuery, leadQueryArgs...).Scan(&result.Leads).Error; err != nil {
		return result, err
	}

	log.Printf("Dados de comparação encontrados: %d sessões, %d leads", result.Sessions, result.Leads)

	return result, nil
}

// ProfessionConversionData representa os dados de conversão para uma profissão
type ProfessionConversionData struct {
	ProfessionID   int                   `json:"profession_id"`
	ProfessionName string                `json:"profession_name"`
	ConversionRate float64               `json:"conversion_rate"`
	PreviousRate   float64               `json:"previous_rate"`
	Growth         float64               `json:"growth"`
	IsIncreasing   bool                  `json:"is_increasing"`
	IsActive       bool                  `json:"is_active"`
	ActiveFunnels  []ActiveFunnelDetails `json:"active_funnels,omitempty"`
}

// ActiveFunnelDetails representa os detalhes de um funil ativo
type ActiveFunnelDetails struct {
	FunnelID       int     `json:"funnel_id"`
	FunnelName     string  `json:"funnel_name"`
	ConversionRate float64 `json:"conversion_rate"`
}

// Estruturas para agregação de dados por profissão
type SessionAggregate struct {
	ProfessionID int   `gorm:"column:profession_id"`
	Count        int64 `gorm:"column:count"`
}

type LeadAggregate struct {
	ProfessionID int   `gorm:"column:profession_id"`
	Count        int64 `gorm:"column:count"`
}

// GetProfessionConversionRates implementa a lógica para obter taxas de conversão para todas as profissões
// Versão otimizada usando GROUP BY e reduzindo o número de queries ao banco de dados
func (uc *dashboardUseCase) GetProfessionConversionRates(
	currentPeriod DatePeriod,
	previousPeriod DatePeriod,
) (map[string]interface{}, error) {
	// Iniciar timer para medir tempo de processamento
	startTime := time.Now()

	// Consultar todas as profissões no sistema, excluindo as que estão em teste
	var professions []struct {
		ProfessionID   int    `gorm:"column:profession_id"`
		ProfessionName string `gorm:"column:profession_name"`
	}

	// Construir a consulta com base na existência da coluna is_testing
	query := uc.db.Table("professions").Select("profession_id, profession_name")

	query = query.Where("is_testing IS NULL OR is_testing = ?", false)

	// Executar a consulta
	if err := query.Find(&professions).Error; err != nil {
		return nil, fmt.Errorf("erro ao consultar profissões: %w", err)
	}

	// Criar mapa de ID -> Nome para uso posterior
	professionMap := make(map[int]string)
	for _, p := range professions {
		professionMap[p.ProfessionID] = p.ProfessionName
	}

	// Consultar funis ativos por profissão para verificar se todos os funis estão inativos
	activeFunnelsQuery := `
		WITH funnels_by_profession AS (
			SELECT 
				prof.profession_id,
				prof.profession_name,
				f.funnel_id,
				f.is_active
			FROM professions prof
			JOIN products p ON prof.profession_id = p.profession_id
			JOIN funnels f ON p.product_id = f.product_id
			WHERE (f.is_testing = false OR f.is_testing IS NULL)
		)
		SELECT 
			profession_id,
			CASE 
				WHEN COUNT(funnel_id) > 0 AND SUM(CASE WHEN is_active = true THEN 1 ELSE 0 END) = 0 THEN false
				ELSE true
			END AS has_any_active_funnels
		FROM funnels_by_profession
		GROUP BY profession_id
	`

	// Obter detalhes de cada funil ativo com taxa de conversão
	funnelDetailsQuery := fmt.Sprintf(`
		WITH date_params AS (
			SELECT 
				TIMESTAMP '%s' as from_date,
				TIMESTAMP '%s' as to_date
		),
		funnel_sessions AS (
			SELECT 
				f.funnel_id,
				f.funnel_name,
				p.profession_id,
				COUNT(*) as session_count
			FROM funnels f
			JOIN products p ON f.product_id = p.product_id
			LEFT JOIN sessions s ON f.funnel_id = s.funnel_id
			WHERE f.is_active = true
			AND (f.is_testing = false OR f.is_testing IS NULL)
			AND s."sessionStart" IS NOT NULL
			AND (s."sessionStart" AT TIME ZONE 'America/Sao_Paulo') 
				BETWEEN (SELECT from_date FROM date_params) AND (SELECT to_date FROM date_params)
			GROUP BY f.funnel_id, f.funnel_name, p.profession_id
		),
		funnel_leads AS (
			SELECT 
				f.funnel_id,
				COUNT(*) as lead_count
			FROM funnels f
			JOIN events e ON f.funnel_id = e.funnel_id
			WHERE f.is_active = true
			AND (f.is_testing = false OR f.is_testing IS NULL)
			AND e.event_type = 'LEAD'
			AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') 
				BETWEEN (SELECT from_date FROM date_params) AND (SELECT to_date FROM date_params)
			GROUP BY f.funnel_id
		)
		SELECT 
			fs.funnel_id,
			fs.funnel_name,
			fs.profession_id,
			fs.session_count,
			COALESCE(fl.lead_count, 0) as lead_count,
			CASE 
				WHEN fs.session_count > 0 
				THEN (ROUND((COALESCE(fl.lead_count, 0)::float / fs.session_count::float) * 100))::numeric(10,2)
				ELSE 0 
			END as conversion_rate
		FROM funnel_sessions fs
		LEFT JOIN funnel_leads fl ON fs.funnel_id = fl.funnel_id
		ORDER BY fs.profession_id, conversion_rate DESC
	`, currentPeriod.From.Format("2006-01-02 15:04:05"), currentPeriod.To.Format("2006-01-02 15:04:05"))

	type ActiveFunnelResult struct {
		ProfessionID        int  `gorm:"column:profession_id"`
		HasAnyActiveFunnels bool `gorm:"column:has_any_active_funnels"`
	}

	var activeFunnelsResults []ActiveFunnelResult
	if err := uc.db.Raw(activeFunnelsQuery).Scan(&activeFunnelsResults).Error; err != nil {
		return nil, fmt.Errorf("erro ao verificar funis ativos: %w", err)
	}

	// Criar mapa para facilitar o acesso à informação de funis ativos
	professionToActiveFunnels := make(map[int]bool)
	for _, result := range activeFunnelsResults {
		professionToActiveFunnels[result.ProfessionID] = result.HasAnyActiveFunnels
	}

	// 1. QUERY OTIMIZADA: Contagem de sessões para o período atual agrupadas por profissão
	var sessionsCurrentPeriod []SessionAggregate

	// Formatar datas diretamente para timestamp com timezone
	var currentFromTime, currentToTime string
	if currentPeriod.TimeFrom != "" {
		currentFromTime = currentPeriod.TimeFrom + ":00"
	} else {
		currentFromTime = "00:00:00"
	}

	if currentPeriod.TimeTo != "" {
		currentToTime = currentPeriod.TimeTo + ":59"
	} else {
		currentToTime = "23:59:59"
	}

	currentFromTimestamp := fmt.Sprintf("%s %s",
		currentPeriod.From.Format("2006-01-02"),
		currentFromTime)

	currentToTimestamp := fmt.Sprintf("%s %s",
		currentPeriod.To.Format("2006-01-02"),
		currentToTime)

	// Executar consulta de sessões atuais
	sessionsCurrentSQL := fmt.Sprintf(`
		SELECT profession_id, COUNT(*) as count
		FROM sessions
		WHERE "landingPage" = 'lp.vagasjustica.com.br'
		AND ("sessionStart" AT TIME ZONE 'America/Sao_Paulo')
		    BETWEEN TIMESTAMP '%s' AND TIMESTAMP '%s'
		GROUP BY profession_id`,
		currentFromTimestamp,
		currentToTimestamp)

	if err := uc.db.Raw(sessionsCurrentSQL).Scan(&sessionsCurrentPeriod).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar sessões do período atual: %w", err)
	}

	// 2. QUERY OTIMIZADA: Contagem de sessões para o período anterior agrupadas por profissão
	var sessionsPreviousPeriod []SessionAggregate

	// Formatar datas para período anterior
	var previousFromTime, previousToTime string
	if previousPeriod.TimeFrom != "" {
		previousFromTime = previousPeriod.TimeFrom + ":00"
	} else {
		previousFromTime = "00:00:00"
	}

	if previousPeriod.TimeTo != "" {
		previousToTime = previousPeriod.TimeTo + ":59"
	} else {
		previousToTime = "23:59:59"
	}

	previousFromTimestamp := fmt.Sprintf("%s %s",
		previousPeriod.From.Format("2006-01-02"),
		previousFromTime)

	previousToTimestamp := fmt.Sprintf("%s %s",
		previousPeriod.To.Format("2006-01-02"),
		previousToTime)

	// Executar consulta de sessões anteriores
	sessionsPreviousSQL := fmt.Sprintf(`
		SELECT profession_id, COUNT(*) as count
		FROM sessions
		WHERE "landingPage" = 'lp.vagasjustica.com.br'
		AND ("sessionStart" AT TIME ZONE 'America/Sao_Paulo')
		    BETWEEN TIMESTAMP '%s' AND TIMESTAMP '%s'
		GROUP BY profession_id`,
		previousFromTimestamp,
		previousToTimestamp)

	if err := uc.db.Raw(sessionsPreviousSQL).Scan(&sessionsPreviousPeriod).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar sessões do período anterior: %w", err)
	}

	// 3. QUERY OTIMIZADA: Contagem de leads para o período atual agrupadas por profissão
	var leadsCurrentPeriod []LeadAggregate

	// Executar consulta de leads atuais
	leadsCurrentSQL := fmt.Sprintf(`
		SELECT profession_id, COUNT(*) as count
		FROM events
		WHERE event_type = 'LEAD'
		AND (event_time AT TIME ZONE 'America/Sao_Paulo')
		    BETWEEN TIMESTAMP '%s' AND TIMESTAMP '%s'
		GROUP BY profession_id`,
		currentFromTimestamp,
		currentToTimestamp)

	if err := uc.db.Raw(leadsCurrentSQL).Scan(&leadsCurrentPeriod).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar leads do período atual: %w", err)
	}

	// 4. QUERY OTIMIZADA: Contagem de leads para o período anterior agrupadas por profissão
	var leadsPreviousPeriod []LeadAggregate

	// Executar consulta de leads anteriores
	leadsPreviousSQL := fmt.Sprintf(`
		SELECT profession_id, COUNT(*) as count
		FROM events
		WHERE event_type = 'LEAD'
		AND (event_time AT TIME ZONE 'America/Sao_Paulo')
		    BETWEEN TIMESTAMP '%s' AND TIMESTAMP '%s'
		GROUP BY profession_id`,
		previousFromTimestamp,
		previousToTimestamp)

	if err := uc.db.Raw(leadsPreviousSQL).Scan(&leadsPreviousPeriod).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar leads do período anterior: %w", err)
	}

	// Agora que temos os detalhes dos funis ativos, podemos completar a consulta
	// Verificar se as datas usadas na query de funis são as mesmas
	funnelDetailsQuery = fmt.Sprintf(`
		WITH date_params AS (
			SELECT 
				TIMESTAMP '%s' as from_date,
				TIMESTAMP '%s' as to_date
		),
		funnel_sessions AS (
			SELECT 
				f.funnel_id,
				f.funnel_name,
				p.profession_id,
				COUNT(*) as session_count
			FROM funnels f
			JOIN products p ON f.product_id = p.product_id
			LEFT JOIN sessions s ON f.funnel_id = s.funnel_id
			WHERE f.is_active = true
			AND (f.is_testing = false OR f.is_testing IS NULL)
			AND s."sessionStart" IS NOT NULL
			AND (s."sessionStart" AT TIME ZONE 'America/Sao_Paulo') 
				BETWEEN (SELECT from_date FROM date_params) AND (SELECT to_date FROM date_params)
			GROUP BY f.funnel_id, f.funnel_name, p.profession_id
		),
		funnel_leads AS (
			SELECT 
				f.funnel_id,
				COUNT(*) as lead_count
			FROM funnels f
			JOIN events e ON f.funnel_id = e.funnel_id
			WHERE f.is_active = true
			AND (f.is_testing = false OR f.is_testing IS NULL)
			AND e.event_type = 'LEAD'
			AND (e.event_time AT TIME ZONE 'America/Sao_Paulo') 
				BETWEEN (SELECT from_date FROM date_params) AND (SELECT to_date FROM date_params)
			GROUP BY f.funnel_id
		)
		SELECT 
			fs.funnel_id,
			fs.funnel_name,
			fs.profession_id,
			fs.session_count,
			COALESCE(fl.lead_count, 0) as lead_count,
			CASE 
				WHEN fs.session_count > 0 
				THEN (ROUND((COALESCE(fl.lead_count, 0)::float / fs.session_count::float) * 100))::numeric(10,2)
				ELSE 0 
			END as conversion_rate
		FROM funnel_sessions fs
		LEFT JOIN funnel_leads fl ON fs.funnel_id = fl.funnel_id
		ORDER BY fs.profession_id, conversion_rate DESC
	`, currentFromTimestamp, currentToTimestamp)

	// Mapear os resultados das queries para mapas em memória
	currentSessionsByProfession := make(map[int]int64)
	for _, s := range sessionsCurrentPeriod {
		currentSessionsByProfession[s.ProfessionID] = s.Count
	}

	previousSessionsByProfession := make(map[int]int64)
	for _, s := range sessionsPreviousPeriod {
		previousSessionsByProfession[s.ProfessionID] = s.Count
	}

	currentLeadsByProfession := make(map[int]int64)
	for _, l := range leadsCurrentPeriod {
		currentLeadsByProfession[l.ProfessionID] = l.Count
	}

	previousLeadsByProfession := make(map[int]int64)
	for _, l := range leadsPreviousPeriod {
		previousLeadsByProfession[l.ProfessionID] = l.Count
	}

	// Buscar detalhes dos funis
	type FunnelDetail struct {
		FunnelID       int     `gorm:"column:funnel_id"`
		FunnelName     string  `gorm:"column:funnel_name"`
		ProfessionID   int     `gorm:"column:profession_id"`
		SessionCount   int64   `gorm:"column:session_count"`
		LeadCount      int64   `gorm:"column:lead_count"`
		ConversionRate float64 `gorm:"column:conversion_rate"`
	}

	var funnelDetails []FunnelDetail
	if err := uc.db.Raw(funnelDetailsQuery).Scan(&funnelDetails).Error; err != nil {
		return nil, fmt.Errorf("erro ao buscar detalhes dos funis: %w", err)
	}

	// Criar mapa para agrupar funis por profissão
	professionToFunnels := make(map[int][]ActiveFunnelDetails)
	for _, detail := range funnelDetails {
		funnelDetail := ActiveFunnelDetails{
			FunnelID:       detail.FunnelID,
			FunnelName:     detail.FunnelName,
			ConversionRate: detail.ConversionRate,
		}

		if _, exists := professionToFunnels[detail.ProfessionID]; !exists {
			professionToFunnels[detail.ProfessionID] = []ActiveFunnelDetails{}
		}

		professionToFunnels[detail.ProfessionID] = append(
			professionToFunnels[detail.ProfessionID],
			funnelDetail,
		)
	}

	// Processar e calcular resultados
	professionData := make(map[string]ProfessionConversionData)

	// Processar todas as profissões
	for profID, profName := range professionMap {
		// Obter contagens dos mapas
		currentSessions := currentSessionsByProfession[profID]
		previousSessions := previousSessionsByProfession[profID]
		currentLeads := currentLeadsByProfession[profID]
		previousLeads := previousLeadsByProfession[profID]

		// Calcular taxa de conversão atual
		var currentRate float64
		if currentSessions > 0 {
			currentRate = float64(currentLeads) / float64(currentSessions) * 100
		}

		// Calcular taxa de conversão anterior
		var previousRate float64
		if previousSessions > 0 {
			previousRate = float64(previousLeads) / float64(previousSessions) * 100
		}

		// Calcular crescimento
		var growth float64
		var isIncreasing bool

		if previousRate > 0 {
			growth = ((currentRate - previousRate) / previousRate) * 100
			isIncreasing = growth > 0
		} else if currentRate > 0 {
			growth = 100 // Se não havia conversão antes e agora tem, considerar 100% de crescimento
			isIncreasing = true
		} else {
			growth = 0
			isIncreasing = false
		}

		// Arredondar valores para melhor visualização
		currentRate = math.Round(currentRate*100) / 100
		previousRate = math.Round(previousRate*100) / 100
		growth = math.Round(growth*100) / 100

		// Adicionar ao mapa resultante, incluindo agora o status de funnels ativos
		professionData[profName] = ProfessionConversionData{
			ProfessionID:   profID,
			ProfessionName: profName,
			ConversionRate: currentRate,
			PreviousRate:   previousRate,
			Growth:         growth,
			IsIncreasing:   isIncreasing,
			IsActive:       professionToActiveFunnels[profID],
			ActiveFunnels:  professionToFunnels[profID],
		}
	}

	// Construir resultado final
	result := make(map[string]interface{})
	result["professions"] = professionData
	result["processing_time_ms"] = time.Since(startTime).Milliseconds()
	result["queries_executed"] = 6 // Agora são 6 queries (adicionamos a consulta de funis ativos e detalhes de conversão)

	// Adicionar metadados sobre os funis
	funnelCounts := make(map[string]interface{})
	totalActiveFunnels := 0
	professionsWithFunnels := 0

	for _, funnels := range professionToFunnels {
		if len(funnels) > 0 {
			professionsWithFunnels++
		}
		totalActiveFunnels += len(funnels)
	}

	funnelCounts["total_active_funnels"] = totalActiveFunnels
	funnelCounts["professions_with_funnels"] = professionsWithFunnels
	result["funnel_stats"] = funnelCounts

	return result, nil
}
