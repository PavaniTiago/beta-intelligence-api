package repositories

import (
	"fmt"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdvancedFilter representa um filtro avançado com propriedade, operador e valor
type AdvancedFilter struct {
	ID       string `json:"id"`
	Property string `json:"property"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type EventRepository interface {
	GetEvents(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []AdvancedFilter, filterCondition string) ([]entities.Event, int64, error)
}

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db}
}

func (r *eventRepository) GetEvents(page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []AdvancedFilter, filterCondition string) ([]entities.Event, int64, error) {
	var events []entities.Event
	var total int64

	// Base query com filtro de data
	baseQuery := r.db.Model(&entities.Event{}).
		Where("events.event_time AT TIME ZONE 'UTC' >= ? AND events.event_time AT TIME ZONE 'UTC' <= ?", from.UTC(), to.UTC())

	// Verificar se precisamos adicionar JOINs para os filtros avançados
	needsUserJoin := false
	needsSessionJoin := false
	needsProfessionJoin := false
	needsProductJoin := false
	needsFunnelJoin := false

	// Analisar os filtros avançados para identificar as tabelas necessárias
	for _, filter := range advancedFilters {
		if strings.HasPrefix(filter.Property, "user.") {
			needsUserJoin = true
		} else if strings.HasPrefix(filter.Property, "session.") {
			needsSessionJoin = true
		} else if strings.HasPrefix(filter.Property, "profession.") {
			needsProfessionJoin = true
		} else if strings.HasPrefix(filter.Property, "product.") {
			needsProductJoin = true
		} else if strings.HasPrefix(filter.Property, "funnel.") {
			needsFunnelJoin = true
		}
	}

	// Adicionar os JOINs necessários
	if needsUserJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN users ON events.user_id = users.user_id")
	}
	if needsSessionJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN sessions ON events.session_id = sessions.session_id")
	}
	if needsProfessionJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN professions ON events.profession_id = professions.profession_id")
	}
	if needsProductJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN products ON events.product_id = products.product_id")
	}
	if needsFunnelJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN funnels ON events.funnel_id = funnels.funnel_id")
	}

	// Gerar SQL para debug
	var sqlStr string
	testQuery := baseQuery.Session(&gorm.Session{DryRun: true})
	sqlStr = testQuery.Find(&events).Statement.SQL.String()
	fmt.Printf("Generated SQL with date/time filter: %s\n", sqlStr)

	// Adicionar filtro de profissão se fornecido
	if len(professionIDs) > 0 {
		fmt.Printf("Applying profession filter with IDs: %v\n", professionIDs)

		// Verificar se os IDs existem na tabela professions
		var existingProfessionIDs []int
		if err := r.db.Model(&entities.Profession{}).
			Where("profession_id IN ?", professionIDs).
			Pluck("profession_id", &existingProfessionIDs).Error; err != nil {
			fmt.Printf("Error checking profession IDs: %v\n", err)
		}

		fmt.Printf("Existing profession IDs in database: %v\n", existingProfessionIDs)

		// Construir a condição IN manualmente
		var placeholders []string
		var values []interface{}
		for _, id := range professionIDs {
			placeholders = append(placeholders, "?")
			values = append(values, id)
		}
		inClause := fmt.Sprintf("events.profession_id IN (%s)", strings.Join(placeholders, ","))
		baseQuery = baseQuery.Where(inClause, values...)

		// Gerar SQL para debug
		var sqlStr string
		testQuery := baseQuery.Session(&gorm.Session{DryRun: true})
		sqlStr = testQuery.Find(&events).Statement.SQL.String()
		fmt.Printf("Generated SQL with profession filter: %s\n", sqlStr)
	}

	// Adicionar filtro de funil se fornecido
	if len(funnelIDs) > 0 {
		baseQuery = baseQuery.Where("events.funnel_id IN ?", funnelIDs)
	}

	// Aplicar filtros avançados
	if len(advancedFilters) > 0 {
		// Validar condição de filtro (AND/OR)
		condition := "AND"
		if filterCondition == "OR" {
			condition = "OR"
		}

		for i, filter := range advancedFilters {
			// Mapear os operadores do frontend para operadores SQL
			var sqlOperator string
			var sqlValue interface{}

			switch filter.Operator {
			case "equals":
				sqlOperator = "="
				sqlValue = filter.Value
			case "not_equals":
				sqlOperator = "!="
				sqlValue = filter.Value
			case "contains":
				sqlOperator = "LIKE"
				sqlValue = "%" + filter.Value + "%"
			case "not_contains":
				sqlOperator = "NOT LIKE"
				sqlValue = "%" + filter.Value + "%"
			default:
				// Operador não suportado, pular este filtro
				continue
			}

			// Tratar propriedades aninhadas como user.fullname
			property := filter.Property
			var tableName, columnName string

			if strings.Contains(property, ".") {
				parts := strings.Split(property, ".")
				if len(parts) == 2 {
					tableName = parts[0]
					columnName = parts[1]

					// Mapear o nome da tabela corretamente
					switch tableName {
					case "user":
						property = "users." + columnName
					case "session":
						property = "sessions." + columnName
					case "profession":
						property = "professions." + columnName
					case "product":
						property = "products." + columnName
					case "funnel":
						property = "funnels." + columnName
					default:
						property = "events." + property
					}
				}
			} else {
				// Se não for propriedade aninhada, assumir que é do evento
				property = "events." + property
			}

			// Construir a condição WHERE
			whereClause := fmt.Sprintf("%s %s ?", property, sqlOperator)

			// Adicionar a condição ao query
			if i == 0 {
				// Para o primeiro filtro, não usamos AND/OR ainda
				baseQuery = baseQuery.Where(whereClause, sqlValue)
			} else {
				// Para filtros subsequentes, usar AND ou OR conforme especificado
				if condition == "AND" {
					baseQuery = baseQuery.Where(whereClause, sqlValue)
				} else {
					baseQuery = baseQuery.Or(whereClause, sqlValue)
				}
			}
		}
	}

	// Get total count AFTER applying all filters
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Calculate offset for pagination
	offset := (page - 1) * limit

	// Removendo a seleção explícita de colunas que estão causando o erro
	query := baseQuery.
		Order(orderBy).
		Offset(offset).
		Limit(limit)

	// Execute query to get events
	if err := query.Find(&events).Error; err != nil {
		return nil, 0, err
	}

	// Verificar os eventos retornados
	fmt.Printf("Number of events returned: %d\n", len(events))
	if len(events) > 0 {
		fmt.Printf("First event profession_id: %d\n", events[0].ProfessionID)
	}

	// Collect all the IDs we need to fetch related data
	var sessionIDs []uuid.UUID
	var userIDs []string
	var professionIDsFromEvents []int
	var productIDs []int
	var funnelIDsFromEvents []int

	for _, event := range events {
		sessionIDs = append(sessionIDs, event.SessionID)
		userIDs = append(userIDs, event.UserID)

		if event.ProfessionID > 0 {
			professionIDsFromEvents = append(professionIDsFromEvents, event.ProfessionID)
		}

		if event.ProductID > 0 {
			productIDs = append(productIDs, event.ProductID)
		}

		if event.FunnelID > 0 {
			funnelIDsFromEvents = append(funnelIDsFromEvents, event.FunnelID)
		}
	}

	// Carregue as sessões explicitamente
	var sessions []entities.Session
	if err := r.db.Where("session_id IN ?", sessionIDs).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	// Create a map for faster lookups
	sessionMap := make(map[uuid.UUID]entities.Session)
	for _, s := range sessions {
		sessionMap[s.ID] = s
	}

	// Fetch users
	var users []entities.User
	if err := r.db.Where("user_id IN ?", userIDs).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	userMap := make(map[string]entities.User)
	for _, u := range users {
		userMap[u.UserID] = u
	}

	// Fetch professions if needed
	professionMap := make(map[int]entities.Profession)
	if len(professionIDsFromEvents) > 0 {
		var professions []entities.Profession
		if err := r.db.Where("profession_id IN ?", professionIDsFromEvents).Find(&professions).Error; err != nil {
			return nil, 0, err
		}

		for _, p := range professions {
			professionMap[p.ProfessionID] = p
		}
	}

	// Fetch products if needed
	productMap := make(map[int]entities.Product)
	if len(productIDs) > 0 {
		var products []entities.Product
		if err := r.db.Where("product_id IN ?", productIDs).Find(&products).Error; err != nil {
			return nil, 0, err
		}

		for _, p := range products {
			productMap[p.ProductID] = p
		}
	}

	// Fetch funnels if needed
	funnelMap := make(map[int]entities.Funnel)
	if len(funnelIDsFromEvents) > 0 {
		var funnels []entities.Funnel
		if err := r.db.Where("funnel_id IN ?", funnelIDsFromEvents).Find(&funnels).Error; err != nil {
			return nil, 0, err
		}

		for _, f := range funnels {
			funnelMap[f.FunnelID] = f
		}
	}

	// Assign related data to events
	for i := range events {
		if session, ok := sessionMap[events[i].SessionID]; ok {
			events[i].Session = session
		}

		if user, ok := userMap[events[i].UserID]; ok {
			events[i].User = user
		}

		if profession, ok := professionMap[events[i].ProfessionID]; ok {
			events[i].Profession = profession
		}

		if product, ok := productMap[events[i].ProductID]; ok {
			events[i].Product = product
		}

		if funnel, ok := funnelMap[events[i].FunnelID]; ok {
			events[i].Funnel = funnel
		}
	}

	// Verificar se há eventos no período
	var countInPeriod int64
	if err := r.db.Model(&entities.Event{}).
		Where("event_time >= ? AND event_time <= ?", from, to).
		Count(&countInPeriod).Error; err != nil {
		fmt.Printf("Error counting events in period: %v\n", err)
	} else {
		fmt.Printf("Number of events in period (%v to %v): %d\n", from, to, countInPeriod)
	}

	return events, total, nil
}
