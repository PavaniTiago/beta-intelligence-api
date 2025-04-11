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
	GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool) ([]entities.Session, int64, error)
	FindSessionByID(ctx context.Context, id string) (*entities.Session, error)
	CountSessions(from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool) (int64, error)
	CountSessionsByPeriods(periods []string) (map[string]int64, error)
	FindActiveSessions(page, limit int, orderBy string) ([]entities.Session, int64, error)
	GetSessionsDateRange() (time.Time, time.Time, error)
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

func (r *SessionRepository) GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool) ([]entities.Session, int64, error) {
	// Gerar chave de cache baseada nos parâmetros
	cacheKey := fmt.Sprintf("sessions:%d:%d:%s:%v:%v:%s:%s:%s:%s:%s:%s:%v",
		page, limit, orderBy, from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive)

	// Tentar obter do cache
	if cached, found := r.cache.Get(cacheKey); found {
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
		"session_id, user_id, session_start, is_active, last_activity, country, city, state, ip_address, user_agent, duration",
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

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	// Verificar se há filtro de período
	hasDateFilter := !from.IsZero() && !to.IsZero()

	if hasDateFilter {
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

		query = query.Where("session_start BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	// Get total count in a separate query
	countQuery := query
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply ordering
	if orderBy != "" {
		query = query.Order(orderBy)
	} else {
		query = query.Order("session_start DESC")
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

func (r *SessionRepository) CountSessions(from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool) (int64, error) {
	// Gerar chave de cache baseada nos parâmetros
	cacheKey := fmt.Sprintf("count_sessions:%v:%v:%s:%s:%s:%s:%s:%s:%v",
		from, to, timeFrom, timeTo, userID, professionID, productID, funnelID, isActive)

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

		query = query.Where("session_start BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	// Armazenar no cache por 5 minutos
	r.cache.Set(cacheKey, count, 5*time.Minute)

	return count, nil
}

func (r *SessionRepository) CountSessionsByPeriods(periods []string) (map[string]int64, error) {
	// Gerar chave de cache baseada nos períodos
	cacheKey := fmt.Sprintf("count_sessions_periods:%v", periods)

	// Tentar obter do cache
	if cached, found := r.cache.Get(cacheKey); found {
		return cached.(map[string]int64), nil
	}

	result := make(map[string]int64)

	for _, period := range periods {
		// Gerar chave de cache para o período específico
		periodCacheKey := fmt.Sprintf("count_sessions_period:%s", period)

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
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, time.UTC)

		// Contar sessões no período
		var count int64
		err = r.db.Model(&entities.Session{}).
			Where("\"sessionStart\" BETWEEN ? AND ?", startOfDay, endOfDay).
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

func (r *SessionRepository) FindActiveSessions(page, limit int, orderBy string) ([]entities.Session, int64, error) {
	var sessions []entities.Session
	var total int64

	// Calculate offset
	offset := (page - 1) * limit

	// Base query
	query := r.db.Model(&entities.Session{}).Where(`"isActive" = ?`, true)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply ordering
	if orderBy != "" {
		query = query.Order(orderBy)
	} else {
		query = query.Order("\"sessionStart\" DESC")
	}

	// Get paginated results with preloads
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

	return sessions, total, nil
}

func (r *SessionRepository) GetSessionsDateRange() (time.Time, time.Time, error) {
	var minDate, maxDate time.Time

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
	err := r.db.Model(&entities.Session{}).
		Where("\"sessionStart\" IS NOT NULL").
		Order("\"sessionStart\" ASC").
		Limit(1).
		First(&minSession).Error

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

	return minDate, maxDate, nil
}
