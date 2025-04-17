package repositories

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

type ISessionRepository interface {
	GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) ([]entities.Session, int64, error)
	FindSessionByID(ctx context.Context, id string) (*entities.Session, error)
	CountSessions(from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) (int64, error)
	CountSessionsByPeriods(periods []string, landingPage string, funnelID string, professionID string) (map[string]int64, error)
	FindActiveSessions(page, limit int, orderBy string, landingPage string, funnelID string, professionID string) ([]entities.Session, int64, error)
	GetSessionsDateRange() (time.Time, time.Time, error)
	CountActiveSessions(professionID string, funnelID string, landingPage string) (int64, error)
}

type SessionRepository struct {
	db    *gorm.DB
	cache *cache.Cache
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{
		db:    db,
		cache: cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (r *SessionRepository) GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) ([]entities.Session, int64, error) {
	// Gerar chave de cache baseada nos parâmetros
	cacheKey := fmt.Sprintf("sessions:%d:%d:%s:%v:%v:%s:%s:%s:%s:%s:%s:%v:%s",
		page, limit, orderBy, from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive, landingPage)

	fmt.Printf("GetSessions chamado com from=%v, to=%v\n", from, to)

	// Tentar obter do cache
	if cached, found := r.cache.Get(cacheKey); found {
		fmt.Println("Retornando dados do cache para sessões")
		return cached.([]entities.Session), 0, nil
	}

	// Adicionar timeout ao contexto
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var sessions []entities.Session
	var total int64

	offset := (page - 1) * limit

	// Otimizar query selecionando apenas campos necessários
	query := r.db.WithContext(ctx).Model(&entities.Session{}).Select(
		"\"session_id\", \"user_id\", \"sessionStart\", \"isActive\", \"lastActivity\", \"country\", \"city\", \"state\", \"ipAddress\", \"userAgent\", \"duration\", \"landingPage\"",
	)

	// Aplicar filtros
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if professionID != "" {
		profID, err := strconv.Atoi(professionID)
		if err == nil {
			query = query.Where("profession_id = ?", profID)
		}
	}

	if productID != "" {
		prodID, err := strconv.Atoi(productID)
		if err == nil {
			query = query.Where("product_id = ?", prodID)
		}
	}

	if funnelID != "" {
		funID, err := strconv.Atoi(funnelID)
		if err == nil {
			query = query.Where("funnel_id = ?", funID)
		}
	}

	if landingPage != "" {
		query = query.Where("\"landingPage\" = ?", landingPage)
	}

	if isActive != nil {
		query = query.Where("\"isActive\" = ?", *isActive)
	}

	// Verificar se há filtro de período
	hasDateFilter := !from.IsZero() && !to.IsZero()
	fmt.Printf("hasDateFilter=%v, from=%v, to=%v\n", hasDateFilter, from, to)

	// Aplicar filtro de data com timezone explícito (se houver)
	if !from.IsZero() && !to.IsZero() {
		fromTime := from
		toTime := to

		// Ajustar hora e minuto se timeFrom fornecido
		if timeFrom != "" {
			timeParts := strings.Split(timeFrom, ":")
			if len(timeParts) >= 2 {
				hour, _ := strconv.Atoi(timeParts[0])
				min, _ := strconv.Atoi(timeParts[1])
				fromTime = time.Date(from.Year(), from.Month(), from.Day(), hour, min, 0, 0, from.Location())
				fmt.Printf("Sessions: Ajustando horário de início para: %s\n", fromTime.Format("2006-01-02 15:04:05"))
			}
		} else {
			// Se não fornecido, usar o início do dia
			fromTime = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		}

		// Ajustar hora e minuto se timeTo fornecido
		if timeTo != "" {
			timeParts := strings.Split(timeTo, ":")
			if len(timeParts) >= 2 {
				hour, _ := strconv.Atoi(timeParts[0])
				min, _ := strconv.Atoi(timeParts[1])
				toTime = time.Date(to.Year(), to.Month(), to.Day(), hour, min, 59, 999999999, to.Location())
				fmt.Printf("Sessions: Ajustando horário de fim para: %s\n", toTime.Format("2006-01-02 15:04:05"))
			}
		} else {
			// Se não fornecido, usar o fim do dia
			toTime = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())
		}

		// Formatar as datas como strings no formato de timestamp SQL
		fromStr := fromTime.Format("2006-01-02 15:04:05")
		toStr := toTime.Format("2006-01-02 15:04:05")

		// Aplicar filtro usando a sintaxe com AT TIME ZONE e TIMESTAMP
		query = query.Where("(\"sessionStart\" AT TIME ZONE 'America/Sao_Paulo') BETWEEN ? AND ?",
			fromStr, toStr)

		fmt.Printf("Sessions: Filtro de data aplicado com timestamptz: %s até %s\n", fromStr, toStr)
	}

	// Get SQL for debug
	stmt := query.Statement
	sql := stmt.SQL.String()
	vars := stmt.Vars
	fmt.Printf("SQL para GetSessions: %s, Vars: %v\n", sql, vars)

	// Get total count in a separate query
	countQuery := query
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply ordering
	if orderBy != "" {
		query = query.Order(orderBy)
	} else {
		query = query.Order("\"sessionStart\" DESC")
	}

	// Apply pagination
	query = query.Offset(offset).Limit(limit)

	// Execute query
	if err := query.Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	// Se tivermos sessões, carregar dados relacionados de forma otimizada
	if len(sessions) > 0 {
		var sessionIDs []uuid.UUID
		for _, session := range sessions {
			sessionIDs = append(sessionIDs, session.ID)
		}

		// Carregar dados relacionados em queries separadas
		var users []entities.User
		var professions []entities.Profession
		var products []entities.Product
		var funnels []entities.Funnel

		if err := r.db.Where("user_id IN ?", sessionIDs).Find(&users).Error; err != nil {
			return nil, 0, err
		}

		if err := r.db.Where("profession_id IN ?", sessionIDs).Find(&professions).Error; err != nil {
			return nil, 0, err
		}

		if err := r.db.Where("product_id IN ?", sessionIDs).Find(&products).Error; err != nil {
			return nil, 0, err
		}

		if err := r.db.Where("funnel_id IN ?", sessionIDs).Find(&funnels).Error; err != nil {
			return nil, 0, err
		}

		// Criar maps para acesso rápido
		userMap := make(map[string]entities.User)
		professionMap := make(map[int]entities.Profession)
		productMap := make(map[int]entities.Product)
		funnelMap := make(map[int]entities.Funnel)

		for _, user := range users {
			userMap[user.UserID] = user
		}
		for _, profession := range professions {
			professionMap[profession.ProfessionID] = profession
		}
		for _, product := range products {
			productMap[product.ProductID] = product
		}
		for _, funnel := range funnels {
			funnelMap[funnel.FunnelID] = funnel
		}

		// Associar dados relacionados
		for i := range sessions {
			if user, ok := userMap[sessions[i].UserID.String()]; ok {
				sessions[i].User = &user

				// Garantir que os dados UTM sejam preenchidos a partir do usuário
				if sessions[i].UtmSource == "" && user.InitialUtmSource != "" {
					sessions[i].UtmSource = user.InitialUtmSource
				}
				if sessions[i].UtmMedium == "" && user.InitialUtmMedium != "" {
					sessions[i].UtmMedium = user.InitialUtmMedium
				}
				if sessions[i].UtmCampaign == "" && user.InitialUtmCampaign != "" {
					sessions[i].UtmCampaign = user.InitialUtmCampaign
				}
				if sessions[i].UtmContent == "" && user.InitialUtmContent != "" {
					sessions[i].UtmContent = user.InitialUtmContent
				}
				if sessions[i].UtmTerm == "" && user.InitialUtmTerm != "" {
					sessions[i].UtmTerm = user.InitialUtmTerm
				}
			}
			if profession, ok := professionMap[*sessions[i].ProfessionID]; ok {
				sessions[i].Profession = &profession
			}
			if product, ok := productMap[*sessions[i].ProductID]; ok {
				sessions[i].Product = &product
			}
			if funnel, ok := funnelMap[*sessions[i].FunnelID]; ok {
				sessions[i].Funnel = &funnel
			}
		}
	}

	// Store in cache
	r.cache.Set(cacheKey, sessions, cache.DefaultExpiration)

	return sessions, total, nil
}

