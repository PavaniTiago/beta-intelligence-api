package repositories

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"gorm.io/gorm"
)

type IUserRepository interface {
	GetUsers(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error)
	FindLeads(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error)
	FindClients(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error)
	FindAnonymous(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error)
	CountLeads(from, to time.Time, timeFrom, timeTo string) (int64, error)
	CountLeadsByPeriods(periods []string) (map[string]int64, error)
	CountClients(from, to time.Time, timeFrom, timeTo string) (int64, error)
	CountClientsByPeriods(periods []string) (map[string]int64, error)
	CountAnonymous(from, to time.Time, timeFrom, timeTo string) (int64, error)
	CountUsers(from, to time.Time, timeFrom, timeTo string) (int64, error)
	GetLeadsDateRange() (time.Time, time.Time, error)
	GetClientsDateRange() (time.Time, time.Time, error)
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// GetUsers retrieves users with optimized query
func (r *UserRepository) GetUsers(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error) {
	var users []entities.User
	var total int64

	offset := (page - 1) * limit

	// Optimize query by using the right indexes
	query := r.db.Model(&entities.User{})

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

		// Use the created_at index
		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	// Get the total count in a separate query to improve performance
	countQuery := query
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	// Always use pagination to avoid retrieving large result sets
	query = query.Offset(offset).Limit(limit)

	// Execute the optimized query
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// FindLeads retorna todos os usuários que são leads com paginação, ordenação e filtro de período
func (r *UserRepository) FindLeads(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error) {
	start := time.Now()
	defer func() {
		log.Printf("FindLeads took: %v", time.Since(start))
	}()

	var leads []entities.User
	var total int64
	offset := (page - 1) * limit

	// Selecionar apenas campos necessários para melhorar performance
	selectFields := []string{
		"user_id", "fullname", "email", "phone",
		"\"isClient\"", "\"isIdentified\"", "created_at",
		"initial_country", "initial_city", "initial_region",
		"initial_marketing_channel", "initial_utm_source",
	}

	// Use the combined index for better performance
	query := r.db.Model(&entities.User{}).Select(selectFields).Where(`"isIdentified" = ? AND "isClient" = ?`, true, false)

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

		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	// Medir tempo da consulta de contagem
	countStart := time.Now()
	// Separate count query for better performance - usar sessão diferente para evitar compartilhamento de estado
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	log.Printf("Count query took: %v", time.Since(countStart))

	if orderBy == "" {
		orderBy = "created_at DESC"
	}
	query = query.Order(orderBy)

	// Always use pagination to improve performance
	query = query.Offset(offset).Limit(limit)

	// Medir tempo da consulta principal
	findStart := time.Now()
	err := query.Find(&leads).Error
	log.Printf("Find query took: %v", time.Since(findStart))

	if err != nil {
		return nil, 0, err
	}

	return leads, total, nil
}

// FindClients retorna todos os usuários que são clientes com paginação, ordenação e filtro de período
func (r *UserRepository) FindClients(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error) {
	var clients []entities.User
	var total int64
	offset := (page - 1) * limit

	// Use the isClient index
	query := r.db.Where(`"isClient" = ?`, true)

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

		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	// Separate count query
	countQuery := query
	if err := countQuery.Model(&entities.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if orderBy == "" {
		orderBy = "user_id asc"
	}
	query = query.Order(orderBy)

	// Use pagination for efficiency
	query = query.Offset(offset).Limit(limit)

	err := query.Find(&clients).Error
	if err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// FindAnonymous retorna todos os usuários anônimos com paginação, ordenação e filtro de período
func (r *UserRepository) FindAnonymous(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error) {
	var anonymous []entities.User
	var total int64
	offset := (page - 1) * limit

	query := r.db.Where(`"isIdentified" = ? AND "isClient" = ?`, false, false)

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

		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	if err := query.Model(&entities.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	// Aplicar paginação apenas se houver filtro de período
	if hasDateFilter {
		query = query.Offset(offset).Limit(limit)
	}

	err := query.Find(&anonymous).Error
	return anonymous, total, err
}

func (r *UserRepository) CountLeads(from, to time.Time, timeFrom, timeTo string) (int64, error) {
	var count int64
	query := r.db.Model(&entities.User{}).Where(`"isIdentified" = ? AND "isClient" = ?`, true, false)

	// Apply date filter
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

		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountLeadsByPeriods counts leads for multiple periods
func (r *UserRepository) CountLeadsByPeriods(periods []string) (map[string]int64, error) {
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

		// Contar leads no período - use os mesmos filtros de CountLeads
		var count int64
		err = r.db.Model(&entities.User{}).
			Where(`"isIdentified" = ? AND "isClient" = ?`, true, false).
			Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
			Count(&count).Error

		if err != nil {
			return nil, err
		}

		result[period] = count
	}

	return result, nil
}

// CountClients retorna apenas a contagem de clientes com filtro de período
func (r *UserRepository) CountClients(from, to time.Time, timeFrom, timeTo string) (int64, error) {
	var count int64
	query := r.db.Model(&entities.User{}).Where(`"isClient" = ?`, true)

	// Apply date filter
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

		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *UserRepository) CountClientsByPeriods(periods []string) (map[string]int64, error) {
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

		// Contar clientes no período - use os mesmos filtros de CountClients
		var count int64
		err = r.db.Model(&entities.User{}).
			Where(`"isClient" = ?`, true).
			Where("created_at BETWEEN ? AND ?", startOfDay, endOfDay).
			Count(&count).Error

		if err != nil {
			return nil, err
		}

		result[period] = count
	}

	return result, nil
}

// CountAnonymous retorna apenas a contagem de usuários anônimos com filtro de período
func (r *UserRepository) CountAnonymous(from, to time.Time, timeFrom, timeTo string) (int64, error) {
	var count int64
	query := r.db.Model(&entities.User{}).Where(`"isIdentified" = ? AND "isClient" = ?`, false, false)

	// Apply date filter
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

		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CountUsers retorna a contagem total de usuários com filtro de período
func (r *UserRepository) CountUsers(from, to time.Time, timeFrom, timeTo string) (int64, error) {
	var count int64
	query := r.db.Model(&entities.User{})

	// Apply date filter
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

		query = query.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetLeadsDateRange returns the min and max dates for leads
func (r *UserRepository) GetLeadsDateRange() (time.Time, time.Time, error) {
	var minDate, maxDate time.Time

	// Get the minimum date
	err := r.db.Model(&entities.User{}).
		Where(`"isIdentified" = ? AND "isClient" = ?`, true, false).
		Order("created_at ASC").
		Limit(1).
		Pluck("created_at", &minDate).Error

	if err != nil {
		return minDate, maxDate, err
	}

	// Get the maximum date
	err = r.db.Model(&entities.User{}).
		Where(`"isIdentified" = ? AND "isClient" = ?`, true, false).
		Order("created_at DESC").
		Limit(1).
		Pluck("created_at", &maxDate).Error

	if err != nil {
		return minDate, maxDate, err
	}

	return minDate, maxDate, nil
}

// GetClientsDateRange returns the min and max dates for clients
func (r *UserRepository) GetClientsDateRange() (time.Time, time.Time, error) {
	var minDate, maxDate time.Time

	// Get the minimum date
	err := r.db.Model(&entities.User{}).
		Where(`"isClient" = ?`, true).
		Order("created_at ASC").
		Limit(1).
		Pluck("created_at", &minDate).Error

	if err != nil {
		return minDate, maxDate, err
	}

	// Get the maximum date
	err = r.db.Model(&entities.User{}).
		Where(`"isClient" = ?`, true).
		Order("created_at DESC").
		Limit(1).
		Pluck("created_at", &maxDate).Error

	if err != nil {
		return minDate, maxDate, err
	}

	return minDate, maxDate, nil
}
