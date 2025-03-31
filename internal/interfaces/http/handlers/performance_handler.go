package handlers

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/PavaniTiago/beta-intelligence-api/internal/domain/entities"
)

type PerformanceHandler struct {
	db *gorm.DB
}

func NewPerformanceHandler(db *gorm.DB) *PerformanceHandler {
	return &PerformanceHandler{
		db: db,
	}
}

// TestLeadPerformance executa testes comparativos entre a versão otimizada e não otimizada
func (h *PerformanceHandler) TestLeadPerformance(c *fiber.Ctx) error {
	// Parâmetros de teste
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	fromDate := c.Query("from", "")
	toDate := c.Query("to", "")

	// Valores padrão para datas
	from := time.Now().AddDate(0, -1, 0) // 1 mês atrás
	to := time.Now()

	// Parse de datas
	if fromDate != "" {
		parsedFrom, err := time.Parse(time.RFC3339, fromDate)
		if err == nil {
			from = parsedFrom
		}
	}

	if toDate != "" {
		parsedTo, err := time.Parse(time.RFC3339, toDate)
		if err == nil {
			to = parsedTo
		}
	}

	// Teste não otimizado (seleciona tudo)
	startNonOptimized := time.Now()
	var leadsNonOpt []entities.User
	var totalNonOpt int64

	// Query não otimizada (sem seleção específica, sem índices especiais)
	queryNonOpt := h.db.Model(&entities.User{}).Where(`"isIdentified" = ? AND "isClient" = ?`, true, false)
	if !from.IsZero() && !to.IsZero() {
		queryNonOpt = queryNonOpt.Where("created_at BETWEEN ? AND ?", from, to)
	}

	countQueryNonOpt := queryNonOpt
	countQueryNonOpt.Count(&totalNonOpt)
	queryNonOpt.Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&leadsNonOpt)

	durationNonOpt := time.Since(startNonOptimized)

	// Teste otimizado (seleciona campos específicos, usa índices)
	startOptimized := time.Now()
	var leadsOpt []entities.User
	var totalOpt int64

	// Query otimizada (com seleção específica e índices)
	selectFields := []string{
		"user_id", "fullname", "email", "phone",
		"\"isClient\"", "\"isIdentified\"", "created_at",
		"initial_country", "initial_city",
	}

	queryOpt := h.db.Model(&entities.User{}).Select(selectFields).Where(`"isIdentified" = ? AND "isClient" = ?`, true, false)
	if !from.IsZero() && !to.IsZero() {
		queryOpt = queryOpt.Where("created_at BETWEEN ? AND ?", from, to)
	}

	// Usar sessão separada para contagem
	countQueryOpt := queryOpt.Session(&gorm.Session{})
	countQueryOpt.Count(&totalOpt)

	// Aplicar ordem e paginação
	queryOpt.Offset((page - 1) * limit).Limit(limit).Order("created_at DESC").Find(&leadsOpt)

	durationOpt := time.Since(startOptimized)

	// Calcular a melhoria
	improvement := float64(durationNonOpt) / float64(durationOpt)

	return c.JSON(fiber.Map{
		"non_optimized": fiber.Map{
			"duration_ms": durationNonOpt.Milliseconds(),
			"count":       totalNonOpt,
			"results":     len(leadsNonOpt),
		},
		"optimized": fiber.Map{
			"duration_ms": durationOpt.Milliseconds(),
			"count":       totalOpt,
			"results":     len(leadsOpt),
		},
		"improvement_factor": fmt.Sprintf("%.2fx mais rápido", improvement),
		"params": fiber.Map{
			"page":  page,
			"limit": limit,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
	})
}