func (r *SessionRepository) FindSessionByID(ctx context.Context, id string) (*entities.Session, error) {
	var session entities.Session

	err := r.db.
		Preload("User").
		Preload("Profession").
		Preload("Product").
		Preload("Funnel").
		Where("session_id = ?", id).
		First(&session).Error

	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (r *SessionRepository) CountSessions(from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool, landingPage string) (int64, error) {
	// Gerar chave de cache baseada nos parâmetros
	cacheKey := fmt.Sprintf("count_sessions:%v:%v:%s:%s:%s:%s:%s:%s:%v:%s",
		from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive, landingPage)

	fmt.Printf("CountSessions chamado com from=%v, to=%v, landingPage=%s\n", from, to, landingPage)

	// Tentar obter do cache
	if cached, found := r.cache.Get(cacheKey); found {
		return cached.(int64), nil
	}

	// Query otimizada para contagem
	query := r.db.Model(&entities.Session{}).Select("COUNT(*)")

	// Aplicar filtros
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if professionID != "" {
		profID, err := strconv.Atoi(professionID)
		if err == nil {
			query = query.Where("profession_id = ?", profID)
		}
	}

	if productID != "" {
		prodID, err := strconv.Atoi(productID)
		if err == nil {
			query = query.Where("product_id = ?", prodID)
		}
	}

	if funnelID != "" {
		funID, err := strconv.Atoi(funnelID)
		if err == nil {
			query = query.Where("funnel_id = ?", funID)
		}
	}

	if landingPage != "" {
		// Usar nome exato da coluna com aspas e operador = para match exato
		query = query.Where("\"landingPage\" = ?", landingPage)
	}

	if isActive != nil {
		query = query.Where(`"isActive" = ?`, *isActive)
	}

	// Verificar se há filtro de período
	if !from.IsZero() && !to.IsZero() {
		fromTime := from
		toTime := to

		if timeFrom != "" {
			timeParts := strings.Split(timeFrom, ":")
			if len(timeParts) >= 2 {
				hour, _ := strconv.Atoi(timeParts[0])
				min, _ := strconv.Atoi(timeParts[1])
				fromTime = time.Date(from.Year(), from.Month(), from.Day(), hour, min, 0, 0, from.Location())
			}
		}

		if timeTo != "" {
			timeParts := strings.Split(timeTo, ":")
			if len(timeParts) >= 2 {
				hour, _ := strconv.Atoi(timeParts[0])
				min, _ := strconv.Atoi(timeParts[1])
				toTime = time.Date(to.Year(), to.Month(), to.Day(), hour, min, 59, 999999999, to.Location())
			}
		}

		// Formatar as datas como strings no formato de timestamp SQL
		fromStr := fromTime.Format("2006-01-02 15:04:05")
		toStr := toTime.Format("2006-01-02 15:04:05")

		// Aplicar filtro usando apenas timezone 'America/Sao_Paulo'
		query = query.Where("(\"sessionStart\" AT TIME ZONE 'America/Sao_Paulo') BETWEEN ? AND ?",
			fromStr, toStr)

		fmt.Printf("Sessions Count: Filtro de data aplicado com timezone: %s até %s\n", fromStr, toStr)
	}

	// Get SQL for debug
	stmt := query.Statement
	sql := stmt.SQL.String()
	vars := stmt.Vars
	fmt.Printf("SQL para CountSessions: %s, Vars: %v\n", sql, vars)

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	// Armazenar no cache por 5 minutos
	r.cache.Set(cacheKey, count, 5*time.Minute)

	return count, nil
}

func (r *SessionRepository) CountSessionsByPeriods(periods []string, landingPage string, funnelID string, professionID string) (map[string]int64, error) {
	// Gerar chave de cache baseada nos períodos
	cacheKey := fmt.Sprintf("count_sessions_periods:%v:%s:%s:%s", periods, landingPage, funnelID, professionID)

	// Tentar obter do cache
	if cached, found := r.cache.Get(cacheKey); found {
		return cached.(map[string]int64), nil
	}

	result := make(map[string]int64)

	for _, period := range periods {
		// Gerar chave de cache para o período específico
		periodCacheKey := fmt.Sprintf("count_sessions_period:%s:%s:%s:%s", period, landingPage, funnelID, professionID)

		// Tentar obter do cache do período
		if cached, found := r.cache.Get(periodCacheKey); found {
			result[period] = cached.(int64)
			continue
		}

		// Parse do período
		date, err := time.Parse("2006-01-02", period)
		if err != nil {
			continue
		}

		// Definir início e fim do dia
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, date.Location())

		// Formatar as datas como strings no formato de timestamp SQL
		startStr := startOfDay.Format("2006-01-02 15:04:05")
		endStr := endOfDay.Format("2006-01-02 15:04:05")

		// Iniciar a consulta
		query := r.db.Model(&entities.Session{})

		// Adicionar filtro de landing page se fornecido
		if landingPage != "" {
			query = query.Where("\"landingPage\" = ?", landingPage)
		}

		// Adicionar filtro de funnel_id se fornecido
		if funnelID != "" {
			funID, err := strconv.Atoi(funnelID)
			if err == nil && funID > 0 {
				query = query.Where("funnel_id = ?", funID)
			}
		}

		// Adicionar filtro de profession_id se fornecido
		if professionID != "" {
			profID, err := strconv.Atoi(professionID)
			if err == nil && profID > 0 {
				query = query.Where("profession_id = ?", profID)
			}
		}

		// Contar sessões no período usando timezone
		var count int64
		err = query.Where("(\"sessionStart\" AT TIME ZONE 'America/Sao_Paulo') BETWEEN ? AND ?",
			startStr, endStr).
			Count(&count).Error

		if err != nil {
			return nil, err
		}

		result[period] = count

		// Armazenar no cache por 5 minutos
		r.cache.Set(periodCacheKey, count, 5*time.Minute)
	}

	// Armazenar resultado completo no cache por 5 minutos
	r.cache.Set(cacheKey, result, 5*time.Minute)

	return result, nil
}

