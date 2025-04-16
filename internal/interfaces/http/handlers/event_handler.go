package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/application/usecases"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/repositories"
	"github.com/gofiber/fiber/v2"
)

type EventHandler struct {
	eventUseCase usecases.EventUseCase
}

func NewEventHandler(eventUseCase usecases.EventUseCase) *EventHandler {
	return &EventHandler{eventUseCase}
}

func (h *EventHandler) GetEvents(c *fiber.Ctx) error {
	// Get query parameters
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	// Verificar se é para retornar apenas a contagem
	countOnly := c.Query("count_only", "false") == "true"

	// Verificar se há busca por tipo específico de evento
	eventType := c.Query("event_type", "")

	// Get sort parameters
	sortBy := c.Query("sortBy", "event_time")
	sortDirection := c.Query("sortDirection", "desc")

	// Obter todos os parâmetros da query usando QueryParams que suporta múltiplos valores
	queryParams := c.Context().QueryArgs()
	fmt.Printf("All query parameters: %v\n", queryParams)

	// Get profession filter - usando uma abordagem que captura todos os valores
	var professionIDs []int

	// Capturar todos os valores de profession_ids[]
	profIDsArray := queryParams.PeekMulti("profession_ids[]")
	fmt.Printf("profession_ids[] values: %v\n", profIDsArray)
	for _, idBytes := range profIDsArray {
		if id, err := strconv.Atoi(string(idBytes)); err == nil {
			professionIDs = append(professionIDs, id)
			fmt.Printf("Added profession_id from array format: %d\n", id)
		}
	}

	// Capturar valores de profession_ids (sem colchetes)
	if profIDsBytes := queryParams.Peek("profession_ids"); profIDsBytes != nil {
		profIDsStr := string(profIDsBytes)
		fmt.Printf("profession_ids value: %s\n", profIDsStr)

		// Decodificar manualmente o valor URL-encoded
		profIDsStr = strings.ReplaceAll(profIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		profIDsArr := strings.Split(profIDsStr, ",")
		for _, idStr := range profIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				professionIDs = append(professionIDs, id)
				fmt.Printf("Added profession_id from plural format: %d\n", id)
			}
		}
	}

	// Capturar valores de profession_id (singular)
	if profIDsBytes := queryParams.Peek("profession_id"); profIDsBytes != nil {
		profIDsStr := string(profIDsBytes)
		fmt.Printf("profession_id value: %s\n", profIDsStr)

		// Decodificar manualmente o valor URL-encoded
		profIDsStr = strings.ReplaceAll(profIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		profIDsArr := strings.Split(profIDsStr, ",")
		for _, idStr := range profIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				professionIDs = append(professionIDs, id)
				fmt.Printf("Added profession_id from singular format: %d\n", id)
			}
		}
	}

	fmt.Printf("Final profession_ids for filtering: %v\n", professionIDs)

	// Get funnel filter - usando a mesma abordagem
	var funnelIDs []int

	// Capturar todos os valores de funnel_ids[]
	funnelIDsArray := queryParams.PeekMulti("funnel_ids[]")
	fmt.Printf("funnel_ids[] values: %v\n", funnelIDsArray)
	for _, idBytes := range funnelIDsArray {
		if id, err := strconv.Atoi(string(idBytes)); err == nil {
			funnelIDs = append(funnelIDs, id)
			fmt.Printf("Added funnel_id from array format: %d\n", id)
		}
	}

	// Capturar valores de funnel_ids (sem colchetes)
	if funnelIDsBytes := queryParams.Peek("funnel_ids"); funnelIDsBytes != nil {
		funnelIDsStr := string(funnelIDsBytes)
		fmt.Printf("funnel_ids value: %s\n", funnelIDsStr)

		// Decodificar manualmente o valor URL-encoded
		funnelIDsStr = strings.ReplaceAll(funnelIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		funnelIDsArr := strings.Split(funnelIDsStr, ",")
		for _, idStr := range funnelIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				funnelIDs = append(funnelIDs, id)
				fmt.Printf("Added funnel_id from plural format: %d\n", id)
			}
		}
	}

	// Capturar valores de funnel_id (singular)
	if funnelIDsBytes := queryParams.Peek("funnel_id"); funnelIDsBytes != nil {
		funnelIDsStr := string(funnelIDsBytes)
		fmt.Printf("funnel_id value: %s\n", funnelIDsStr)

		// Decodificar manualmente o valor URL-encoded
		funnelIDsStr = strings.ReplaceAll(funnelIDsStr, "%2C", ",")

		// Split por vírgula para obter múltiplos IDs
		funnelIDsArr := strings.Split(funnelIDsStr, ",")
		for _, idStr := range funnelIDsArr {
			if id, err := strconv.Atoi(strings.TrimSpace(idStr)); err == nil {
				funnelIDs = append(funnelIDs, id)
				fmt.Printf("Added funnel_id from singular format: %d\n", id)
			}
		}
	}

	fmt.Printf("Final funnel_ids for filtering: %v\n", funnelIDs)

	// Validate sort direction
	if sortDirection != "asc" && sortDirection != "desc" {
		sortDirection = "desc"
	}

	// Validate sortBy field and build orderBy
	validSortFields := map[string]string{
		// Event fields
		"event_id":     "e.event_id",
		"event_name":   "e.event_name",
		"pageview_id":  "e.pageview_id",
		"session_id":   "e.session_id",
		"event_time":   "e.event_time",
		"event_source": "e.event_source",
		"event_type":   "e.event_type",

		// User fields
		"fullname":  "u.fullname",
		"email":     "u.email",
		"phone":     "u.phone",
		"is_client": "u.isClient",

		// UTM fields (agora vindo da tabela users)
		"utm_source":   "u.\"initialUtmSource\"",
		"utm_medium":   "u.\"initialUtmMedium\"",
		"utm_campaign": "u.\"initialUtmCampaign\"",
		"utm_content":  "u.\"initialUtmContent\"",
		"utm_term":     "u.\"initialUtmTerm\"",

		// Session fields
		"country": "u.\"initialCountry\"",
		"state":   "u.\"initialRegion\"",
		"city":    "u.\"initialCity\"",
		"ip":      "u.\"initialIp\"",

		// Profession fields
		"profession_name": "professions.profession_name",
		"meta_pixel":      "professions.meta_pixel",
		"meta_token":      "professions.meta_token",

		// Product fields
		"product_name": "products.product_name",

		// Funnel fields
		"funnel_name": "funnels.funnel_name",
		"funnel_tag":  "funnels.funnel_tag",
		"global":      "funnels.global",
	}

	orderBy := "e.event_time desc" // default ordering
	if field, ok := validSortFields[sortBy]; ok {
		orderBy = field + " " + sortDirection
	}

	// Parse date parameters
	from := c.Query("from", "")
	to := c.Query("to", "")

	var fromTime, toTime time.Time
	var err error

	// Definir localização de Brasília (UTC-3)
	brazilLocation := GetBrasilLocation()

	// Tenta primeiro o formato com hora (ISO 8601)
	if from != "" {
		// Tenta primeiro o formato com hora
		fromTime, err = time.Parse(time.RFC3339, from)
		if err != nil {
			// Se falhar, tenta o formato apenas com data
			fromTime, err = time.Parse("2006-01-02", from)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid from date format. Use YYYY-MM-DD or YYYY-MM-DDThh:mm:ssZ",
				})
			}
			// Se for apenas data, define para o início do dia em horário de Brasília
			fromTime = time.Date(fromTime.Year(), fromTime.Month(), fromTime.Day(), 0, 0, 0, 0, brazilLocation)
		} else {
			// Se tiver timezone, converte para Brasília
			fromTime = fromTime.In(brazilLocation)
		}
	} else {
		// If no from date, use 30 days ago as default - definir para Brasília
		now := time.Now().In(brazilLocation)
		fromTime = now.AddDate(0, 0, -30)
	}

	if to != "" {
		// Tenta primeiro o formato com hora
		toTime, err = time.Parse(time.RFC3339, to)
		if err != nil {
			// Se falhar, tenta o formato apenas com data
			toTime, err = time.Parse("2006-01-02", to)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid to date format. Use YYYY-MM-DD or YYYY-MM-DDThh:mm:ssZ",
				})
			}
			// Se for apenas data, define para o final do dia em horário de Brasília
			toTime = time.Date(toTime.Year(), toTime.Month(), toTime.Day(), 23, 59, 59, 999999999, brazilLocation)
		} else {
			// Se tiver timezone, converte para Brasília
			toTime = toTime.In(brazilLocation)
		}
	} else {
		// If no to date, use current time in Brasília
		toTime = time.Now().In(brazilLocation)
	}

	// Garantir que "from" seja sempre anterior a "to"
	if fromTime.After(toTime) {
		fromTime, toTime = toTime, fromTime
	}

	// Certificar-se de trabalhar com horário de Brasília
	fromTime = fromTime.In(brazilLocation)
	toTime = toTime.In(brazilLocation)

	// Após processar os parâmetros de data
	fmt.Printf("From time: %v (Brasília: %v)\n", fromTime, fromTime.In(brazilLocation))
	fmt.Printf("To time: %v (Brasília: %v)\n", toTime, toTime.In(brazilLocation))

	// Processar filtros avançados
	var advancedFilters []repositories.AdvancedFilter

	// Obter condição de filtro (AND/OR)
	filterCondition := c.Query("filter_condition", "AND")
	if filterCondition != "AND" && filterCondition != "OR" {
		filterCondition = "AND" // valor padrão
	}

	// Obter e processar os filtros avançados
	advancedFiltersStr := c.Query("advanced_filters", "")
	if advancedFiltersStr != "" {
		fmt.Printf("DEBUG: Filtros avançados recebidos (raw): %s\n", advancedFiltersStr)

		// Decodificar o JSON
		err := json.Unmarshal([]byte(advancedFiltersStr), &advancedFilters)
		if err != nil {
			fmt.Printf("Error parsing advanced filters: %v\n", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid advanced_filters format. Expected JSON array.",
			})
		}

		fmt.Printf("DEBUG: Filtros avançados decodificados: %+v\n", advancedFilters)
		for i, filter := range advancedFilters {
			fmt.Printf("DEBUG: Filtro #%d: property=%s, operator=%s, value=%s\n",
				i+1, filter.Property, filter.Operator, filter.Value)
		}

		// IMPORTANTE: Verificar se há filtros usando session.utm_*
		for _, filter := range advancedFilters {
			if strings.HasPrefix(filter.Property, "session.utm_") {
				fmt.Printf("AVISO: Filtro usando session.utm_* (%s) não é mais suportado. Use user.utm_* em vez disso.\n",
					filter.Property)
			}
		}
	} else {
		fmt.Printf("DEBUG: Nenhum filtro avançado fornecido\n")
	}

	// Após processar os parâmetros de data
	fmt.Printf("From time: %v (Brasília: %v)\n", fromTime, fromTime.In(brazilLocation))
	fmt.Printf("To time: %v (Brasília: %v)\n", toTime, toTime.In(brazilLocation))
	fmt.Printf("Advanced filters: %+v\n", advancedFilters)
	fmt.Printf("Filter condition: %s\n", filterCondition)

	// Como removemos a implementação de filtros por horário, vamos passar valores vazios para esses parâmetros
	timeFrom := ""
	timeTo := ""

	// Imprimir em log os parâmetros da requisição para debug
	fmt.Printf("Parâmetros da requisição: page=%d, limit=%d, sortBy=%s, advancedFilters=%+v\n",
		page, limit, sortBy, advancedFilters)

	// Verificar período e all_data
	period := c.Query("period", "false") == "true"
	allData := c.Query("all_data", "false") == "true"

	// Se count_only for true, tratar contagem
	if countOnly {
		// Se for para buscar leads por tipo de evento
		if eventType == "LEAD" {
			// Tratamento específico para dashboard com períodos
			periodsParam := c.Query("periods", "")

			// Verificar se é para buscar todos os dados históricos
			if allData {
				fmt.Println("Buscando todos os LEADs históricos")
				// Implementar lógica para buscar intervalo de datas completo de eventos LEAD
				// Gerar array de todas as datas no intervalo
				firstDate, lastDate, err := h.eventUseCase.GetEventsDateRange(eventType)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fmt.Sprintf("Erro ao obter intervalo de datas de LEADs: %v", err),
					})
				}

				// Normalizar as datas para formato de data apenas (sem horas)
				firstDateOnly := time.Date(firstDate.Year(), firstDate.Month(), firstDate.Day(), 0, 0, 0, 0, firstDate.Location())
				lastDateOnly := time.Date(lastDate.Year(), lastDate.Month(), lastDate.Day(), 0, 0, 0, 0, lastDate.Location())

				// Pegar o funnel_id, se existir
				var funnelID int
				if len(funnelIDs) > 0 {
					funnelID = funnelIDs[0]
				}

				// Pegar o profession_id, se existir
				var professionID int
				if len(professionIDs) > 0 {
					professionID = professionIDs[0]
				}

				// Gerar array de todas as datas no intervalo
				dateRange := GenerateDateRange(firstDateOnly, lastDateOnly)
				result, err := h.eventUseCase.CountEventsByPeriods(dateRange, eventType, advancedFilters, funnelID, professionID)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fmt.Sprintf("Erro ao contar LEADs por períodos: %v", err),
					})
				}

				return c.JSON(fiber.Map{
					"periods":       result,
					"start_date":    firstDateOnly.Format("2006-01-02"),
					"end_date":      lastDateOnly.Format("2006-01-02"),
					"all_data":      true,
					"funnel_id":     funnelID,
					"profession_id": professionID,
				})
			} else if period && (!fromTime.IsZero() && !toTime.IsZero()) {
				fmt.Println("Buscando LEADs por período específico")
				// Pegar o funnel_id, se existir
				var funnelID int
				if len(funnelIDs) > 0 {
					funnelID = funnelIDs[0]
				}

				// Pegar o profession_id, se existir
				var professionID int
				if len(professionIDs) > 0 {
					professionID = professionIDs[0]
				}

				// Gerar array de datas no intervalo from-to
				dateRange := GenerateDateRange(fromTime, toTime)
				result, err := h.eventUseCase.CountEventsByPeriods(dateRange, eventType, advancedFilters, funnelID, professionID)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fmt.Sprintf("Erro ao contar LEADs por períodos: %v", err),
					})
				}

				return c.JSON(fiber.Map{
					"periods":       result,
					"from":          fromTime.Format(time.RFC3339),
					"to":            toTime.Format(time.RFC3339),
					"funnel_id":     funnelID,
					"profession_id": professionID,
				})
			} else if periodsParam != "" {
				fmt.Println("Buscando LEADs por períodos específicos")
				// Pegar o funnel_id, se existir
				var funnelID int
				if len(funnelIDs) > 0 {
					funnelID = funnelIDs[0]
				}

				// Pegar o profession_id, se existir
				var professionID int
				if len(professionIDs) > 0 {
					professionID = professionIDs[0]
				}

				periods := strings.Split(periodsParam, ",")
				result, err := h.eventUseCase.CountEventsByPeriods(periods, eventType, advancedFilters, funnelID, professionID)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": fmt.Sprintf("Erro ao contar LEADs por períodos: %v", err),
					})
				}

				return c.JSON(fiber.Map{
					"periods":       result,
					"funnel_id":     funnelID,
					"profession_id": professionID,
				})
			}

			// Contagem normal de LEADs
			count, err := h.eventUseCase.CountEvents(fromTime, toTime, timeFrom, timeTo, eventType, professionIDs, funnelIDs, advancedFilters, filterCondition)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": fmt.Sprintf("Erro ao contar LEADs: %v", err),
				})
			}

			return c.JSON(fiber.Map{
				"count":     count,
				"from":      fromTime.Format(time.RFC3339),
				"to":        toTime.Format(time.RFC3339),
				"time_from": timeFrom,
				"time_to":   timeTo,
			})
		}

		// Para outros tipos de contagem de eventos
		count, err := h.eventUseCase.CountEvents(fromTime, toTime, timeFrom, timeTo, eventType, professionIDs, funnelIDs, advancedFilters, filterCondition)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Erro ao contar eventos: %v", err),
			})
		}

		return c.JSON(fiber.Map{
			"count": count,
			"from":  fromTime.Format(time.RFC3339),
			"to":    toTime.Format(time.RFC3339),
		})
	}

	// Código existente para buscar eventos quando não é count_only
	events, total, err := h.eventUseCase.GetEvents(c.Context(), page, limit, orderBy, fromTime, toTime, timeFrom, timeTo, professionIDs, funnelIDs, advancedFilters, filterCondition)
	if err != nil {
		fmt.Printf("Error fetching events: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Verificar e imprimir informações sobre os eventos retornados
	if len(events) > 0 {
		fmt.Printf("Total de eventos retornados: %d\n", len(events))
		fmt.Printf("Primeiro evento: ID=%s, Type=%s\n", events[0].EventID, events[0].EventType)
		fmt.Printf("UTM data do primeiro evento: %+v\n", events[0].UtmData)

		// Verificar explicitamente se utm_data está presente
		if events[0].UtmData == nil {
			fmt.Printf("AVISO: utm_data é nil para o primeiro evento!\n")
			// Garantir que utm_data nunca seja nil
			for i := range events {
				if events[i].UtmData == nil {
					events[i].UtmData = &entities.UtmData{
						UtmSource:   events[i].User.InitialUtmSource,
						UtmMedium:   events[i].User.InitialUtmMedium,
						UtmCampaign: events[i].User.InitialUtmCampaign,
						UtmContent:  events[i].User.InitialUtmContent,
						UtmTerm:     events[i].User.InitialUtmTerm,
					}
				}
			}
		}
	} else {
		fmt.Printf("Nenhum evento retornado!\n")
	}

	return c.JSON(fiber.Map{
		"events": events,
		"meta": fiber.Map{
			"total":             total,
			"page":              page,
			"limit":             limit,
			"last_page":         (total + int64(limit) - 1) / int64(limit),
			"from":              fromTime.Format(time.RFC3339),
			"to":                toTime.Format(time.RFC3339),
			"sort_by":           sortBy,
			"sort_direction":    sortDirection,
			"profession_ids":    professionIDs,
			"funnel_ids":        funnelIDs,
			"filter_condition":  filterCondition,
			"advanced_filters":  advancedFilters,
			"valid_sort_fields": getKeys(validSortFields),
			"timezone":          "America/Sao_Paulo", // Adicionar informação sobre o timezone
		},
	})
}

// Helper function to get map keys
func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
