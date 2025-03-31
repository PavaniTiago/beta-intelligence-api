package repositories

import (
	"context"
	"fmt"
	"log"
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
	// Criar cache com expiração de 5 minutos e limpeza a cada 10 minutos
	c := cache.New(5*time.Minute, 10*time.Minute)
	return &SessionRepository{
		db:    db,
		cache: c,
	}
}

func (r *SessionRepository) GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool) ([]entities.Session, int64, error) {
	start := time.Now()
	defer func() {
		log.Printf("GetSessions took: %v", time.Since(start))
	}()

	var sessions []entities.Session
	var total int64

	offset := (page - 1) * limit

	// Selecionar apenas campos necessários da tabela de sessões
	selectFields := []string{
		"sessions.session_id",
		"sessions.user_id",
		"sessions.\"sessionStart\"",
		"sessions.\"isActive\"",
		"sessions.\"lastActivity\"",
		"sessions.\"duration\"",
		"sessions.country",
		"sessions.\"marketingChannel\"",
		"sessions.profession_id",
		"sessions.product_id",
		"sessions.funnel_id",
	}

	// Otimizar query: usar join apenas para contagem e depois fazer preload seletivo
	query := r.db.WithContext(ctx).Model(&entities.Session{}).Select(selectFields)

	// Adicionar filtros
	if userID != "" {
		query = query.Where("sessions.user_id = ?", userID)
	}

	if professionID != "" {
		profID, err := strconv.Atoi(professionID)
		if err == nil {
			query = query.Where("sessions.profession_id = ?", profID)
		}
	}

	if productID != "" {
		prodID, err := strconv.Atoi(productID)
		if err == nil {
			query = query.Where("sessions.product_id = ?", prodID)
		}
	}

	if funnelID != "" {
		funID, err := strconv.Atoi(funnelID)
		if err == nil {
			query = query.Where("sessions.funnel_id = ?", funID)
		}
	}

	if isActive != nil {
		query = query.Where("sessions.\"isActive\" = ?", *isActive)
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

		query = query.Where("sessions.\"sessionStart\" BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	// Medir tempo da contagem
	countStart := time.Now()
	// Usar sessão separada para evitar compartilhamento de estado
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	log.Printf("Sessions count query took: %v", time.Since(countStart))

	// Apply ordering
	if orderBy != "" {
		query = query.Order(orderBy)
	} else {
		query = query.Order("sessions.\"sessionStart\" DESC")
	}

	// Apply pagination
	query = query.Offset(offset).Limit(limit)

	// Execute the query without using Preload to optimize performance
	findStart := time.Now()
	if err := query.Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	log.Printf("Sessions find query took: %v", time.Since(findStart))

	// If we have no sessions, return early
	if len(sessions) == 0 {
		return sessions, total, nil
	}

	// Collect all the session IDs
	var sessionIDs []uuid.UUID
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
	}

	// Otimizar preloads: carregar relações apenas se houver sessões
	if len(sessionIDs) > 0 {
		// Agora preload seletivo de relacionamentos apenas com campos necessários
		preloadStart := time.Now()

		// Preload com seleção específica de campos
		if err := r.db.Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id, fullname, email")
		}).Preload("Profession", func(db *gorm.DB) *gorm.DB {
			return db.Select("profession_id, name")
		}).Preload("Product", func(db *gorm.DB) *gorm.DB {
			return db.Select("product_id, name")
		}).Preload("Funnel", func(db *gorm.DB) *gorm.DB {
			return db.Select("funnel_id, name")
		}).Where("session_id IN ?", sessionIDs).Find(&sessions).Error; err != nil {
			return nil, 0, err
		}

		log.Printf("Sessions preload took: %v", time.Since(preloadStart))
	}

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
	start := time.Now()
	defer func() {
		log.Printf("CountSessions took: %v", time.Since(start))
	}()

	// Gerar chave de cache baseada nos parâmetros
	cacheKey := fmt.Sprintf("count_sessions:%s:%s:%s:%s:%s:%s:%s:%v",
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		timeFrom, timeTo,
		userID, professionID, productID, funnelID)

	if isActive != nil {
		cacheKey = fmt.Sprintf("%s:active_%v", cacheKey, *isActive)
	}

	// Verificar se o resultado está em cache
	if cachedValue, found := r.cache.Get(cacheKey); found {
		log.Printf("Cache hit for CountSessions: %s", cacheKey)
		return cachedValue.(int64), nil
	}

	// Se não estiver em cache, executar a consulta normal
	log.Printf("Cache miss for CountSessions: %s", cacheKey)
	query := r.db.Model(&entities.Session{})

	// Adicionar filtros
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

		query = query.Where(`"sessionStart" BETWEEN ? AND ?`, fromTime.UTC(), toTime.UTC())
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	// Armazenar resultado em cache
	r.cache.Set(cacheKey, count, cache.DefaultExpiration)
	return count, nil
}

func (r *SessionRepository) CountSessionsByPeriods(periods []string) (map[string]int64, error) {
	result := make(map[string]int64)

	for _, period := range periods {
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
	}

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

	// Get the minimum date
	err := r.db.Model(&entities.Session{}).
		Order("\"sessionStart\" ASC").
		Limit(1).
		Pluck("\"sessionStart\"", &minDate).Error

	if err != nil {
		return minDate, maxDate, err
	}

	// Get the maximum date
	err = r.db.Model(&entities.Session{}).
		Order("\"sessionStart\" DESC").
		Limit(1).
		Pluck("\"sessionStart\"", &maxDate).Error

	if err != nil {
		return minDate, maxDate, err
	}

	return minDate, maxDate, nil
}