// TestSessionPerformance executa testes comparativos entre a versão otimizada e não otimizada
func (h *PerformanceHandler) TestSessionPerformance(c *fiber.Ctx) error {
	// Parâmetros de teste
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	fromDate := c.Query("from", "")
	toDate := c.Query("to", "")

	// Valores padrão para datas
	from := time.Now().AddDate(0, -1, 0) // 1 mês atrás
	to := time.Now()

	// Parse de datas
	if fromDate != "" {
		parsedFrom, err := time.Parse(time.RFC3339, fromDate)
		if err == nil {
			from = parsedFrom
		}
	}

	if toDate != "" {
		parsedTo, err := time.Parse(time.RFC3339, toDate)
		if err == nil {
			to = parsedTo
		}
	}

	// Teste não otimizado
	startNonOptimized := time.Now()
	var sessionsNonOpt []entities.Session
	var totalNonOpt int64

	// Query não otimizada (sem seleção específica)
	queryNonOpt := h.db.Model(&entities.Session{})
	if !from.IsZero() && !to.IsZero() {
		queryNonOpt = queryNonOpt.Where(`"sessionStart" BETWEEN ? AND ?`, from, to)
	}

	// Contagem e busca em queries separadas
	countQueryNonOpt := queryNonOpt
	countQueryNonOpt.Count(&totalNonOpt)

	// Aplicar paginação e ordenação
	queryNonOpt.Preload("User").Preload("Profession").Preload("Product").Preload("Funnel").
		Offset((page - 1) * limit).Limit(limit).
		Order(`"sessionStart" DESC`).Find(&sessionsNonOpt)

	durationNonOpt := time.Since(startNonOptimized)

	// Teste otimizado
	startOptimized := time.Now()
	var sessionsOpt []entities.Session
	var totalOpt int64

	// Campos específicos para seleção
	selectFields := []string{
		"session_id", "user_id", "\"sessionStart\"", "\"isActive\"",
		"\"lastActivity\"", "country", "profession_id", "product_id", "funnel_id",
	}

	// Query otimizada (com seleção específica)
	queryOpt := h.db.Model(&entities.Session{}).Select(selectFields)
	if !from.IsZero() && !to.IsZero() {
		queryOpt = queryOpt.Where(`"sessionStart" BETWEEN ? AND ?`, from, to)
	}

	// Usar sessão separada para contagem
	countQueryOpt := queryOpt.Session(&gorm.Session{})
	countQueryOpt.Count(&totalOpt)

	// Buscar sessões
	queryOpt.Offset((page - 1) * limit).Limit(limit).
		Order(`"sessionStart" DESC`).Find(&sessionsOpt)

	// Se houver resultados, buscar relações de forma otimizada
	if len(sessionsOpt) > 0 {
		var sessionIDs []uuid.UUID
		for _, s := range sessionsOpt {
			sessionIDs = append(sessionIDs, s.ID)
		}

		// Preload otimizado para cada relação
		h.db.Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("user_id, fullname, email")
		}).Preload("Profession", func(db *gorm.DB) *gorm.DB {
			return db.Select("profession_id, name")
		}).Preload("Product", func(db *gorm.DB) *gorm.DB {
			return db.Select("product_id, name")
		}).Preload("Funnel", func(db *gorm.DB) *gorm.DB {
			return db.Select("funnel_id, name")
		}).Where("session_id IN ?", sessionIDs).Find(&sessionsOpt)
	}

	durationOpt := time.Since(startOptimized)

	// Calcular a melhoria
	improvement := float64(durationNonOpt) / float64(durationOpt)

	return c.JSON(fiber.Map{
		"non_optimized": fiber.Map{
			"duration_ms": durationNonOpt.Milliseconds(),
			"count":       totalNonOpt,
			"results":     len(sessionsNonOpt),
		},
		"optimized": fiber.Map{
			"duration_ms": durationOpt.Milliseconds(),
			"count":       totalOpt,
			"results":     len(sessionsOpt),
		},
		"improvement_factor": fmt.Sprintf("%.2fx mais rápido", improvement),
		"params": fiber.Map{
			"page":  page,
			"limit": limit,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		},
	})
}
