package usecases

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

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

// Modificar a estrutura DashboardResult para incluir dados por hora
type DashboardResult struct {
	// Métricas principais
	Sessions MetricResult `json:"sessions"`
	Leads    MetricResult `json:"leads"`

	// Contagens por período
	PeriodCounts        map[string]int64   `json:"period_counts,omitempty"`
	LeadsByDay          map[string]int64   `json:"leads_by_day,omitempty"`
	ConversionRateByDay map[string]float64 `json:"conversion_rate_by_day,omitempty"`

	// Dados de conversão
	ConversionRate MetricResult `json:"conversion_rate"`

	// Parâmetros usados na consulta
	Filters map[string]string `json:"filters"`

	// Novo campo para dados por hora
	HourlyData *HourlyMetrics `json:"hourly_data,omitempty"`
}

// DashboardUseCase define a interface para operações do dashboard unificado
type DashboardUseCase interface {
	GetUnifiedDashboard(params map[string]string, currentPeriod DatePeriod, previousPeriod DatePeriod) (DashboardResult, error)
}

// ISessionRepository adiciona a interface do repositório de sessão necessária para otimização
type ISessionRepository interface {
	CountSessionsByDateRange(from, to time.Time, timeFrom, timeTo, userID, professionID, productID, funnelID string, landingPage string) (int64, error)
	GetSessionsCountByDays(from, to time.Time, timeFrom, timeTo, userID, professionID, productID, funnelID string, landingPage string) (map[string]int64, error)
}