func (r *SessionRepository) FindActiveSessions(page, limit int, orderBy string, landingPage string, funnelID string, professionID string) ([]entities.Session, int64, error) {
	var sessions []entities.Session
	var total int64

	// Calculate offset
	offset := (page - 1) * limit

	// Base query com joins
	query := r.db.Model(&entities.Session{}).
		Select(`
			sessions.*,
			users.first_name as user_first_name,
			users.last_name as user_last_name,
			users.initial_utm_source as user_utm_source,
			users.initial_utm_medium as user_utm_medium,
			users.initial_utm_campaign as user_utm_campaign,
			professions.profession_id as profession_id,
			professions.profession_name as profession_name,
			products.product_id as product_id,
			products.product_name as product_name,
			funnels.funnel_id as funnel_id,
			funnels.funnel_name as funnel_name
		`).
		Joins("LEFT JOIN users ON sessions.user_id = users.user_id").
		Joins("LEFT JOIN professions ON sessions.profession_id = professions.profession_id").
		Joins("LEFT JOIN products ON sessions.product_id = products.product_id").
		Joins("LEFT JOIN funnels ON sessions.funnel_id = funnels.funnel_id").
		Where(`sessions."isActive" = ?`, true)

	// Adicionar filtro de landing page se fornecido
	if landingPage != "" {
		query = query.Where(`sessions."landingPage" = ?`, landingPage)
	}

	// Adicionar filtro de funnel_id se fornecido
	if funnelID != "" {
		funnelIDInt, err := strconv.Atoi(funnelID)
		if err == nil {
			query = query.Where("sessions.funnel_id = ?", funnelIDInt)
		}
	}

	// Adicionar filtro de profession_id se fornecido
	if professionID != "" {
		profIDInt, err := strconv.Atoi(professionID)
		if err == nil {
			query = query.Where("sessions.profession_id = ?", profIDInt)
		}
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply ordering
	if orderBy != "" {
		// Adicionar prefixo de tabela para evitar ambiguidade
		if !strings.Contains(orderBy, ".") {
			// Para colunas comuns em sessions
			if strings.Contains(orderBy, "sessionStart") ||
				strings.Contains(orderBy, "lastActivity") ||
				strings.Contains(orderBy, "isActive") ||
				strings.Contains(orderBy, "landingPage") {
				orderBy = "sessions." + orderBy
			}
		}
		query = query.Order(orderBy)
	} else {
		query = query.Order(`sessions."sessionStart" DESC`)
	}

	// Get paginated results com preloads para garantir o carregamento completo das entidades relacionadas
	err := query.Offset(offset).
		Limit(limit).
		Preload("User").
		Preload("Profession").
		Preload("Product").
		Preload("Funnel").
		Find(&sessions).Error

	if err != nil {
		return nil, 0, err
	}

	// Garantir que as relações estejam corretamente preenchidas
	for i := range sessions {
		// Assegurar que dados UTM estejam preenchidos a partir do usuário
		if sessions[i].User != nil {
			if sessions[i].UtmSource == "" && sessions[i].User.InitialUtmSource != "" {
				sessions[i].UtmSource = sessions[i].User.InitialUtmSource
			}
			if sessions[i].UtmMedium == "" && sessions[i].User.InitialUtmMedium != "" {
				sessions[i].UtmMedium = sessions[i].User.InitialUtmMedium
			}
			if sessions[i].UtmCampaign == "" && sessions[i].User.InitialUtmCampaign != "" {
				sessions[i].UtmCampaign = sessions[i].User.InitialUtmCampaign
			}
		}
	}

	return sessions, total, nil
}

func (r *SessionRepository) GetSessionsDateRange() (time.Time, time.Time, error) {
	var minDate, maxDate time.Time

	fmt.Println("GetSessionsDateRange chamado")

	// Verificar se existem sessões com datas não nulas
	var count int64
	if err := r.db.Model(&entities.Session{}).Where("\"sessionStart\" IS NOT NULL").Count(&count).Error; err != nil {
		return minDate, maxDate, err
	}

	if count == 0 {
		return minDate, maxDate, nil
	}

	// Get the minimum date usando First em vez de Pluck
	var minSession entities.Session
	minQuery := r.db.Model(&entities.Session{}).
		Where("\"sessionStart\" IS NOT NULL").
		Order("\"sessionStart\" ASC").
		Limit(1)

	// Get SQL for debug
	stmt := minQuery.Statement
	sql := stmt.SQL.String()
	vars := stmt.Vars
	fmt.Printf("SQL para encontrar data mínima: %s, Vars: %v\n", sql, vars)

	err := minQuery.First(&minSession).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Retornar data zero sem erro se não encontrar
			return minDate, maxDate, nil
		}
		return minDate, maxDate, err
	}

	// Usar a data encontrada
	minDate = minSession.SessionStart

	// Get the maximum date usando First em vez de Pluck
	var maxSession entities.Session
	err = r.db.Model(&entities.Session{}).
		Where("\"sessionStart\" IS NOT NULL").
		Order("\"sessionStart\" DESC").
		Limit(1).
		First(&maxSession).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Retornar data zero sem erro se não encontrar
			return minDate, maxDate, nil
		}
		return minDate, maxDate, err
	}

	// Usar a data encontrada
	maxDate = maxSession.SessionStart

	fmt.Printf("Intervalo de datas para sessões: de %v até %v\n", minDate, maxDate)
	return minDate, maxDate, nil
}

func (r *SessionRepository) CountActiveSessions(professionID string, funnelID string, landingPage string) (int64, error) {
	var count int64

	// Query base simplificada
	query := r.db.Model(&entities.Session{}).Where(`"isActive" = ?`, true)

	// Filtros adicionais
	if landingPage != "" {
		query = query.Where(`"landingPage" = ?`, landingPage)
	}

	// Filtro de profession_id
	if professionID != "" {
		profIDInt, err := strconv.Atoi(professionID)
		if err == nil {
			query = query.Where("profession_id = ?", profIDInt)
		}
	}

	// Filtro de funnel_id
	if funnelID != "" {
		funnelIDInt, err := strconv.Atoi(funnelID)
		if err == nil {
			query = query.Where("funnel_id = ?", funnelIDInt)
		}
	}

	// Executar a contagem
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}
