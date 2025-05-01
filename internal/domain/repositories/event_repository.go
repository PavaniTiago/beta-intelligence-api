package repositories

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/utils"
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
	CountEvents(from, to time.Time, timeFrom, timeTo string, eventType string, professionIDs, funnelIDs []int, advancedFilters []AdvancedFilter, filterCondition string) (int64, error)
	CountEventsByPeriods(periods []string, eventType string, advancedFilters []AdvancedFilter, funnelID int, professionID int) (map[string]int64, error)
	GetEventsDateRange(eventType string) (time.Time, time.Time, error)
	CountEventsByDateRange(from, to time.Time, timeFrom, timeTo string, eventType string, professionIDs, funnelIDs []int, logicalOperator string) (int64, error)
	GetEventsByHours(date time.Time, eventType, userID, professionID, productID, funnelID string, landingPage string) (map[string]int64, error)
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

	// Obter localização de Brasília usando a função centralizada
	brazilLocation := utils.GetBrasilLocation()

	// Converter timestamps para horário de Brasília
	if !from.IsZero() {
		from = from.In(brazilLocation)
	}
	if !to.IsZero() {
		to = to.In(brazilLocation)
	}

	// Verificar se estamos buscando apenas eventos PURCHASE
	isPurchaseOnlyQuery := false
	needsSessionJoin := false
	needsUserJoin := false
	needsProfessionJoin := false
	needsProductJoin := false
	needsFunnelJoin := false
	needsSurveyJoin := false

	// Análise de filtros para determinar JOINs necessários
	for _, filter := range advancedFilters {
		if (filter.Property == "event_type" || filter.Property == "events.event_type" || filter.Property == "e.event_type") &&
			filter.Operator == "equals" &&
			filter.Value == "PURCHASE" {
			isPurchaseOnlyQuery = true
		}

		// Verificar se precisamos de JOIN com sessions
		if strings.Contains(filter.Property, "session.") ||
			strings.Contains(filter.Property, "utm") ||
			filter.Property == "utmSource" ||
			filter.Property == "utmMedium" ||
			filter.Property == "utmCampaign" ||
			filter.Property == "utmContent" ||
			filter.Property == "utmTerm" {
			needsSessionJoin = true
		}

		// Verificar se precisamos de JOIN com users
		if strings.Contains(filter.Property, "user.") ||
			strings.Contains(filter.Property, "initial") {
			needsUserJoin = true
		}

		// Verificar se precisamos de JOIN com professions
		if strings.Contains(filter.Property, "profession.") {
			needsProfessionJoin = true
		}

		// Verificar se precisamos de JOIN com products
		if strings.Contains(filter.Property, "product.") {
			needsProductJoin = true
		}

		// Verificar se precisamos de JOIN com funnels
		if strings.Contains(filter.Property, "funnel.") {
			needsFunnelJoin = true
		}
	}

	// Se temos filtros de profissão, precisamos do JOIN
	if len(professionIDs) > 0 {
		needsProfessionJoin = true
	}

	// Se temos filtros de funil, precisamos do JOIN
	if len(funnelIDs) > 0 {
		needsFunnelJoin = true
	}

	// Na página 1, sempre incluímos todos os JOINs para dar dados completos
	if page == 1 {
		needsUserJoin = true
		needsSessionJoin = true
		needsProfessionJoin = true
		needsProductJoin = true
		needsFunnelJoin = true
		needsSurveyJoin = true
	}

	// Inicializar a consulta base - sempre usar alias 'e' para eventos
	baseQuery := r.db.Model(&entities.Event{}).Table("events e")

	// JOIN com users (obrigatório para UTMs do usuário)
	if needsUserJoin {
		baseQuery = baseQuery.Joins("JOIN users u ON e.user_id = u.user_id")
	} else {
		// Se não for necessário o JOIN completo, usar LEFT JOIN apenas para campos básicos
		baseQuery = baseQuery.Joins("LEFT JOIN users u ON e.user_id = u.user_id")
	}

	// Aplicar outros JOINs apenas se necessários
	if needsSessionJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN sessions s ON e.session_id = s.session_id")
	}

	if needsProfessionJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN professions ON e.profession_id = professions.profession_id")
	}

	if needsProductJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN products ON e.product_id = products.product_id")
	}

	if needsFunnelJoin {
		baseQuery = baseQuery.Joins("LEFT JOIN funnels ON e.funnel_id = funnels.funnel_id")
	}

	// JOINs para surveys apenas se necessários
	if needsSurveyJoin {
		baseQuery = baseQuery.
			// Join com surveys baseado no funnel_id
			Joins("LEFT JOIN surveys sv ON sv.funnel_id = funnels.funnel_id").
			// Join com survey_responses baseado no event_id e survey_id
			Joins("LEFT JOIN survey_responses sr ON (sr.event_id = e.event_id AND sr.survey_id = sv.survey_id)")
	}

	// Se for uma consulta específica para PURCHASE, aplicar diretamente
	if isPurchaseOnlyQuery {
		fmt.Printf("DEBUG: Aplicando filtro direto para PURCHASE\n")
		baseQuery = baseQuery.Where("e.event_type = ?", "PURCHASE")
	}

	// Aplicar filtro de data em UTC diretamente, sem conversão de timezone no SQL
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
				fmt.Printf("Events: Ajustando horário de início para: %s\n", fromTime.Format("2006-01-02 15:04:05"))
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
				fmt.Printf("Events: Ajustando horário de fim para: %s\n", toTime.Format("2006-01-02 15:04:05"))
			}
		} else {
			// Se não fornecido, usar o fim do dia
			toTime = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())
		}

		// Converter para UTC antes de aplicar à consulta
		fromTimeUTC := fromTime.UTC()
		toTimeUTC := toTime.UTC()

		// Aplicar filtro em UTC diretamente, permitindo uso de índices
		baseQuery = baseQuery.Where("e.event_time BETWEEN ? AND ?",
			fromTimeUTC.Format("2006-01-02 15:04:05"),
			toTimeUTC.Format("2006-01-02 15:04:05"))

		fmt.Printf("Events: Filtro de data aplicado em UTC: %s até %s\n",
			fromTimeUTC.Format("2006-01-02 15:04:05"),
			toTimeUTC.Format("2006-01-02 15:04:05"))
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

				// Verificar se estamos tentando filtrar UTMs direto em events
				if strings.Contains(property, "e.utm") || strings.Contains(property, "utmCampaign") ||
					strings.Contains(property, "utmSource") || strings.Contains(property, "utmMedium") ||
					strings.Contains(property, "utmContent") || strings.Contains(property, "utmTerm") {

					// Converter para usar a tabela de sessões
					if strings.Contains(property, "e.utmSource") {
						property = "s.\"utmSource\""
					} else if strings.Contains(property, "e.utmMedium") {
						property = "s.\"utmMedium\""
					} else if strings.Contains(property, "e.utmCampaign") {
						property = "s.\"utmCampaign\""
					} else if strings.Contains(property, "e.utmContent") {
						property = "s.\"utmContent\""
					} else if strings.Contains(property, "e.utmTerm") {
						property = "s.\"utmTerm\""
					} else if property == "utmSource" || property == "e.\"utmSource\"" {
						property = "s.\"utmSource\""
					} else if property == "utmMedium" || property == "e.\"utmMedium\"" {
						property = "s.\"utmMedium\""
					} else if property == "utmCampaign" || property == "e.\"utmCampaign\"" {
						property = "s.\"utmCampaign\""
					} else if property == "utmContent" || property == "e.\"utmContent\"" {
						property = "s.\"utmContent\""
					} else if property == "utmTerm" || property == "e.\"utmTerm\"" {
						property = "s.\"utmTerm\""
					}

					fmt.Printf("DEBUG REPO: Corrigindo UTM para usar sessão: %s -> %s\n", filter.Property, property)
				}

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

				// Verificar se estamos tentando filtrar UTMs direto em events
				if strings.Contains(property, "e.utm") || strings.Contains(property, "utmCampaign") ||
					strings.Contains(property, "utmSource") || strings.Contains(property, "utmMedium") ||
					strings.Contains(property, "utmContent") || strings.Contains(property, "utmTerm") {

					// Converter para usar a tabela de sessões
					if strings.Contains(property, "e.utmSource") {
						property = "s.\"utmSource\""
					} else if strings.Contains(property, "e.utmMedium") {
						property = "s.\"utmMedium\""
					} else if strings.Contains(property, "e.utmCampaign") {
						property = "s.\"utmCampaign\""
					} else if strings.Contains(property, "e.utmContent") {
						property = "s.\"utmContent\""
					} else if strings.Contains(property, "e.utmTerm") {
						property = "s.\"utmTerm\""
					} else if property == "utmSource" || property == "e.\"utmSource\"" {
						property = "s.\"utmSource\""
					} else if property == "utmMedium" || property == "e.\"utmMedium\"" {
						property = "s.\"utmMedium\""
					} else if property == "utmCampaign" || property == "e.\"utmCampaign\"" {
						property = "s.\"utmCampaign\""
					} else if property == "utmContent" || property == "e.\"utmContent\"" {
						property = "s.\"utmContent\""
					} else if property == "utmTerm" || property == "e.\"utmTerm\"" {
						property = "s.\"utmTerm\""
					}

					fmt.Printf("DEBUG REPO: Corrigindo UTM para usar sessão: %s -> %s\n", filter.Property, property)
				}

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
	query := baseQuery.Order(orderBy).Offset(offset).Limit(limit)

	// Para consultas comuns, usar SELECT mais enxuto
	if page > 1 || (!needsUserJoin && !needsSessionJoin && !needsSurveyJoin) {
		// SELECT básico apenas com campos essenciais
		query = query.Select(`
			e.event_id, 
			e.event_name,
			e.event_time,
			e.event_type,
			e.event_source,
			e.user_id,
			e.profession_id,
			e.product_id,
			e.funnel_id,
			e.event_propeties
		`)
	} else {
		// Para primeira página ou quando filtros específicos são necessários
		// usar SELECT completo com todos os campos relacionados
		fieldsToSelect := []string{
			"e.*",
			"e.event_propeties",
		}

		// Adicionar campos de usuário se necessário
		if needsUserJoin {
			fieldsToSelect = append(fieldsToSelect, []string{
				"u.user_id",
				"u.fullname",
				"u.email",
				"u.phone",
				"u.\"isClient\"",
				"u.\"initialCountry\"",
				"u.\"initialCity\"",
				"u.\"initialRegion\"",
				"u.\"initialIp\"",
				"u.\"initialUserAgent\"",
				"u.\"initialUtmSource\"",
				"u.\"initialUtmMedium\"",
				"u.\"initialUtmCampaign\"",
				"u.\"initialUtmContent\"",
				"u.\"initialUtmTerm\"",
			}...)
		}

		// Adicionar campos de sessão se necessário
		if needsSessionJoin {
			fieldsToSelect = append(fieldsToSelect, []string{
				"s.session_id",
				"s.\"isActive\"",
				"s.\"sessionStart\"",
				"s.\"lastActivity\"",
				"s.country",
				"s.city",
				"s.state",
				"s.\"ipAddress\"",
				"s.\"userAgent\"",
				"s.\"utmSource\"",
				"s.\"utmMedium\"",
				"s.\"utmCampaign\"",
				"s.\"utmContent\"",
				"s.\"utmTerm\"",
				"s.duration",
			}...)
		}

		// Adicionar campos de survey se necessário
		if needsSurveyJoin {
			fieldsToSelect = append(fieldsToSelect, []string{
				"sv.survey_id",
				"sv.survey_name",
				"sv.funnel_id as survey_funnel_id",
				"sr.id as survey_response_id",
				"sr.total_score",
				"sr.completed",
				"sr.faixa",
				"sr.created_at as survey_response_created_at",
			}...)
		}

		// Aplicar lista de campos completa
		query = query.Select(strings.Join(fieldsToSelect, ", "))
	}

	// Modificar preload para incluir User e Session
	query = query.
		Preload("User").
		Preload("Session")

	// Executar a consulta para obter eventos
	if err := query.Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("erro ao buscar eventos: %w", err)
	}

	// Adicionar log para verificar os dados brutos retornados pela consulta
	if len(events) > 0 {
		fmt.Printf("DEBUG CONSULTA CRUA: SQL = %s\n", query.Statement.SQL.String())
		if events[0].User.UserID != "" {
			fmt.Printf("DEBUG CONSULTA CRUA: Primeiro usuário - ID=%s, Country=%s, UTM=%s\n",
				events[0].User.UserID, events[0].User.InitialCountry, events[0].User.InitialUtmSource)
		} else {
			fmt.Printf("DEBUG CONSULTA CRUA: Primeiro usuário é NIL\n")
		}
	}

	// Se não temos eventos, retornar imediatamente
	if len(events) == 0 {
		return events, total, nil
	}

	// Debug: Verificar dados geo e UTM do primeiro evento logo após a consulta
	if len(events) > 0 {
		fmt.Printf("DEBUG CONSULTA INICIAL: Evento %s - User.InitialCountry=%s, User.InitialCity=%s\n",
			events[0].EventID, events[0].User.InitialCountry, events[0].User.InitialCity)
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
		fmt.Printf("DEBUG: Evento após processamento UTM - ID: %s\n", events[0].EventID)
	}

	// Processar dados de survey de forma otimizada
	if len(events) > 0 && needsSurveyJoin {
		// Coletar todos os funnel_ids e event_ids
		var funnelIDs []int
		var eventIDs []string

		funnelIDMap := make(map[int]bool)
		for _, event := range events {
			if event.FunnelID > 0 {
				// Evitar duplicatas
				if !funnelIDMap[event.FunnelID] {
					funnelIDs = append(funnelIDs, event.FunnelID)
					funnelIDMap[event.FunnelID] = true
				}
				eventIDs = append(eventIDs, event.EventID.String())
			}
		}

		// Se temos funnels, buscamos todos os surveys relacionados de uma vez
		if len(funnelIDs) > 0 {
			type SurveyData struct {
				SurveyID   int64
				SurveyName string
				FunnelID   int
				ResponseID *string
				EventID    *string
				TotalScore *int
				Completed  *bool
				Faixa      *string
				CreatedAt  *time.Time
			}

			var surveyData []SurveyData

			// Uma única consulta para obter todos os dados de survey relevantes
			query := `SELECT 
				sv.survey_id, sv.survey_name, sv.funnel_id,
				sr.id, sr.event_id, sr.total_score, sr.completed, sr.faixa, sr.created_at
				FROM surveys sv
				LEFT JOIN survey_responses sr ON sr.survey_id = sv.survey_id
				WHERE sv.funnel_id IN (?) AND (sr.event_id IN (?) OR sr.event_id IS NULL)`

			// Substituir placeholders para lista IN
			query = strings.Replace(query, "(?) AND (sr.event_id IN (?)", fmt.Sprintf("(%s) AND (sr.event_id IN (%s)",
				strings.Trim(strings.Repeat("?,", len(funnelIDs)), ","),
				strings.Trim(strings.Repeat("?,", len(eventIDs)), ",")), 1)

			// Preparar os argumentos
			args := make([]interface{}, 0, len(funnelIDs)+len(eventIDs))
			for _, id := range funnelIDs {
				args = append(args, id)
			}
			for _, id := range eventIDs {
				args = append(args, id)
			}

			// Executar a consulta
			if err := r.db.Raw(query, args...).Scan(&surveyData).Error; err != nil {
				fmt.Printf("Erro ao buscar dados de survey: %v\n", err)
			} else {
				// Mapear surveys por funnel_id e respostas por event_id
				surveysMap := make(map[int]*entities.Survey)
				responsesMap := make(map[string]*entities.SurveyResponse)

				// Primeiro mapear todos os surveys
				for _, sd := range surveyData {
					if _, exists := surveysMap[sd.FunnelID]; !exists {
						surveysMap[sd.FunnelID] = &entities.Survey{
							SurveyID: sd.SurveyID,
							Name:     sd.SurveyName,
							FunnelID: sd.FunnelID,
						}
					}

					// Se temos uma resposta, mapeá-la pelo event_id
					if sd.ResponseID != nil && sd.EventID != nil && *sd.ResponseID != "" && *sd.EventID != "" {
						totalScoreVal := 0
						if sd.TotalScore != nil {
							totalScoreVal = *sd.TotalScore
						}

						completedVal := false
						if sd.Completed != nil {
							completedVal = *sd.Completed
						}

						faixaVal := ""
						if sd.Faixa != nil {
							faixaVal = *sd.Faixa
						}

						createdAtVal := time.Now()
						if sd.CreatedAt != nil {
							createdAtVal = *sd.CreatedAt
						}

						response := &entities.SurveyResponse{
							ID:         *sd.ResponseID,
							SurveyID:   sd.SurveyID,
							EventID:    *sd.EventID,
							TotalScore: totalScoreVal,
							Completed:  completedVal,
							Faixa:      faixaVal,
							CreatedAt:  createdAtVal,
						}

						responsesMap[*sd.EventID] = response
					}
				}

				// Depois associar os surveys e respostas aos eventos
				for i, event := range events {
					if event.FunnelID > 0 {
						// Associar survey
						if survey, exists := surveysMap[event.FunnelID]; exists {
							events[i].Survey = survey
						}

						// Associar response
						if response, exists := responsesMap[event.EventID.String()]; exists {
							events[i].SurveyResponse = response
						}
					}
				}

				// Coletar todos os IDs de survey_response para buscar as respostas
				var surveyResponseIDs []string
				for _, response := range responsesMap {
					surveyResponseIDs = append(surveyResponseIDs, response.ID)
				}

				// Buscar as respostas de survey em uma única consulta
				if len(surveyResponseIDs) > 0 {
					var answers []entities.SurveyAnswer
					if err := r.db.Where("survey_response_id IN ?", surveyResponseIDs).Find(&answers).Error; err == nil {
						// Mapear respostas por ID de survey_response
						answerMap := make(map[string][]entities.SurveyAnswer)
						for _, answer := range answers {
							answerMap[answer.SurveyResponseID] = append(answerMap[answer.SurveyResponseID], answer)
						}

						// Anexar respostas aos eventos
						for i, event := range events {
							if event.SurveyResponse != nil {
								if answers, ok := answerMap[event.SurveyResponse.ID]; ok {
									events[i].SurveyResponse.Answers = answers
								}
							}
						}
					}
				}
			}
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
		// Garantir que temos dados UTM, priorizando sessão, e usando usuário se necessário
		var utmSource, utmMedium, utmCampaign, utmContent, utmTerm string

		// Verificar se temos dados da sessão primeiro
		if events[i].Session.ID != uuid.Nil {
			utmSource = events[i].Session.UtmSource
			utmMedium = events[i].Session.UtmMedium
			utmCampaign = events[i].Session.UtmCampaign
			utmContent = events[i].Session.UtmContent
			utmTerm = events[i].Session.UtmTerm
		}

		// Se os dados da sessão estiverem vazios, tentar obter do usuário
		if utmSource == "" && events[i].User.UserID != "" {
			utmSource = events[i].User.InitialUtmSource
		}
		if utmMedium == "" && events[i].User.UserID != "" {
			utmMedium = events[i].User.InitialUtmMedium
		}
		if utmCampaign == "" && events[i].User.UserID != "" {
			utmCampaign = events[i].User.InitialUtmCampaign
		}
		if utmContent == "" && events[i].User.UserID != "" {
			utmContent = events[i].User.InitialUtmContent
		}
		if utmTerm == "" && events[i].User.UserID != "" {
			utmTerm = events[i].User.InitialUtmTerm
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

		// Verificar se temos dados da sessão para preencher dados geográficos
		if events[i].Session.ID != uuid.Nil {
			// Usar informações geográficas da sessão se disponíveis
			events[i].InitialCountry = events[i].Session.Country
			events[i].InitialRegion = events[i].Session.State
			events[i].InitialCity = events[i].Session.City
			events[i].InitialIp = events[i].Session.IpAddress
		}

		// Se dados geográficos da sessão não estiverem disponíveis, usar do usuário
		if events[i].InitialCountry == "" && events[i].User.UserID != "" {
			events[i].InitialCountry = events[i].User.InitialCountry
			events[i].InitialCountryCode = events[i].User.InitialCountryCode
			events[i].InitialRegion = events[i].User.InitialRegion
			events[i].InitialCity = events[i].User.InitialCity
			events[i].InitialZip = events[i].User.InitialZip
			events[i].InitialIp = events[i].User.InitialIp
		}

		// Se for um evento PURCHASE sem dados UTM, fazer uma tentativa extra
		if events[i].EventType == "PURCHASE" && events[i].UtmData != nil && events[i].UtmData.UtmSource == "" {
			fmt.Printf("ALERTA: Evento PURCHASE %s não tem dados UTM na sessão ou usuário!\n", events[i].EventID)

			// Consulta especial para tentar recuperar os dados UTM da sessão para este evento
			var sessionDirectQuery entities.Session
			if err := r.db.
				Select(`session_id, "utmSource", "utmMedium", "utmCampaign", "utmContent", "utmTerm"`).
				Where("session_id = ?", events[i].SessionID).
				First(&sessionDirectQuery).Error; err == nil {

				// Encontramos a sessão, aplicar UTMs
				fmt.Printf("DEBUG: Recuperada sessão via consulta direta para PURCHASE %s: %s\n",
					events[i].EventID, sessionDirectQuery.ID)

				events[i].UtmData = &entities.UtmData{
					UtmSource:   sessionDirectQuery.UtmSource,
					UtmMedium:   sessionDirectQuery.UtmMedium,
					UtmCampaign: sessionDirectQuery.UtmCampaign,
					UtmContent:  sessionDirectQuery.UtmContent,
					UtmTerm:     sessionDirectQuery.UtmTerm,
				}

				// Também definir campos individuais
				events[i].UtmSource = sessionDirectQuery.UtmSource
				events[i].UtmMedium = sessionDirectQuery.UtmMedium
				events[i].UtmCampaign = sessionDirectQuery.UtmCampaign
				events[i].UtmContent = sessionDirectQuery.UtmContent
				events[i].UtmTerm = sessionDirectQuery.UtmTerm
			} else if events[i].User.UserID != "" {
				// Se falhar, tentar o usuário como último recurso
				var userDirectQuery entities.User
				if err := r.db.
					Select(`user_id, "initialUtmSource", "initialUtmMedium", "initialUtmCampaign", "initialUtmContent", "initialUtmTerm"`).
					Where("user_id = ?", events[i].UserID).
					First(&userDirectQuery).Error; err == nil {

					// Encontramos o usuário, aplicar UTMs como último recurso
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
		}

		// Verificar se temos dados do usuário (apenas para dados não-UTM)
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

	if len(advancedFilters) > 0 {
		hasUtmSourceNotEmptyFilter := false
		for _, filter := range advancedFilters {
			// Verificar apenas filtros explícitos de session.utm_source, ignorando user.utm_*
			if filter.Property == "session.utm_source" && filter.Operator == "not_equals" && filter.Value == "" {
				hasUtmSourceNotEmptyFilter = true
				break
			}
		}

		if hasUtmSourceNotEmptyFilter {
			// Filtragem final: manter apenas eventos com UTM não vazio
			fmt.Println("DEBUG: Aplicando filtro final para remover eventos com session.utm_source vazio")
			filteredEvents := []entities.Event{}
			for _, event := range events {
				// Verificar UTM EXCLUSIVAMENTE na sessão (utmSource)
				hasValidUtm := false

				// Apenas sessões com utmSource preenchido passam no filtro
				if event.Session.ID != uuid.Nil && event.Session.UtmSource != "" {
					hasValidUtm = true
					fmt.Printf("DEBUG: Evento %s passou no filtro com session.utm_source=%s\n",
						event.EventID, event.Session.UtmSource)
				}

				if hasValidUtm {
					filteredEvents = append(filteredEvents, event)
				} else {
					fmt.Printf("DEBUG: Removendo evento %s com session.utm_source vazio ou sem sessão\n", event.EventID)
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

	// Verificar a estrutura final do primeiro evento (se houver)
	if len(events) > 0 {
		event := events[0]
		fmt.Printf("\nESTRUTURA FINAL DO EVENTO: %s\n", event.EventID)
		fmt.Printf("  Dados principais: type=%s, source=%s\n", event.EventType, event.EventSource)
		fmt.Printf("  Dados geo principais: initialCountry=%s, initialCity=%s, initialRegion=%s\n",
			event.InitialCountry, event.InitialCity, event.InitialRegion)
		fmt.Printf("  Dados UTM principais: utmSource=%s, utmMedium=%s\n",
			event.UtmSource, event.UtmMedium)
		fmt.Printf("  Dados do usuário: initialCountry=%s, initialCity=%s, initialRegion=%s\n",
			event.User.InitialCountry, event.User.InitialCity, event.User.InitialRegion)
		fmt.Printf("  Dados UTM do usuário: initialUtmSource=%s, initialUtmMedium=%s\n",
			event.User.InitialUtmSource, event.User.InitialUtmMedium)
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
		// Se não for propriedade aninhada, verificar se é um campo UTM
		if isUtmField(property) {
			// Se for UTM sem prefixo, mapear para a sessão por padrão
			switch property {
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
		} else {
			// Se não for UTM e não tiver prefixo, assumir que é do evento
			property = "e." + property
			fmt.Printf("DEBUG: Propriedade simples, assumindo events: %s\n", property)
		}
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

// CountEvents conta eventos com filtros aplicados, incluindo tipo específico
func (r *eventRepository) CountEvents(from, to time.Time, timeFrom, timeTo string, eventType string, professionIDs, funnelIDs []int, advancedFilters []AdvancedFilter, filterCondition string) (int64, error) {
	// Obter localização de Brasília usando a função centralizada
	brazilLocation := utils.GetBrasilLocation()

	// Converter timestamps para horário de Brasília
	if !from.IsZero() {
		from = from.In(brazilLocation)
	}
	if !to.IsZero() {
		to = to.In(brazilLocation)
	}

	// Inicializar a consulta base para contagem
	query := r.db.Model(&entities.Event{}).Table("events e")

	// JOIN com users para ter acesso a UTMs
	query = query.
		Joins("JOIN users u ON e.user_id = u.user_id")

	// Aplicar filtro de tipo de evento, se fornecido
	if eventType != "" {
		query = query.Where("e.event_type = ?", eventType)
	}

	// Aplicar filtro de data
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
				fmt.Printf("CountEvents: Ajustando horário de início para: %s\n", fromTime.Format("2006-01-02 15:04:05"))
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
				fmt.Printf("CountEvents: Ajustando horário de fim para: %s\n", toTime.Format("2006-01-02 15:04:05"))
			}
		} else {
			// Se não fornecido, usar o fim do dia
			toTime = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 999999999, to.Location())
		}

		// Formatar as datas como strings no formato de timestamp SQL
		fromStr := fromTime.Format("2006-01-02 15:04:05")
		toStr := toTime.Format("2006-01-02 15:04:05")

		// Aplicar filtro usando apenas timezone 'America/Sao_Paulo' sem converter de UTC primeiro
		query = query.Where("(e.event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN ? AND ?",
			fromStr, toStr)

		fmt.Printf("CountEvents: Filtro de data aplicado com timezone: %s até %s\n", fromStr, toStr)
	}

	// Adicionar filtro de profissão se fornecido
	if len(professionIDs) > 0 {
		query = query.Where("e.profession_id IN ?", professionIDs)
	}

	// Adicionar filtro de funil se fornecido
	if len(funnelIDs) > 0 {
		query = query.Where("e.funnel_id IN ?", funnelIDs)
	}

	// Aplicar filtros avançados (reutilizando a mesma lógica usada em GetEvents)
	if len(advancedFilters) > 0 {
		// Definir operador lógico entre filtros (AND/OR)
		if filterCondition == "OR" {
			// TODO: Implementar lógica de filtros OR
			fmt.Println("Filtros OR para CountEvents não implementados completamente")
		}

		// Aplicar cada filtro individualmente como AND por padrão
		for _, filter := range advancedFilters {
			// Seria necessário implementar a lógica completa de filtros aqui
			// Implementar a lógica de filtros avançados similar à GetEvents
			fmt.Printf("Aplicando filtro avançado: %s %s %s\n", filter.Property, filter.Operator, filter.Value)
		}
	}

	// Contar resultados
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

// CountEventsByPeriods conta eventos agrupados por períodos (dias)
func (r *eventRepository) CountEventsByPeriods(periods []string, eventType string, advancedFilters []AdvancedFilter, funnelID int, professionID int) (map[string]int64, error) {
	result := make(map[string]int64)

	// Obter localização de Brasília usando a função centralizada
	brazilLocation := utils.GetBrasilLocation()

	// Processar cada período individualmente
	for _, period := range periods {
		// Parse da data do período
		date, err := time.Parse("2006-01-02", period)
		if err != nil {
			continue
		}

		// Configurar início e fim do dia no horário de Brasília
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, brazilLocation)
		endOfDay := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 999999999, brazilLocation)

		// Inicializar consulta para contagem de eventos no período
		query := r.db.Model(&entities.Event{}).Table("events e")

		// JOIN com users para ter acesso a UTMs
		query = query.
			Joins("JOIN users u ON e.user_id = u.user_id")

		// Aplicar filtro de tipo de evento, se fornecido
		if eventType != "" {
			query = query.Where("e.event_type = ?", eventType)
		}

		// Filtrar por data do período
		query = query.Where("(e.event_time AT TIME ZONE 'America/Sao_Paulo') BETWEEN ? AND ?", startOfDay.Format("2006-01-02 15:04:05"), endOfDay.Format("2006-01-02 15:04:05"))

		// Filtrar por funnel_id se fornecido e maior que 0
		if funnelID > 0 {
			query = query.Where("e.funnel_id = ?", funnelID)
		}

		// Filtrar por profession_id se fornecido e maior que 0
		if professionID > 0 {
			query = query.Where("e.profession_id = ?", professionID)
		}

		// Aplicar filtros avançados se existirem
		if len(advancedFilters) > 0 {
			// Implementar a lógica de filtros avançados similar à CountEvents
		}

		// Contar eventos para este período
		var count int64
		if err := query.Count(&count).Error; err != nil {
			return nil, err
		}

		result[period] = count
	}

	return result, nil
}

// GetEventsDateRange retorna o intervalo de datas (mínima e máxima) dos eventos
func (r *eventRepository) GetEventsDateRange(eventType string) (time.Time, time.Time, error) {
	var minDate, maxDate time.Time

	// Inicializar consulta base
	minQuery := r.db.Model(&entities.Event{}).Table("events e")

	// Aplicar filtro de tipo de evento, se fornecido
	if eventType != "" {
		minQuery = minQuery.Where("e.event_type = ?", eventType)
	}

	// Buscar data mínima usando timezone São Paulo
	var minResult struct {
		MinDate time.Time
	}

	err := minQuery.
		Select("MIN(e.event_time AT TIME ZONE 'America/Sao_Paulo') as min_date").
		Scan(&minResult).Error

	if err != nil {
		return minDate, maxDate, err
	}

	if minResult.MinDate.IsZero() {
		// Nenhum resultado encontrado
		return minDate, maxDate, nil
	}

	minDate = minResult.MinDate

	// Buscar data máxima usando timezone São Paulo
	maxQuery := r.db.Model(&entities.Event{}).Table("events e")

	// Aplicar mesmo filtro de tipo
	if eventType != "" {
		maxQuery = maxQuery.Where("e.event_type = ?", eventType)
	}

	var maxResult struct {
		MaxDate time.Time
	}

	err = maxQuery.
		Select("MAX(e.event_time AT TIME ZONE 'America/Sao_Paulo') as max_date").
		Scan(&maxResult).Error

	if err != nil {
		return minDate, maxDate, err
	}

	maxDate = maxResult.MaxDate

	return minDate, maxDate, nil
}

// CountEventsByDateRange conta eventos em um intervalo de datas (otimizado para dashboard)
func (r *eventRepository) CountEventsByDateRange(
	from, to time.Time,
	timeFrom, timeTo string,
	eventType string,
	professionIDs, funnelIDs []int,
	logicalOperator string,
) (int64, error) {
	var count int64

	// Ajustar datas para incluir o dia inteiro quando é o mesmo dia
	isSameDay := from.Year() == to.Year() && from.Month() == to.Month() && from.Day() == to.Day()

	// Formatar as datas para o formato do PostgreSQL
	var fromStr, toStr string

	if isSameDay {
		// Para o mesmo dia, garantir que o intervalo seja do início ao fim do dia
		fromStr = fmt.Sprintf("%s 00:00:00", from.Format("2006-01-02"))
		toStr = fmt.Sprintf("%s 23:59:59", to.Format("2006-01-02"))
	} else {
		// Diferentes dias, usar as horas especificadas
		// Se timeFrom e timeTo forem fornecidos, usá-los
		fromHour := "00:00:00"
		if timeFrom != "" {
			fromHour = timeFrom + ":00"
		}

		toHour := "23:59:59"
		if timeTo != "" {
			toHour = timeTo + ":59"
		}

		fromStr = fmt.Sprintf("%s %s", from.Format("2006-01-02"), fromHour)
		toStr = fmt.Sprintf("%s %s", to.Format("2006-01-02"), toHour)
	}

	// Criar query base
	query := r.db.Model(&entities.Event{})

	// Aplicar filtro de tipo de evento
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	// Aplicar filtro de data com timezone São Paulo
	if !from.IsZero() && !to.IsZero() {
		// Usar log para debug
		fmt.Printf("Consulta de eventos com intervalo: %s até %s, tipo: %s\n", fromStr, toStr, eventType)

		query = query.Where("(\"event_time\" AT TIME ZONE 'America/Sao_Paulo') BETWEEN ? AND ?",
			fromStr, toStr)
	}

	// Aplicar os filtros de profissão e funil
	if len(professionIDs) > 0 || len(funnelIDs) > 0 {
		// Se ambos os filtros estiverem presentes, aplicar o operador lógico (AND/OR)
		if len(professionIDs) > 0 && len(funnelIDs) > 0 {
			if logicalOperator == "OR" {
				// OR: pelo menos um dos filtros deve corresponder
				query = query.Where("profession_id IN ? OR funnel_id IN ?", professionIDs, funnelIDs)
			} else {
				// AND: ambos os filtros devem corresponder (padrão)
				query = query.Where("profession_id IN ? AND funnel_id IN ?", professionIDs, funnelIDs)
			}
		} else if len(professionIDs) > 0 {
			// Somente filtro de profissão
			query = query.Where("profession_id IN ?", professionIDs)
		} else {
			// Somente filtro de funil
			query = query.Where("funnel_id IN ?", funnelIDs)
		}
	}

	// Obter SQL para debug
	stmt := query.Statement
	sql := stmt.SQL.String()
	vars := stmt.Vars
	fmt.Printf("SQL para CountEventsByDateRange: %s, Vars: %v\n", sql, vars)

	// Executar consulta de contagem diretamente
	err := query.Count(&count).Error

	// Log para debug
	fmt.Printf("Contagem de eventos para %s - %s, tipo %s: %d\n", fromStr, toStr, eventType, count)

	return count, err
}

// GetEventsByHours retorna a contagem de eventos por hora para um dia específico
func (r *eventRepository) GetEventsByHours(date time.Time, eventType, userID, professionID, productID, funnelID string, landingPage string) (map[string]int64, error) {
	result := make(map[string]int64)

	// Garantir que estamos trabalhando com a data sem componente de hora
	day := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	// Definir início e fim do dia
	startOfDay := fmt.Sprintf("%s 00:00:00", day.Format("2006-01-02"))
	endOfDay := fmt.Sprintf("%s 23:59:59", day.Format("2006-01-02"))

	// Definir a string de formato de hora para a consulta SQL
	// Para PostgreSQL, usamos to_char com timezone São Paulo para extrair a hora
	hourFormat := "to_char((\"event_time\" AT TIME ZONE 'America/Sao_Paulo'), 'HH24')"

	// Criar estrutura temporária para receber os resultados
	type HourlyCount struct {
		Hour  string
		Count int64
	}
	var counts []HourlyCount

	// Criar query base
	query := r.db.Model(&entities.Event{}).
		Select(hourFormat + " as hour, count(*) as count")

	// Filtrar por tipo de evento se fornecido
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}

	// Filtrar pelo intervalo de data
	query = query.Where("(\"event_time\" AT TIME ZONE 'America/Sao_Paulo') BETWEEN ? AND ?",
		startOfDay, endOfDay)

	// Aplicar outros filtros, se fornecidos
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Aplicar filtro por profissão
	if professionID != "" {
		query = query.Where("profession_id = ?", professionID)
	}

	// Aplicar filtro por produto
	if productID != "" {
		query = query.Where("product_id = ?", productID)
	}

	// Aplicar filtro por funil
	if funnelID != "" {
		query = query.Where("funnel_id = ?", funnelID)
	}

	// Aplicar filtro por landing page
	if landingPage != "" {
		query = query.Where("\"landingPage\" = ?", landingPage)
	}

	// Agrupar e ordenar por hora
	query = query.Group("hour").Order("hour")

	// Obter SQL para debug
	stmt := query.Statement
	sql := stmt.SQL.String()
	vars := stmt.Vars
	fmt.Printf("SQL para GetEventsByHours: %s, Vars: %v\n", sql, vars)

	// Executar a consulta
	err := query.Find(&counts).Error
	if err != nil {
		return result, err
	}

	// Converter o resultado para o formato de mapa
	for _, c := range counts {
		result[c.Hour] = c.Count
	}

	// Preencher horas que não retornaram dados com zero
	for hour := 0; hour < 24; hour++ {
		hourStr := fmt.Sprintf("%02d", hour)
		if _, exists := result[hourStr]; !exists {
			result[hourStr] = 0
		}
	}

	return result, nil
}