// IEventRepository adiciona a interface do repositório de eventos necessária para otimização
type IEventRepository interface {
	CountEventsByDateRange(from, to time.Time, timeFrom, timeTo string, eventType string, professionIDs, funnelIDs []int, logicalOperator string) (int64, error)
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

	// Inicializar o resultado
	result := DashboardResult{
		Filters:             params,
		PeriodCounts:        make(map[string]int64),
		LeadsByDay:          make(map[string]int64),
		ConversionRateByDay: make(map[string]float64),
	}

	// Extrair parâmetros para as consultas
	professionID := params["profession_id"]
	funnelID := params["funnel_id"]
	landingPage := params["landingPage"]
	userID := params["user_id"]
	productID := params["product_id"]
	timeFrame := params["time_frame"]
	if timeFrame == "" {
		timeFrame = "daily"
	}

	// Usar método otimizado para obter TODAS as contagens em uma única operação
	dashboardData, err := uc.getOptimizedDashboardData(
		currentPeriod,
		previousPeriod,
		professionID,
		funnelID,
		landingPage,
		userID,
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
	result.LeadsByDay = dashboardData.LeadsByDay
	result.ConversionRateByDay = dashboardData.ConversionRateByDay
	result.HourlyData = dashboardData.HourlyData // Transferir dados por hora, se existirem

	// Adicionar tempo de processamento
	processingTime := time.Since(startTime).Milliseconds()
	result.Filters["processing_time_ms"] = fmt.Sprintf("%d", processingTime)

	return result, nil
}

// Estrutura para armazenar os dados otimizados do dashboard
type OptimizedDashboardData struct {
	Sessions            MetricResult
	Leads               MetricResult
	ConversionRate      MetricResult
	PeriodCounts        map[string]int64
	LeadsByDay          map[string]int64
	ConversionRateByDay map[string]float64
	HourlyData          *HourlyMetrics // Dados por hora (presente apenas quando consulta é para um único dia)
}

// getOptimizedDashboardData obtém todos os dados do dashboard em uma operação otimizada
func (uc *dashboardUseCase) getOptimizedDashboardData(
	currentPeriod DatePeriod,
	previousPeriod DatePeriod,
	professionID string,
	funnelID string,
	landingPage string,
	userID string,
	productID string,
) (OptimizedDashboardData, error) {
	// Iniciar timer para medir desempenho de cada operação
	startTime := time.Now()
	queryTimes := make(map[string]time.Duration)

	result := OptimizedDashboardData{
		PeriodCounts:        make(map[string]int64),
		LeadsByDay:          make(map[string]int64),
		ConversionRateByDay: make(map[string]float64),
	}

	// Verificar se estamos consultando um único dia
	isSingleDayQuery := currentPeriod.From.Year() == currentPeriod.To.Year() &&
		currentPeriod.From.Month() == currentPeriod.To.Month() &&
		currentPeriod.From.Day() == currentPeriod.To.Day()

	// Data formatada para registro
	currentPeriodStr := fmt.Sprintf("%s até %s",
		currentPeriod.From.Format("2006-01-02"),
		currentPeriod.To.Format("2006-01-02"))

	fmt.Printf("Processando dashboard para período: %s (isSingleDay: %v)\n", currentPeriodStr, isSingleDayQuery)

	// Execução paralela de consultas para máxima performance
	var wg sync.WaitGroup
	var errorMutex sync.Mutex
	var resultMutex sync.Mutex
	var errors []error

	// 1. Obter contagem de sessões atuais
	wg.Add(1)
	var currentSessions int64
	go func() {
		defer wg.Done()

		queryStart := time.Now()
		count, err := uc.sessionRepository.CountSessionsByDateRange(
			currentPeriod.From,
			currentPeriod.To,
			currentPeriod.TimeFrom,
			currentPeriod.TimeTo,
			userID,
			professionID,
			productID,
			funnelID,
			landingPage,
		)

		resultMutex.Lock()
		queryTimes["current_sessions"] = time.Since(queryStart)
		resultMutex.Unlock()

		if err != nil {
			errorMutex.Lock()
			errors = append(errors, fmt.Errorf("erro ao contar sessões atuais: %w", err))
			errorMutex.Unlock()
			return
		}
		resultMutex.Lock()
		currentSessions = count
		resultMutex.Unlock()
	}()

	// 2. Obter contagem de sessões anteriores
	wg.Add(1)
	var previousSessions int64
	go func() {
		defer wg.Done()
		count, err := uc.sessionRepository.CountSessionsByDateRange(
			previousPeriod.From,
			previousPeriod.To,
			previousPeriod.TimeFrom,
			previousPeriod.TimeTo,
			userID,
			professionID,
			productID,
			funnelID,
			landingPage,
		)
		if err != nil {
			errorMutex.Lock()
			errors = append(errors, fmt.Errorf("erro ao contar sessões anteriores: %w", err))
			errorMutex.Unlock()
			return
		}
		resultMutex.Lock()
		previousSessions = count
		resultMutex.Unlock()
	}()

	// 3. Obter contagem de sessões por dia
	wg.Add(1)
	go func() {
		defer wg.Done()
		periodCounts, err := uc.sessionRepository.GetSessionsCountByDays(
			currentPeriod.From,
			currentPeriod.To,
			currentPeriod.TimeFrom,
			currentPeriod.TimeTo,
			userID,
			professionID,
			productID,
			funnelID,
			landingPage,
		)
		if err != nil {
			errorMutex.Lock()
			errors = append(errors, fmt.Errorf("erro ao contar sessões por dia: %w", err))
			errorMutex.Unlock()
			return
		}
		resultMutex.Lock()
		result.PeriodCounts = periodCounts
		resultMutex.Unlock()
	}()

	// 4 e 5. Obter contagem de leads atual e anterior
	var professionIDs, funnelIDs []int
	if professionID != "" {
		profID, err := strconv.Atoi(professionID)
		if err == nil && profID > 0 {
			professionIDs = []int{profID}
		}
	}
	if funnelID != "" {
		funID, err := strconv.Atoi(funnelID)
		if err == nil && funID > 0 {
			funnelIDs = []int{funID}
		}
	}

	// 4. Contagem de leads atual
	wg.Add(1)
	var currentLeads int64
	go func() {
		defer wg.Done()
		count, err := uc.eventRepository.CountEventsByDateRange(
			currentPeriod.From,
			currentPeriod.To,
			currentPeriod.TimeFrom,
			currentPeriod.TimeTo,
			"LEAD",
			professionIDs,
			funnelIDs,
			"AND",
		)
		if err != nil {
			errorMutex.Lock()
			errors = append(errors, fmt.Errorf("erro ao contar leads atuais: %w", err))
			errorMutex.Unlock()
			return
		}
		resultMutex.Lock()
		currentLeads = count
		resultMutex.Unlock()
	}()

	// 5. Contagem de leads anterior
	wg.Add(1)
	var previousLeads int64
	go func() {
		defer wg.Done()
		count, err := uc.eventRepository.CountEventsByDateRange(
			previousPeriod.From,
			previousPeriod.To,
			previousPeriod.TimeFrom,
			previousPeriod.TimeTo,
			"LEAD",
			professionIDs,
			funnelIDs,
			"AND",
		)
		if err != nil {
			errorMutex.Lock()
			errors = append(errors, fmt.Errorf("erro ao contar leads anteriores: %w", err))
			errorMutex.Unlock()
			return
		}
		resultMutex.Lock()
		previousLeads = count
		resultMutex.Unlock()
	}()

	// 6. Contagem de leads por dia
	wg.Add(1)
	go func() {
		defer wg.Done()

		if isSingleDayQuery {
			// Para um único dia, fazer uma única consulta
			dateStr := currentPeriod.From.Format("2006-01-02")

			count, err := uc.eventRepository.CountEventsByDateRange(
				currentPeriod.From,
				currentPeriod.To,
				currentPeriod.TimeFrom,
				currentPeriod.TimeTo,
				"LEAD",
				professionIDs,
				funnelIDs,
				"AND",
			)

			if err == nil {
				resultMutex.Lock()
				result.LeadsByDay[dateStr] = count
				resultMutex.Unlock()
			}
		} else {
			// Para múltiplos dias, gerar datas no intervalo
			dates := generateDateRange(currentPeriod.From, currentPeriod.To)

			// Para cada dia, fazer uma contagem
			for _, dateStr := range dates {
				date, _ := time.Parse("2006-01-02", dateStr)
				nextDate := date.AddDate(0, 0, 1)

				// Contar leads para este dia específico
				count, err := uc.eventRepository.CountEventsByDateRange(
					date,
					nextDate,
					currentPeriod.TimeFrom,
					currentPeriod.TimeTo,
					"LEAD",
					professionIDs,
					funnelIDs,
					"AND",
				)

				if err != nil {
					continue // Ignorar erros e continuar
				}

				resultMutex.Lock()
				result.LeadsByDay[dateStr] = count
				resultMutex.Unlock()
			}
		}
	}()

	// Aguardar conclusão de todas as goroutines
	wg.Wait()

	// Verificar se houve erros
	if len(errors) > 0 {
		return result, errors[0] // Retorna apenas o primeiro erro
	}

	// Calcular métricas
	result.Sessions = calculateMetric(currentSessions, previousSessions)
	result.Leads = calculateMetric(currentLeads, previousLeads)

	// Calcular taxa de conversão
	var currentConversionRate, previousConversionRate int64
	if currentSessions > 0 {
		currentConversionRate = int64((float64(currentLeads) / float64(currentSessions)) * 100)
	}
	if previousSessions > 0 {
		previousConversionRate = int64((float64(previousLeads) / float64(previousSessions)) * 100)
	}
	result.ConversionRate = calculateMetric(currentConversionRate, previousConversionRate)

	// Calcular taxa de conversão por dia
	for dateStr, sessionCount := range result.PeriodCounts {
		leadCount, hasLeads := result.LeadsByDay[dateStr]
		if hasLeads && sessionCount > 0 {
			// Calcular porcentagem de conversão para este dia
			conversionRate := float64(leadCount) / float64(sessionCount) * 100
			// Arredondar para 2 casas decimais
			conversionRate = math.Round(conversionRate*100) / 100
			result.ConversionRateByDay[dateStr] = conversionRate
		} else {
			result.ConversionRateByDay[dateStr] = 0
		}
	}

	// Para consultas de dia único, garantir que os resultados diários reflitam o total
	if isSingleDayQuery {
		singleDay := currentPeriod.From.Format("2006-01-02")

		// Sincronizar contagens para garantir consistência
		if _, ok := result.PeriodCounts[singleDay]; !ok || result.PeriodCounts[singleDay] == 0 {
			result.PeriodCounts[singleDay] = currentSessions
		}

		if _, ok := result.LeadsByDay[singleDay]; !ok {
			result.LeadsByDay[singleDay] = currentLeads
		}

		// Recalcular taxa de conversão para garantir consistência
		if currentSessions > 0 {
			result.ConversionRateByDay[singleDay] = math.Round((float64(currentLeads)/float64(currentSessions)*100)*100) / 100
		}
	}

	// Para consultas de dia único, obter dados por hora
	if isSingleDayQuery {
		fmt.Printf("Obtendo dados por hora para o dia %s\n", currentPeriod.From.Format("2006-01-02"))

		queryStart := time.Now()
		hourlyData, err := uc.GetHourlyData(
			currentPeriod.From,
			userID,
			professionID,
			productID,
			funnelID,
			landingPage,
		)

		resultMutex.Lock()
		queryTimes["hourly_data"] = time.Since(queryStart)
		resultMutex.Unlock()

		if err != nil {
			fmt.Printf("Erro ao obter dados por hora: %v\n", err)
		} else if hourlyData != nil {
			// Incluir dados horários no resultado final
			result.HourlyData = hourlyData
			fmt.Printf("Dados por hora obtidos com sucesso: %d sessões, %d leads\n",
				countMapValues(hourlyData.SessionsByHour),
				countMapValues(hourlyData.LeadsByHour))
		}
	}

	// Registrar estatísticas de performance
	totalTime := time.Since(startTime)
	fmt.Printf("Performance do dashboard - Tempo total: %v\n", totalTime)
	for query, duration := range queryTimes {
		percentOfTotal := float64(duration.Milliseconds()) / float64(totalTime.Milliseconds()) * 100
		fmt.Printf("  - %s: %v (%.1f%%)\n", query, duration, percentOfTotal)
	}

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

// calculateMetric calcula a métrica com comparação percentual
func calculateMetric(current, previous int64) MetricResult {
	var percentage float64
	isIncreasing := current > previous

	if previous > 0 {
		percentage = float64(current-previous) / float64(previous) * 100
	} else if current > 0 {
		percentage = 100 // Se anterior era zero e atual é positivo, aumento de 100%
	}

	// Aplicar valor absoluto à porcentagem
	if percentage < 0 {
		percentage = -percentage
	}

	// Arredondar para duas casas decimais
	percentage = float64(int(percentage*100)) / 100

	return MetricResult{
		Current:      current,
		Previous:     previous,
		Percentage:   percentage,
		IsIncreasing: isIncreasing,
	}
}

// Método auxiliar para verificar se from e to são o mesmo dia
func isSameDay(from, to time.Time) bool {
	return from.Year() == to.Year() && from.Month() == to.Month() && from.Day() == to.Day()
}

// GetHourlyData obtém dados agrupados por hora para um único dia
func (u *dashboardUseCase) GetHourlyData(date time.Time, userID, professionID, productID, funnelID string, landingPage string) (*HourlyMetrics, error) {
	startTime := time.Now()
	dateStr := date.Format("2006-01-02")

	// Gerar uma chave para identificar esta consulta (para uso futuro com cache)
	_ = fmt.Sprintf("hourly:%s:%s:%s:%s:%s:%s",
		dateStr, userID, professionID, productID, funnelID, landingPage)

	// Logging para debug
	log.Printf("Obtendo dados por hora para a data %s com landingPage=%s", dateStr, landingPage)

	result := &HourlyMetrics{
		SessionsByHour:       make(map[string]int64),
		LeadsByHour:          make(map[string]int64),
		ConversionRateByHour: make(map[string]float64),
	}

	// Inicializar mapas com zeros para todas as horas
	for hour := 0; hour < 24; hour++ {
		hourStr := fmt.Sprintf("%02d", hour)
		result.SessionsByHour[hourStr] = 0
		result.LeadsByHour[hourStr] = 0
		result.ConversionRateByHour[hourStr] = 0
	}

	// Consulta direta por hora para sessions - simplificada e corrigida
	sessionQueryStr := `
		SELECT 
			LPAD(EXTRACT(HOUR FROM "sessionStart" AT TIME ZONE 'America/Sao_Paulo')::text, 2, '0') as hour, 
			COUNT(*) as count
		FROM sessions
		WHERE DATE("sessionStart" AT TIME ZONE 'America/Sao_Paulo') = ?`

	sessionArgs := []interface{}{dateStr}

	// Adicionar filtros apenas se não forem vazios
	if landingPage != "" {
		sessionQueryStr += ` AND "landingPage" = ?`
		sessionArgs = append(sessionArgs, landingPage)
	}

	if professionID != "" {
		sessionQueryStr += " AND profession_id = ?"
		sessionArgs = append(sessionArgs, professionID)
	}

	if productID != "" {
		sessionQueryStr += " AND product_id = ?"
		sessionArgs = append(sessionArgs, productID)
	}

	if funnelID != "" {
		sessionQueryStr += " AND funnel_id = ?"
		sessionArgs = append(sessionArgs, funnelID)
	}

	// Agrupar e ordenar
	sessionQueryStr += `
		GROUP BY hour
		ORDER BY hour`

	// Mostrar a consulta SQL para debug
	log.Printf("SQL de sessões por hora: %s", sessionQueryStr)
	log.Printf("Parâmetros: %v", sessionArgs)

	// Executar consulta de sessions
	var sessionsByHour []struct {
		Hour  string `gorm:"column:hour"`
		Count int64  `gorm:"column:count"`
	}

	if err := u.db.Raw(sessionQueryStr, sessionArgs...).Scan(&sessionsByHour).Error; err != nil {
		log.Printf("Erro na consulta SQL para sessões por hora: %v", err)
	} else {
		for _, item := range sessionsByHour {
			result.SessionsByHour[item.Hour] = item.Count
		}
		log.Printf("Dados de sessões por hora obtidos: %d registros", len(sessionsByHour))

		// Log dos primeiros resultados para debug
		if len(sessionsByHour) > 0 {
			log.Printf("Exemplo de dados de sessões: hora %s = %d",
				sessionsByHour[0].Hour, sessionsByHour[0].Count)
		}
	}

	// Consulta direta por hora para leads - simplificada e corrigida
	leadQueryStr := `
		SELECT 
			LPAD(EXTRACT(HOUR FROM "event_time" AT TIME ZONE 'America/Sao_Paulo')::text, 2, '0') as hour, 
			COUNT(*) as count
		FROM events
		WHERE DATE("event_time" AT TIME ZONE 'America/Sao_Paulo') = ?
		AND event_type = 'LEAD'`

	leadArgs := []interface{}{dateStr}

	// Adicionar filtros apenas se não forem vazios
	if landingPage != "" {
		leadQueryStr += ` AND "landingPage" = ?`
		leadArgs = append(leadArgs, landingPage)
	}

	if professionID != "" {
		leadQueryStr += " AND profession_id = ?"
		leadArgs = append(leadArgs, professionID)
	}

	if productID != "" {
		leadQueryStr += " AND product_id = ?"
		leadArgs = append(leadArgs, productID)
	}

	if funnelID != "" {
		leadQueryStr += " AND funnel_id = ?"
		leadArgs = append(leadArgs, funnelID)
	}

	// Agrupar e ordenar
	leadQueryStr += `
		GROUP BY hour
		ORDER BY hour`

	// Mostrar a consulta SQL para debug
	log.Printf("SQL de leads por hora: %s", leadQueryStr)
	log.Printf("Parâmetros: %v", leadArgs)

	// Executar consulta de leads
	var leadsByHour []struct {
		Hour  string `gorm:"column:hour"`
		Count int64  `gorm:"column:count"`
	}

	if err := u.db.Raw(leadQueryStr, leadArgs...).Scan(&leadsByHour).Error; err != nil {
		log.Printf("Erro na consulta SQL para leads por hora: %v", err)
	} else {
		for _, item := range leadsByHour {
			result.LeadsByHour[item.Hour] = item.Count
		}
		log.Printf("Dados de leads por hora obtidos: %d registros", len(leadsByHour))

		// Log dos primeiros resultados para debug
		if len(leadsByHour) > 0 {
			log.Printf("Exemplo de dados de leads: hora %s = %d",
				leadsByHour[0].Hour, leadsByHour[0].Count)
		}
	}

	// Calcular taxa de conversão por hora
	for hour := 0; hour < 24; hour++ {
		hourStr := fmt.Sprintf("%02d", hour)
		sessions := result.SessionsByHour[hourStr]
		leads := result.LeadsByHour[hourStr]

		var rate float64 = 0
		if sessions > 0 {
			// Calcular taxa de conversão como percentagem com 2 casas decimais
			rate = math.Round(float64(leads)/float64(sessions)*10000) / 100
		}
		result.ConversionRateByHour[hourStr] = rate
	}

	// Registrar tempo total para fins de performance
	duration := time.Since(startTime)
	log.Printf("Dados por hora calculados em %v: %d sessões, %d leads no total",
		duration, countMapValues(result.SessionsByHour), countMapValues(result.LeadsByHour))

	// Logar um resumo dos dados para verificação
	sessionTotal := int64(0)
	leadTotal := int64(0)
	for hour := 0; hour < 24; hour++ {
		hourStr := fmt.Sprintf("%02d", hour)
		sessionTotal += result.SessionsByHour[hourStr]
		leadTotal += result.LeadsByHour[hourStr]
	}
	log.Printf("RESUMO - Total de sessões por hora: %d, Total de leads por hora: %d",
		sessionTotal, leadTotal)

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

// Nota: Para otimizar as consultas de dados por hora, é recomendável adicionar os seguintes índices ao banco de dados:
//
// 1. Para a tabela sessions:
// CREATE INDEX idx_sessions_start_time ON sessions USING btree (date_trunc('hour', "sessionStart"::timestamptz AT TIME ZONE 'America/Sao_Paulo'));
// CREATE INDEX idx_sessions_landing_page ON sessions USING btree ("landingPage");
// CREATE INDEX idx_sessions_date_filters ON sessions USING btree ("sessionStart", profession_id, product_id, funnel_id);
//
// 2. Para a tabela events:
// CREATE INDEX idx_events_time ON events USING btree (date_trunc('hour', "event_time"::timestamptz AT TIME ZONE 'America/Sao_Paulo'));
// CREATE INDEX idx_events_type_time ON events USING btree (event_type, "event_time");
// CREATE INDEX idx_events_landing_page ON events USING btree ("landingPage");
//
// Estas otimizações podem melhorar significativamente o desempenho das consultas por hora.
