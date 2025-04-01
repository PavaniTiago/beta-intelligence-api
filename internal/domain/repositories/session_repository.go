package repositories

import (
	"context"

	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/google/uuid"

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
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {

	return &SessionRepository{
		db: db,
	}
}

func (r *SessionRepository) GetSessions(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, userID, professionID, productID, funnelID string, isActive *bool) ([]entities.Session, int64, error) {

	var sessions []entities.Session
	var total int64

	offset := (page - 1) * limit

	// Optimize query: Only preload what's necessary based on context
	// Select only needed fields for initial query to improve performance
	query := r.db.Model(&entities.Session{}).Select("sessions.*")

	// Use explicit joins instead of Preload for better query control
	query = query.Joins("LEFT JOIN users ON sessions.user_id = users.user_id")
	query = query.Joins("LEFT JOIN professions ON sessions.profession_id = professions.profession_id")
	query = query.Joins("LEFT JOIN products ON sessions.product_id = products.product_id")
	query = query.Joins("LEFT JOIN funnels ON sessions.funnel_id = funnels.funnel_id")

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

	// Get the total count in a separate query to improve performance
	countQuery := query

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply ordering
	if orderBy != "" {
		query = query.Order(orderBy)
	} else {
		query = query.Order("sessions.\"sessionStart\" DESC")
	}

	// Apply pagination
	query = query.Offset(offset).Limit(limit)

	// Execute the query without using Preload to optimize performance

	if err := query.Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	// If we have no sessions, return early
	if len(sessions) == 0 {
		return sessions, total, nil
	}

	// Collect all the session IDs
	var sessionIDs []uuid.UUID
	for _, session := range sessions {
		sessionIDs = append(sessionIDs, session.ID)
	}

	if len(sessionIDs) > 0 {
		// Now preload related data in a single query for each relation
		var sessionsWithData []entities.Session
		r.db.Where("session_id IN ?", sessionIDs).
			Preload("User").
			Preload("Profession").
			Preload("Product").
			Preload("Funnel").
			Find(&sessionsWithData)

		// Map the results back to our original sessions slice
		sessionMap := make(map[uuid.UUID]entities.Session)
		for _, s := range sessionsWithData {
			sessionMap[s.ID] = s

		}

		for i := range sessions {
			if s, ok := sessionMap[sessions[i].ID]; ok {
				sessions[i] = s
			}
		}
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

		query = query.Where("\"sessionStart\" BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

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
