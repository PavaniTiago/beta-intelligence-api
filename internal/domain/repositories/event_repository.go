package repositories

import (
	"context"
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
	GetEvents(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []AdvancedFilter, filterCondition string) ([]entities.Event, int64, error)
}

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db}
}

func (r *eventRepository) GetEvents(ctx context.Context, page, limit int, orderBy string, from, to time.Time, timeFrom, timeTo string, professionIDs, funnelIDs []int, advancedFilters []AdvancedFilter, filterCondition string) ([]entities.Event, int64, error) {
	var events []entities.Event
	var total int64

	// Definir localização de Brasília (UTC-3)
	brazilLocation, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Fallback para UTC-3 se não conseguir carregar a localização
		brazilLocation = time.FixedZone("BRT", -3*60*60)
	}

	// Converter timestamps para horário de Brasília
	if !from.IsZero() {
		from = from.In(brazilLocation)
	}
	if !to.IsZero() {
		to = to.In(brazilLocation)
	}

	// Verificar se estamos buscando apenas eventos PURCHASE
	isPurchaseOnlyQuery := false
	for _, filter := range advancedFilters {
		if (filter.Property == "event_type" || filter.Property == "events.event_type" || filter.Property == "e.event_type") &&
			filter.Operator == "equals" &&
			filter.Value == "PURCHASE" {
			isPurchaseOnlyQuery = true
			fmt.Printf("DEBUG: Detectado filtro apenas para PURCHASE\n")
			break
		}
	}

	// Inicializar a consulta base - sempre usar alias 'e' para eventos
	baseQuery := r.db.Model(&entities.Event{}).Table("events e")

	// Sempre incluir todos os JOINs necessários
	// JOIN com users (alias u) para obter dados UTM (fonte primária e exclusiva de UTMs)
	baseQuery = baseQuery.
		Joins("JOIN users u ON e.user_id = u.user_id").
		Joins("LEFT JOIN sessions s ON e.session_id = s.session_id").
		Joins("LEFT JOIN professions ON e.profession_id = professions.profession_id").
		Joins("LEFT JOIN products ON e.product_id = products.product_id").
		Joins("LEFT JOIN funnels ON e.funnel_id = funnels.funnel_id")

	// Se for uma consulta específica para PURCHASE, aplicar diretamente
	if isPurchaseOnlyQuery {
		fmt.Printf("DEBUG: Aplicando filtro direto para PURCHASE\n")
		baseQuery = baseQuery.Where("e.event_type = ?", "PURCHASE")
	}

	// Aplicar filtro de data com timezone explícito (se houver)
	if !from.IsZero() && !to.IsZero() {
		baseQuery = baseQuery.Where("e.event_time >= ? AND e.event_time <= ?", from, to)
	}

	// Adicionar filtro de profissão se fornecido
	if len(professionIDs) > 0 {
		baseQuery = baseQuery.Where("e.profession_id IN ?", professionIDs)
	}

	// Adicionar filtro de funil se fornecido
	if len(funnelIDs) > 0 {
		baseQuery = baseQuery.Where("e.funnel_id IN ?", funnelIDs)
	}

	// Aplicar filtros avançados
	if len(advancedFilters) > 0 {
		fmt.Printf("DEBUG REPO: Aplicando %d filtros avançados com condição '%s'\n", len(advancedFilters), filterCondition)
		for i, filter := range advancedFilters {
			fmt.Printf("DEBUG REPO: Filtro #%d: property=%s, operator=%s, value=%s\n",
				i+1, filter.Property, filter.Operator, filter.Value)
		}

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
				property := processPropertyName(filter.Property)

				fmt.Printf("DEBUG REPO: Property '%s' mapeada para '%s'\n", filter.Property, property)

				// Verificar casos especiais para PURCHASE
				if (filter.Property == "event_type" || filter.Property == "events.event_type" || filter.Property == "e.event_type") &&
					filter.Operator == "equals" &&
					filter.Value == "PURCHASE" {
					// Este é um filtro para event_type = 'PURCHASE'
					fmt.Printf("DEBUG REPO: Detectado filtro para event_type = 'PURCHASE'\n")
				}

				// Corrigir aspas para colunas case-sensitive
				if strings.Contains(property, ".") && !strings.Contains(property, "\"") {
					parts := strings.Split(property, ".")
					if len(parts) == 2 && needsQuotesForColumn(parts[0], parts[1]) {
						property = fmt.Sprintf("%s.\"%s\"", parts[0], parts[1])
						fmt.Printf("DEBUG REPO: Adicionando aspas ao campo: %s\n", property)
					}
				}

				switch filter.Operator {
				case "equals":
					sqlOperator = "="
					sqlValue = filter.Value
				case "not_equals":
					// Tratamento especial para comparação com string vazia em campos UTM
					if filter.Value == "" && isUtmField(filter.Property) {
						// Determinar qual tabela usar baseado na propriedade
						var whereClause string
						if strings.HasPrefix(filter.Property, "user.") {
							// Para campos UTM do usuário
							switch {
							case strings.Contains(filter.Property, "utm_source"):
								whereClause = "COALESCE(u.\"initialUtmSource\", '') != ?"
							case strings.Contains(filter.Property, "utm_medium"):
								whereClause = "COALESCE(u.\"initialUtmMedium\", '') != ?"
							case strings.Contains(filter.Property, "utm_campaign"):
								whereClause = "COALESCE(u.\"initialUtmCampaign\", '') != ?"
							case strings.Contains(filter.Property, "utm_content"):
								whereClause = "COALESCE(u.\"initialUtmContent\", '') != ?"
							case strings.Contains(filter.Property, "utm_term"):
								whereClause = "COALESCE(u.\"initialUtmTerm\", '') != ?"
							default:
								whereClause = fmt.Sprintf("COALESCE(%s, '') != ?", property)
							}
						} else if strings.HasPrefix(filter.Property, "session.") {
							// Para campos UTM da sessão
							switch {
							case strings.Contains(filter.Property, "utm_source"):
								whereClause = "COALESCE(s.\"utmSource\", '') != ?"
							case strings.Contains(filter.Property, "utm_medium"):
								whereClause = "COALESCE(s.\"utmMedium\", '') != ?"
							case strings.Contains(filter.Property, "utm_campaign"):
								whereClause = "COALESCE(s.\"utmCampaign\", '') != ?"
							case strings.Contains(filter.Property, "utm_content"):
								whereClause = "COALESCE(s.\"utmContent\", '') != ?"
							case strings.Contains(filter.Property, "utm_term"):
								whereClause = "COALESCE(s.\"utmTerm\", '') != ?"
							default:
								whereClause = fmt.Sprintf("COALESCE(%s, '') != ?", property)
							}
						} else {
							// Para UTMs sem prefixo explícito, usamos tanto do usuário quanto da sessão
							whereClause = "(COALESCE(u.\"initialUtmSource\", '') != ? OR COALESCE(s.\"utmSource\", '') != ?)"
							baseQuery = baseQuery.Where(whereClause, "", "")
							continue
						}

						fmt.Printf("DEBUG REPO: Condição especial para UTM vazio: %s\n", whereClause)
						baseQuery = baseQuery.Where(whereClause, "")
						continue
					}
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
					fmt.Printf("DEBUG REPO: Operador não suportado: %s\n", filter.Operator)
					continue
				}

				// Adicionar à lista de condições OR
				clauseStr := fmt.Sprintf("%s %s ?", property, sqlOperator)

				// Tratamento especial para colunas case-sensitive
				if strings.Contains(clauseStr, "initialUtm") && !strings.Contains(clauseStr, "\"") {
					// Assegurar que colunas initialUtm* tenham aspas duplas
					clauseStr = strings.Replace(clauseStr, "u.initialUtmSource", "u.\"initialUtmSource\"", -1)
					clauseStr = strings.Replace(clauseStr, "u.initialUtmMedium", "u.\"initialUtmMedium\"", -1)
					clauseStr = strings.Replace(clauseStr, "u.initialUtmCampaign", "u.\"initialUtmCampaign\"", -1)
					clauseStr = strings.Replace(clauseStr, "u.initialUtmContent", "u.\"initialUtmContent\"", -1)
					clauseStr = strings.Replace(clauseStr, "u.initialUtmTerm", "u.\"initialUtmTerm\"", -1)
				}

				// Tratamento especial para colunas UTM da sessão
				if strings.Contains(clauseStr, "utmSource") && !strings.Contains(clauseStr, "\"") {
					clauseStr = strings.Replace(clauseStr, "s.utmSource", "s.\"utmSource\"", -1)
					clauseStr = strings.Replace(clauseStr, "s.utmMedium", "s.\"utmMedium\"", -1)
					clauseStr = strings.Replace(clauseStr, "s.utmCampaign", "s.\"utmCampaign\"", -1)
					clauseStr = strings.Replace(clauseStr, "s.utmContent", "s.\"utmContent\"", -1)
					clauseStr = strings.Replace(clauseStr, "s.utmTerm", "s.\"utmTerm\"", -1)
				}

				orConditions = append(orConditions, clauseStr)
				orValues = append(orValues, sqlValue)
				fmt.Printf("DEBUG REPO: Adicionada condição OR: %s com valor %v\n", clauseStr, sqlValue)
			}

			// Se temos condições OR, adicioná-las à consulta dentro de parênteses
			if len(orConditions) > 0 {
				orClause := "(" + strings.Join(orConditions, " OR ") + ")"
				fmt.Printf("DEBUG REPO: Aplicando cláusula OR completa: %s\n", orClause)
				baseQuery = baseQuery.Where(orClause, orValues...)
			}
		} else {
			// Caso AND - podemos aplicar os filtros sequencialmente
			for _, filter := range advancedFilters {
				// Mapear os operadores do frontend para operadores SQL
				var sqlOperator string
				var sqlValue interface{}
				property := processPropertyName(filter.Property)

				fmt.Printf("DEBUG REPO: Property '%s' mapeada para '%s'\n", filter.Property, property)

				// Verificar casos especiais para PURCHASE
				if (filter.Property == "event_type" || filter.Property == "events.event_type" || filter.Property == "e.event_type") &&
					filter.Operator == "equals" &&
					filter.Value == "PURCHASE" {
					// Este é um filtro para event_type = 'PURCHASE'
					fmt.Printf("DEBUG REPO: Detectado filtro para event_type = 'PURCHASE'\n")
				}

				// Corrigir aspas para colunas case-sensitive
				if strings.Contains(property, ".") && !strings.Contains(property, "\"") {
					parts := strings.Split(property, ".")
					if len(parts) == 2 && needsQuotesForColumn(parts[0], parts[1]) {
						property = fmt.Sprintf("%s.\"%s\"", parts[0], parts[1])
						fmt.Printf("DEBUG REPO: Adicionando aspas ao campo: %s\n", property)
					}
				}

				switch filter.Operator {
				case "equals":
					sqlOperator = "="
					sqlValue = filter.Value
				case "not_equals":
					// Tratamento especial para comparação com string vazia em campos UTM
					if filter.Value == "" && isUtmField(filter.Property) {
						// Determinar qual tabela usar baseado na propriedade
						var whereClause string
						if strings.HasPrefix(filter.Property, "user.") {
							// Para campos UTM do usuário
							switch {
							case strings.Contains(filter.Property, "utm_source"):
								whereClause = "COALESCE(u.\"initialUtmSource\", '') != ?"
							case strings.Contains(filter.Property, "utm_medium"):
								whereClause = "COALESCE(u.\"initialUtmMedium\", '') != ?"
							case strings.Contains(filter.Property, "utm_campaign"):
								whereClause = "COALESCE(u.\"initialUtmCampaign\", '') != ?"
							case strings.Contains(filter.Property, "utm_content"):
								whereClause = "COALESCE(u.\"initialUtmContent\", '') != ?"
							case strings.Contains(filter.Property, "utm_term"):
								whereClause = "COALESCE(u.\"initialUtmTerm\", '') != ?"
							default:
								whereClause = fmt.Sprintf("COALESCE(%s, '') != ?", property)
							}
						} else if strings.HasPrefix(filter.Property, "session.") {
							// Para campos UTM da sessão
							switch {
							case strings.Contains(filter.Property, "utm_source"):
								whereClause = "COALESCE(s.\"utmSource\", '') != ?"
							case strings.Contains(filter.Property, "utm_medium"):
								whereClause = "COALESCE(s.\"utmMedium\", '') != ?"
							case strings.Contains(filter.Property, "utm_campaign"):
								whereClause = "COALESCE(s.\"utmCampaign\", '') != ?"
							case strings.Contains(filter.Property, "utm_content"):
								whereClause = "COALESCE(s.\"utmContent\", '') != ?"
							case strings.Contains(filter.Property, "utm_term"):
								whereClause = "COALESCE(s.\"utmTerm\", '') != ?"
							default:
								whereClause = fmt.Sprintf("COALESCE(%s, '') != ?", property)
							}
						} else {
							// Para UTMs sem prefixo explícito, usamos tanto do usuário quanto da sessão
							whereClause = "(COALESCE(u.\"initialUtmSource\", '') != ? OR COALESCE(s.\"utmSource\", '') != ?)"
							baseQuery = baseQuery.Where(whereClause, "", "")
							continue
						}

						fmt.Printf("DEBUG REPO: Condição especial para UTM vazio: %s\n", whereClause)
						baseQuery = baseQuery.Where(whereClause, "")
						continue
					}
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
					fmt.Printf("DEBUG REPO: Operador não suportado: %s\n", filter.Operator)
					continue
				}

				// Adicionar o filtro à consulta
				whereClause := fmt.Sprintf("%s %s ?", property, sqlOperator)
				fmt.Printf("DEBUG REPO: Aplicando filtro AND: %s com valor %v\n", whereClause, sqlValue)

				// Tratamento especial para colunas case-sensitive
				if strings.Contains(whereClause, "initialUtm") && !strings.Contains(whereClause, "\"") {
					// Assegurar que colunas initialUtm* tenham aspas duplas
					whereClause = strings.Replace(whereClause, "u.initialUtmSource", "u.\"initialUtmSource\"", -1)
					whereClause = strings.Replace(whereClause, "u.initialUtmMedium", "u.\"initialUtmMedium\"", -1)
					whereClause = strings.Replace(whereClause, "u.initialUtmCampaign", "u.\"initialUtmCampaign\"", -1)
					whereClause = strings.Replace(whereClause, "u.initialUtmContent", "u.\"initialUtmContent\"", -1)
					whereClause = strings.Replace(whereClause, "u.initialUtmTerm", "u.\"initialUtmTerm\"", -1)
				}

				// Tratamento especial para colunas UTM da sessão
				if strings.Contains(whereClause, "utmSource") && !strings.Contains(whereClause, "\"") {
					whereClause = strings.Replace(whereClause, "s.utmSource", "s.\"utmSource\"", -1)
					whereClause = strings.Replace(whereClause, "s.utmMedium", "s.\"utmMedium\"", -1)
					whereClause = strings.Replace(whereClause, "s.utmCampaign", "s.\"utmCampaign\"", -1)
					whereClause = strings.Replace(whereClause, "s.utmContent", "s.\"utmContent\"", -1)
					whereClause = strings.Replace(whereClause, "s.utmTerm", "s.\"utmTerm\"", -1)
				}

				baseQuery = baseQuery.Where(whereClause, sqlValue)
			}
		}
	} else {
		fmt.Printf("DEBUG REPO: Nenhum filtro avançado para aplicar\n")
	}

	// Obter contagem total numa consulta separada para melhorar performance
	countQuery := baseQuery
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao contar eventos: %w", err)
	}

	// Calcular offset para paginação
	offset := (page - 1) * limit

	// Executar a consulta com paginação e ordenação de forma eficiente
	// Garantir que estamos usando o orderBy exato que veio como parâmetro
	query := baseQuery.Order(orderBy).Offset(offset).Limit(limit)

	// SELECT explícito que inclui campos UTM tanto do usuário quanto da sessão
	query = query.Select(`
		e.*, 
		u.user_id,
		u.fullname,
		u.email,
		u.phone,
		u."isClient",
		u."initialCountry", 
		u."initialCity", 
		u."initialRegion",
		u."initialIp",
		u."initialUserAgent",
		u."initialUtmSource", 
		u."initialUtmMedium", 
		u."initialUtmCampaign", 
		u."initialUtmContent", 
		u."initialUtmTerm",
		s.session_id,
		s."isActive",
		s."sessionStart",
		s."lastActivity",
		s.country,
		s.city,
		s.state,
		s."ipAddress",
		s."userAgent",
		s."utmSource",
		s."utmMedium",
		s."utmCampaign",
		s."utmContent",
		s."utmTerm",
		s.duration
	`)

	// Modificar preload para incluir User e Session
	query = query.Preload("User").Preload("Session")

	// Executar a consulta para obter eventos
	if err := query.Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao buscar eventos: %w", err)
	}

	// Se não temos eventos, retornar imediatamente
	if len(events) == 0 {
		return events, total, nil
	}

	// Converter todos os timestamps para horário de Brasília
	for i := range events {
		events[i].EventTime = events[i].EventTime.In(brazilLocation)
		if !events[i].Session.SessionStart.IsZero() {
			events[i].Session.SessionStart = events[i].Session.SessionStart.In(brazilLocation)
		}
		if !events[i].Session.LastActivity.IsZero() {
			events[i].Session.LastActivity = events[i].Session.LastActivity.In(brazilLocation)
		}
	}

	// Debug: Imprimir os primeiros eventos para verificar se temos dados UTM
	if len(events) > 0 {
		fmt.Printf("DEBUG: Primeiro evento antes do processamento - ID: %s, Tipo: %s\n", events[0].EventID, events[0].EventType)
		fmt.Printf("DEBUG: Dados UTM do usuário no banco - initialUtmSource: %s\n", events[0].User.InitialUtmSource)
	}

	// Coletar IDs de eventos PURCHASE para processamento especial
	var purchaseEventIDs []uuid.UUID
	for _, event := range events {
		if event.EventType == "PURCHASE" {
			purchaseEventIDs = append(purchaseEventIDs, event.EventID)
			fmt.Printf("DEBUG: Evento PURCHASE detectado ID=%s, UserID=%s\n", event.EventID, event.UserID)
		}
	}

	// Recarregar explicitamente dados para eventos PURCHASE para garantir dados UTM
	if len(purchaseEventIDs) > 0 {
		fmt.Printf("DEBUG: Recarregando dados para %d eventos PURCHASE\n", len(purchaseEventIDs))
		// Consulta especial apenas para eventos PURCHASE
		var purchaseUserData []struct {
			EventID            uuid.UUID `gorm:"column:event_id"`
			UserID             string    `gorm:"column:user_id"`
			InitialUtmSource   string    `gorm:"column:initialUtmSource"`
			InitialUtmMedium   string    `gorm:"column:initialUtmMedium"`
			InitialUtmCampaign string    `gorm:"column:initialUtmCampaign"`
			InitialUtmContent  string    `gorm:"column:initialUtmContent"`
			InitialUtmTerm     string    `gorm:"column:initialUtmTerm"`
		}

		if err := r.db.Raw(`
			SELECT e.event_id, u.user_id, u."initialUtmSource", u."initialUtmMedium", 
			       u."initialUtmCampaign", u."initialUtmContent", u."initialUtmTerm"
			FROM events e
			JOIN users u ON e.user_id = u.user_id
			WHERE e.event_id IN ? AND e.event_type = 'PURCHASE'`,
			purchaseEventIDs).Scan(&purchaseUserData).Error; err == nil {

			// Criar mapa para acessar rapidamente
			purchaseDataMap := make(map[uuid.UUID]struct {
				UserID             string
				InitialUtmSource   string
				InitialUtmMedium   string
				InitialUtmCampaign string
				InitialUtmContent  string
				InitialUtmTerm     string
			})

			for _, data := range purchaseUserData {
				purchaseDataMap[data.EventID] = struct {
					UserID             string
					InitialUtmSource   string
					InitialUtmMedium   string
					InitialUtmCampaign string
					InitialUtmContent  string
					InitialUtmTerm     string
				}{
					UserID:             data.UserID,
					InitialUtmSource:   data.InitialUtmSource,
					InitialUtmMedium:   data.InitialUtmMedium,
					InitialUtmCampaign: data.InitialUtmCampaign,
					InitialUtmContent:  data.InitialUtmContent,
					InitialUtmTerm:     data.InitialUtmTerm,
				}
				fmt.Printf("DEBUG: Dados PURCHASE carregados: EventID=%s, UTMSource=%s\n",
					data.EventID, data.InitialUtmSource)
			}

			// Agora, aplicar esses dados aos eventos correspondentes
			for i := range events {
				if events[i].EventType == "PURCHASE" {
					if data, ok := purchaseDataMap[events[i].EventID]; ok {
						fmt.Printf("DEBUG: Aplicando dados UTM para PURCHASE %s: %s\n",
							events[i].EventID, data.InitialUtmSource)

						// Garantir que User não seja nil
						if events[i].User.UserID == "" {
							events[i].User = entities.User{
								UserID:             data.UserID,
								InitialUtmSource:   data.InitialUtmSource,
								InitialUtmMedium:   data.InitialUtmMedium,
								InitialUtmCampaign: data.InitialUtmCampaign,
								InitialUtmContent:  data.InitialUtmContent,
								InitialUtmTerm:     data.InitialUtmTerm,
							}
						} else {
							// Atualizar apenas os dados UTM
							events[i].User.InitialUtmSource = data.InitialUtmSource
							events[i].User.InitialUtmMedium = data.InitialUtmMedium
							events[i].User.InitialUtmCampaign = data.InitialUtmCampaign
							events[i].User.InitialUtmContent = data.InitialUtmContent
							events[i].User.InitialUtmTerm = data.InitialUtmTerm
						}

						// Atualizar os campos UTM do evento
						events[i].UtmSource = data.InitialUtmSource
						events[i].UtmMedium = data.InitialUtmMedium
						events[i].UtmCampaign = data.InitialUtmCampaign
						events[i].UtmContent = data.InitialUtmContent
						events[i].UtmTerm = data.InitialUtmTerm

						// Também garantir que UtmData esteja preenchido
						events[i].UtmData = &entities.UtmData{
							UtmSource:   data.InitialUtmSource,
							UtmMedium:   data.InitialUtmMedium,
							UtmCampaign: data.InitialUtmCampaign,
							UtmContent:  data.InitialUtmContent,
							UtmTerm:     data.InitialUtmTerm,
						}
					}
				}
			}
		} else {
			fmt.Printf("ERRO ao recarregar dados PURCHASE: %v\n", err)
		}
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

	// Buscar todos os dados relacionados em uma única consulta otimizada para cada tipo
	// Isso é mais eficiente que usar Preload em consultas separadas

	// Mapas para armazenar os dados relacionados
	sessionMap := make(map[uuid.UUID]entities.Session)
	userMap := make(map[string]entities.User)
	professionMap := make(map[int]entities.Profession)
	productMap := make(map[int]entities.Product)
	funnelMap := make(map[int]entities.Funnel)

	// Buscar usuários apenas se houver usuários
	if len(userIDs) > 0 {
		var users []entities.User
		if err := r.db.
			Select(`user_id, fullname, email, phone, "isClient", 
			       "initialUtmSource", "initialUtmMedium", "initialUtmCampaign", "initialUtmContent", "initialUtmTerm",
			       "initialCountry", "initialCity", "initialRegion", "initialIp", "initialUserAgent"`).
			Where("user_id IN ?", userIDs).
			Find(&users).Error; err == nil {
			for _, u := range users {
				// Debug: mostra dados UTM de cada usuário carregado
				fmt.Printf("DEBUG: Usuário %s tem initialUtmSource=%s, initialCountry=%s\n",
					u.UserID, u.InitialUtmSource, u.InitialCountry)
				userMap[u.UserID] = u
			}
		} else {
			fmt.Printf("ERRO ao buscar usuários: %v\n", err)
		}
	}

	// Buscar sessões apenas se houver sessões
	if len(sessionIDs) > 0 {
		var sessions []entities.Session
		if err := r.db.Where("session_id IN ?", sessionIDs).Find(&sessions).Error; err == nil {
			for _, s := range sessions {
				// Debug: mostra dados UTM de cada sessão carregada
				fmt.Printf("DEBUG: Sessão %s tem utmSource=%s\n", s.ID, s.UtmSource)
				sessionMap[s.ID] = s
			}
		}
	}

	// Buscar profissões apenas se necessário
	if len(professionIDsFromEvents) > 0 {
		var professions []entities.Profession
		if err := r.db.Where("profession_id IN ?", professionIDsFromEvents).Find(&professions).Error; err == nil {
			for _, p := range professions {
				professionMap[p.ProfessionID] = p
			}
		}
	}

	// Buscar produtos apenas se necessário
	if len(productIDs) > 0 {
		var products []entities.Product
		if err := r.db.Where("product_id IN ?", productIDs).Find(&products).Error; err == nil {
			for _, p := range products {
				productMap[p.ProductID] = p
			}
		}
	}

	// Buscar funis apenas se necessário
	if len(funnelIDsFromEvents) > 0 {
		var funnels []entities.Funnel
		if err := r.db.Where("funnel_id IN ?", funnelIDsFromEvents).Find(&funnels).Error; err == nil {
			for _, f := range funnels {
				funnelMap[f.FunnelID] = f
			}
		}
	}

	// Associar dados relacionados aos eventos
	for i := range events {
		// Garantir que temos dados UTM, priorizando usuário, mas usando sessão se necessário
		var utmSource, utmMedium, utmCampaign, utmContent, utmTerm string

		// Verificar se temos dados do usuário
		if events[i].User.UserID != "" {
			utmSource = events[i].User.InitialUtmSource
			utmMedium = events[i].User.InitialUtmMedium
			utmCampaign = events[i].User.InitialUtmCampaign
			utmContent = events[i].User.InitialUtmContent
			utmTerm = events[i].User.InitialUtmTerm
		}

		// Se os dados do usuário estiverem vazios, tentar obter da sessão
		if utmSource == "" && events[i].Session.ID != uuid.Nil {
			utmSource = events[i].Session.UtmSource
		}
		if utmMedium == "" && events[i].Session.ID != uuid.Nil {
			utmMedium = events[i].Session.UtmMedium
		}
		if utmCampaign == "" && events[i].Session.ID != uuid.Nil {
			utmCampaign = events[i].Session.UtmCampaign
		}
		if utmContent == "" && events[i].Session.ID != uuid.Nil {
			utmContent = events[i].Session.UtmContent
		}
		if utmTerm == "" && events[i].Session.ID != uuid.Nil {
			utmTerm = events[i].Session.UtmTerm
		}

		// Atualizar os campos UTM do evento com os dados combinados
		events[i].UtmData = &entities.UtmData{
			UtmSource:   utmSource,
			UtmMedium:   utmMedium,
			UtmCampaign: utmCampaign,
			UtmContent:  utmContent,
			UtmTerm:     utmTerm,
		}

		// Também definir os campos UTM individuais para compatibilidade
		events[i].UtmSource = utmSource
		events[i].UtmMedium = utmMedium
		events[i].UtmCampaign = utmCampaign
		events[i].UtmContent = utmContent
		events[i].UtmTerm = utmTerm

		// Debug para eventos PURCHASE para garantir que os dados UTM estão sendo associados
		if events[i].EventType == "PURCHASE" {
			fmt.Printf("DEBUG: Evento PURCHASE %s - UTM final: source=%s, medium=%s, campaign=%s\n",
				events[i].EventID, utmSource, utmMedium, utmCampaign)
		}

		// Debug - mostrar dados finais do evento
		fmt.Printf("DEBUG: Evento %s - UTM data final: %+v\n",
			events[i].EventID, events[i].UtmData)

		// Verificar se temos dados do usuário - ÚNICA FONTE DE UTMs e dados geográficos
		if user, ok := userMap[events[i].UserID]; ok {
			events[i].User = user

			// Debug para eventos PURCHASE para garantir que os dados UTM estão sendo associados
			if events[i].EventType == "PURCHASE" {
				fmt.Printf("DEBUG: Associando UTMs ao evento PURCHASE %s: %s, %s, %s\n",
					events[i].EventID, user.InitialUtmSource, user.InitialUtmMedium, user.InitialUtmCampaign)

				// Verificação adicional para garantir que UtmData está sendo definido corretamente
				if events[i].UtmData != nil {
					fmt.Printf("DEBUG: UtmData para evento PURCHASE %s: %+v\n", events[i].EventID, *events[i].UtmData)
				} else {
					fmt.Printf("ERRO: UtmData é nil para evento PURCHASE %s\n", events[i].EventID)
					// Garantir que UtmData não seja nil
					events[i].UtmData = &entities.UtmData{
						UtmSource:   user.InitialUtmSource,
						UtmMedium:   user.InitialUtmMedium,
						UtmCampaign: user.InitialUtmCampaign,
						UtmContent:  user.InitialUtmContent,
						UtmTerm:     user.InitialUtmTerm,
					}
				}
			}

			// Debug - mostrar dados finais do evento
			fmt.Printf("DEBUG: Evento %s - UTM data final do usuário: %+v\n",
				events[i].EventID, events[i].UtmData)
		} else {
			// Se não temos usuário, definir UTM vazio
			events[i].UtmData = &entities.UtmData{}

			// Se for um evento PURCHASE sem usuário, gerar alerta
			if events[i].EventType == "PURCHASE" {
				fmt.Printf("ALERTA: Evento PURCHASE %s não tem usuário associado!\n", events[i].EventID)

				// Consulta especial para tentar recuperar os dados UTM para este evento
				var userDirectQuery entities.User
				if err := r.db.
					Select(`user_id, "initialUtmSource", "initialUtmMedium", "initialUtmCampaign", "initialUtmContent", "initialUtmTerm"`).
					Where("user_id = ?", events[i].UserID).
					First(&userDirectQuery).Error; err == nil {

					// Encontramos o usuário, aplicar UTMs
					fmt.Printf("DEBUG: Recuperado usuário via consulta direta para PURCHASE %s: %s\n",
						events[i].EventID, userDirectQuery.UserID)

					events[i].UtmData = &entities.UtmData{
						UtmSource:   userDirectQuery.InitialUtmSource,
						UtmMedium:   userDirectQuery.InitialUtmMedium,
						UtmCampaign: userDirectQuery.InitialUtmCampaign,
						UtmContent:  userDirectQuery.InitialUtmContent,
						UtmTerm:     userDirectQuery.InitialUtmTerm,
					}

					// Também definir campos individuais
					events[i].UtmSource = userDirectQuery.InitialUtmSource
					events[i].UtmMedium = userDirectQuery.InitialUtmMedium
					events[i].UtmCampaign = userDirectQuery.InitialUtmCampaign
					events[i].UtmContent = userDirectQuery.InitialUtmContent
					events[i].UtmTerm = userDirectQuery.InitialUtmTerm
				}
			}
		}

		// Verificar se temos dados da sessão (apenas para não-UTM dados)
		if session, ok := sessionMap[events[i].SessionID]; ok {
			events[i].Session = session
			// NÃO copiar dados UTM da sessão de forma alguma
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

	if len(advancedFilters) > 0 {
		hasUtmSourceNotEmptyFilter := false
		for _, filter := range advancedFilters {
			// Verificar apenas filtros explícitos de user.utm_source, ignorando session.utm_*
			if filter.Property == "user.utm_source" && filter.Operator == "not_equals" && filter.Value == "" {
				hasUtmSourceNotEmptyFilter = true
				break
			}
		}

		if hasUtmSourceNotEmptyFilter {
			// Filtragem final: manter apenas eventos com UTM não vazio
			fmt.Println("DEBUG: Aplicando filtro final para remover eventos com user.utm_source vazio")
			filteredEvents := []entities.Event{}
			for _, event := range events {
				// Verificar UTM EXCLUSIVAMENTE no usuário (initialUtmSource)
				hasValidUtm := false

				// Apenas usuários com initialUtmSource preenchido passam no filtro
				if event.User.UserID != "" && event.User.InitialUtmSource != "" {
					hasValidUtm = true
					fmt.Printf("DEBUG: Evento %s passou no filtro com user.utm_source=%s\n",
						event.EventID, event.User.InitialUtmSource)
				}

				if hasValidUtm {
					filteredEvents = append(filteredEvents, event)
				} else {
					fmt.Printf("DEBUG: Removendo evento %s com user.utm_source vazio ou sem usuário\n", event.EventID)
				}
			}

			fmt.Printf("DEBUG: Após filtro final, eventos: %d -> %d\n", len(events), len(filteredEvents))
			events = filteredEvents
			// Ajustar o total se necessário
			if int64(len(filteredEvents)) < total {
				total = int64(len(filteredEvents))
			}
		}
	}

	return events, total, nil
}

// Função auxiliar para processar o nome da propriedade
func processPropertyName(rawProperty string) string {
	property := rawProperty
	fmt.Printf("DEBUG: Processando propriedade: %s\n", rawProperty)

	if strings.Contains(property, ".") {
		parts := strings.Split(property, ".")
		if len(parts) == 2 {
			tableName := parts[0]
			columnName := parts[1]
			fmt.Printf("DEBUG: Propriedade dividida em: tabela=%s, coluna=%s\n", tableName, columnName)

			// Mapear campos UTM específicos para a tabela correspondente
			if isUtmField(columnName) {
				if tableName == "user" {
					// Mapear UTMs do usuário
					switch columnName {
					case "utm_source":
						return "u.\"initialUtmSource\""
					case "utm_medium":
						return "u.\"initialUtmMedium\""
					case "utm_campaign":
						return "u.\"initialUtmCampaign\""
					case "utm_content":
						return "u.\"initialUtmContent\""
					case "utm_term":
						return "u.\"initialUtmTerm\""
					}
				} else if tableName == "session" {
					// Mapear UTMs da sessão com nomes de colunas corretos
					switch columnName {
					case "utm_source":
						return "s.\"utmSource\""
					case "utm_medium":
						return "s.\"utmMedium\""
					case "utm_campaign":
						return "s.\"utmCampaign\""
					case "utm_content":
						return "s.\"utmContent\""
					case "utm_term":
						return "s.\"utmTerm\""
					}
				}
			}

			// Mapear campos geográficos para o usuário
			if tableName == "user" {
				switch columnName {
				case "country":
					return "u.\"initialCountry\""
				case "city":
					return "u.\"initialCity\""
				case "state", "region":
					return "u.\"initialRegion\""
				case "ip", "ip_address", "ipAddress":
					return "u.\"initialIp\""
				case "user_agent", "userAgent":
					return "u.\"initialUserAgent\""
				}
			}

			// Verificar outras colunas que precisam de aspas
			if needsQuotes(columnName) {
				if tableName == "user" {
					return fmt.Sprintf("u.\"%s\"", columnName)
				} else if tableName == "session" {
					return fmt.Sprintf("s.\"%s\"", columnName)
				}
			}

			// Para campos não-UTM, usar a tabela correspondente com o alias correto
			switch tableName {
			case "user":
				property = "u." + columnName
			case "session":
				property = "s." + columnName
			case "profession":
				property = "professions." + columnName
			case "product":
				property = "products." + columnName
			case "funnel":
				property = "funnels." + columnName
			case "event", "events":
				property = "e." + columnName
			default:
				property = "e." + property
			}
			fmt.Printf("DEBUG: Mapeando para: %s\n", property)
		}
	} else {
		// Se não for propriedade aninhada, assumir que é do evento
		property = "e." + property
		fmt.Printf("DEBUG: Propriedade simples, assumindo events: %s\n", property)
	}

	fmt.Printf("DEBUG: Propriedade final: %s\n", property)
	return property
}

// Função auxiliar para verificar se um nome de coluna precisa de aspas duplas
func needsQuotes(columnName string) bool {
	// Colunas case-sensitive ou que contenham caracteres especiais
	return strings.ContainsAny(columnName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") ||
		strings.ContainsAny(columnName, "-") ||
		strings.HasPrefix(columnName, "initial") ||
		strings.HasPrefix(columnName, "utm") ||
		strings.HasPrefix(columnName, "is")
}

// Função auxiliar para verificar se um nome de campo (não a propriedade completa) é um campo UTM
func isUtmField(columnName string) bool {
	return columnName == "utm_source" ||
		columnName == "utm_medium" ||
		columnName == "utm_campaign" ||
		columnName == "utm_content" ||
		columnName == "utm_term"
}

// Função auxiliar para verificar se uma coluna específica precisa de aspas
func needsQuotesForColumn(table, column string) bool {
	// Verificar se o nome da coluna contém letras maiúsculas
	if strings.ContainsAny(column, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return true
	}

	// Verificar prefixos específicos que indicam campos que precisam de aspas
	if strings.HasPrefix(column, "initial") ||
		strings.HasPrefix(column, "utm") ||
		strings.HasPrefix(column, "is") {
		return true
	}

	// Verificar combinações específicas de tabela.coluna que necessitam de aspas
	specialCases := map[string]bool{
		"u.initialUtmSource":   true,
		"u.initialUtmMedium":   true,
		"u.initialUtmCampaign": true,
		"u.initialUtmContent":  true,
		"u.initialUtmTerm":     true,
		"u.initialCountry":     true,
		"u.initialCity":        true,
		"u.initialRegion":      true,
		"u.initialIp":          true,
		"u.initialUserAgent":   true,
		"u.isClient":           true,
		"s.utmSource":          true,
		"s.utmMedium":          true,
		"s.utmCampaign":        true,
		"s.utmContent":         true,
		"s.utmTerm":            true,
	}

	key := fmt.Sprintf("%s.%s", table, column)
	return specialCases[key]
}
