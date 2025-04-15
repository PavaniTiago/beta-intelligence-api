package repositories

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

type IUserRepository interface {
	GetUsers(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error)
	FindLeads(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error)
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
	db    *gorm.DB
	cache *cache.Cache
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db:    db,
		cache: cache.New(5*time.Minute, 10*time.Minute),
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
func (r *UserRepository) FindLeads(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string) ([]entities.User, int64, error) {
	// Gerar chave de cache baseada nos parâmetros
	cacheKey := fmt.Sprintf("leads:%d:%d:%s:%v:%v:%s:%s",
		page, limit, orderBy, from, to, timeFrom, timeTo)

	fmt.Printf("FindLeads chamado com from=%v, to=%v\n", from, to)

	// Tentar obter do cache
	if cached, found := r.cache.Get(cacheKey); found {
		fmt.Println("Retornando dados do cache para leads")
		return cached.([]entities.User), 0, nil
	}

	// Adicionar timeout ao contexto
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var users []entities.User
	var total int64

	offset := (page - 1) * limit

	// Query otimizada para contagem
	countQuery := r.db.WithContext(ctx).Model(&entities.User{}).
		Where(`"isIdentified" = ? AND "isClient" = ?`, true, false)

	// Aplicar filtros de data
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

		countQuery = countQuery.Where("created_at BETWEEN ? AND ?", fromTime.UTC(), toTime.UTC())
		fmt.Printf("Aplicando filtro de data no CountQuery: %v até %v\n", fromTime.UTC(), toTime.UTC())
	}

	// Get SQL for debug (countQuery)
	countStmt := countQuery.Statement
	countSQL := countStmt.SQL.String()
	countVars := countStmt.Vars
	fmt.Printf("SQL para contagem de leads: %s, Vars: %v\n", countSQL, countVars)

	// Obter total
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Query principal otimizada
	query := r.db.WithContext(ctx).Model(&entities.User{}).
		Select("user_id, name, email, phone, created_at, \"isIdentified\", \"isClient\"").
		Where(`"isIdentified" = ? AND "isClient" = ?`, true, false)

	// Aplicar filtros de data
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
		fmt.Printf("Aplicando filtro de data no Query principal: %v até %v\n", fromTime.UTC(), toTime.UTC())
	}

	// Get SQL for debug (main query)
	stmt := query.Statement
	sql := stmt.SQL.String()
	vars := stmt.Vars
	fmt.Printf("SQL para busca de leads: %s, Vars: %v\n", sql, vars)

	// Aplicar ordenação
	if orderBy != "" {
		query = query.Order(orderBy)
	} else {
		query = query.Order("created_at DESC")
	}

	// Aplicar paginação
	query = query.Offset(offset).Limit(limit)

	// Executar query
	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	// Armazenar no cache por 5 minutos
	r.cache.Set(cacheKey, users, 5*time.Minute)

	return users, total, nil
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
	fmt.Printf("CountLeads chamado com from=%v, to=%v\n", from, to)

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
		fmt.Printf("Aplicando filtro de data: %v até %v\n", fromTime.UTC(), toTime.UTC())
	}

	// Get SQL for debug
	stmt := query.Statement
	sql := stmt.SQL.String()
	vars := stmt.Vars
	fmt.Printf("SQL para CountLeads: %s, Vars: %v\n", sql, vars)

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

	fmt.Println("GetLeadsDateRange chamado")

	// Get the minimum date
	var minUser entities.User
	minQuery := r.db.Model(&entities.User{}).
		Where(`"isIdentified" = ? AND "isClient" = ? AND created_at IS NOT NULL`, true, false).
		Order("created_at ASC").
		Limit(1)

	// Get SQL for debug
	minStmt := minQuery.Statement
	minSQL := minStmt.SQL.String()
	minVars := minStmt.Vars
	fmt.Printf("SQL para encontrar data mínima de leads: %s, Vars: %v\n", minSQL, minVars)

	err := minQuery.First(&minUser).Error

	if err != nil {
		return minDate, maxDate, err
	}

	minDate = minUser.CreatedAt

	// Get the maximum date
	var maxUser entities.User
	maxQuery := r.db.Model(&entities.User{}).
		Where(`"isIdentified" = ? AND "isClient" = ? AND created_at IS NOT NULL`, true, false).
		Order("created_at DESC").
		Limit(1)

	// Get SQL for debug
	maxStmt := maxQuery.Statement
	maxSQL := maxStmt.SQL.String()
	maxVars := maxStmt.Vars
	fmt.Printf("SQL para encontrar data máxima de leads: %s, Vars: %v\n", maxSQL, maxVars)

	err = maxQuery.First(&maxUser).Error

	if err != nil {
		return minDate, maxDate, err
	}

	maxDate = maxUser.CreatedAt

	fmt.Printf("Intervalo de datas para leads: de %v até %v\n", minDate, maxDate)
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
