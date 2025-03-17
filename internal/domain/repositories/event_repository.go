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

	// Inicializar a consulta base
	baseQuery := r.db.Model(&entities.Event{})

	// Adicionar JOINs preemptivamente para todas as tabelas relacionadas
	// Isso simplifica a lógica e evita problemas quando os filtros combinam dados de diferentes tabelas
	baseQuery = baseQuery.
		Joins("LEFT JOIN users ON events.user_id = users.user_id").
		Joins("LEFT JOIN sessions ON events.session_id = sessions.session_id").
		Joins("LEFT JOIN professions ON events.profession_id = professions.profession_id").
		Joins("LEFT JOIN products ON events.product_id = products.product_id").
		Joins("LEFT JOIN funnels ON events.funnel_id = funnels.funnel_id")

	// Aplicar filtro de data com timezone explícito
	// Converter para UTC para garantir consistência nas consultas
	baseQuery = baseQuery.Where("events.event_time >= ? AND events.event_time <= ?", from, to)

	fmt.Printf("Aplicando filtro de data: from=%v, to=%v\n", from, to)

	// Gerar SQL para debug após aplicar os JOINs
	testQuery := baseQuery.Session(&gorm.Session{DryRun: true})
	sqlStr := testQuery.Find(&events).Statement.SQL.String()
	fmt.Printf("Generated SQL após JOINs e filtro de data: %s\n", sqlStr)

	// Adicionar filtro de profissão se fornecido
	if len(professionIDs) > 0 {
		baseQuery = baseQuery.Where("events.profession_id IN ?", professionIDs)
		fmt.Printf("Aplicando filtro de profissão com IDs: %v\n", professionIDs)
	}

	// Adicionar filtro de funil se fornecido
	if len(funnelIDs) > 0 {
		baseQuery = baseQuery.Where("events.funnel_id IN ?", funnelIDs)
		fmt.Printf("Aplicando filtro de funil com IDs: %v\n", funnelIDs)
	}

	// Aplicar filtros avançados
	if len(advancedFilters) > 0 {
		fmt.Printf("Aplicando %d filtros avançados com condição: %s\n", len(advancedFilters), filterCondition)

		// Definir operador lógico entre filtros (AND/OR)
		condition := "AND"
		if filterCondition == "OR" {
			condition = "OR"
		}

		// Para filtros OR, precisamos construir uma cláusula completa para não interferir nos outros filtros
		if condition == "OR" {
			orConditions := []string{}
			orValues := []interface{}{}

			for _, filter := range advancedFilters {
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

				if strings.Contains(property, ".") {
					parts := strings.Split(property, ".")
					if len(parts) == 2 {
						tableName := parts[0]
						columnName := parts[1]

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

				// Adicionar à lista de condições OR
				orConditions = append(orConditions, fmt.Sprintf("%s %s ?", property, sqlOperator))
				orValues = append(orValues, sqlValue)
			}

			// Se temos condições OR, adicioná-las à consulta dentro de parênteses
			if len(orConditions) > 0 {
				orClause := "(" + strings.Join(orConditions, " OR ") + ")"
				query := orClause
				baseQuery = baseQuery.Where(query, orValues...)
			}
		} else {
			// Caso AND - podemos aplicar os filtros sequencialmente
			for _, filter := range advancedFilters {
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

				if strings.Contains(property, ".") {
					parts := strings.Split(property, ".")
					if len(parts) == 2 {
						tableName := parts[0]
						columnName := parts[1]

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

				// Adicionar o filtro à consulta
				whereClause := fmt.Sprintf("%s %s ?", property, sqlOperator)
				baseQuery = baseQuery.Where(whereClause, sqlValue)

				fmt.Printf("Adicionado filtro: %s %s %v\n", property, sqlOperator, sqlValue)
			}
		}
	}

	// Gerar SQL final para debug
	testFinalQuery := baseQuery.Session(&gorm.Session{DryRun: true})
	finalSqlStr := testFinalQuery.Find(&events).Statement.SQL.String()
	fmt.Printf("SQL Final após todos os filtros: %s\n", finalSqlStr)

	// Obter contagem total após aplicar todos os filtros
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao contar eventos: %w", err)
	}

	// Calcular offset para paginação
	offset := (page - 1) * limit

	// Executar a consulta com paginação e ordenação
	query := baseQuery.
		Order(orderBy).
		Offset(offset).
		Limit(limit)

	// Executar a consulta para obter eventos
	if err := query.Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao buscar eventos: %w", err)
	}

	// Verificar os eventos retornados
	fmt.Printf("Número de eventos retornados: %d (total no banco: %d)\n", len(events), total)

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

	return events, total, nil
}
