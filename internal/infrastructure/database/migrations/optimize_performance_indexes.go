package migrations

import (
	"log"

	"gorm.io/gorm"
)

// OptimizePerformanceIndexes adiciona índices otimizados para melhorar o desempenho das consultas
func OptimizePerformanceIndexes(db *gorm.DB) error {
	log.Println("Adicionando índices de performance otimizados...")

	// Índices para tabela users

	// Índice composto para consultas em leads (isIdentified=true, isClient=false)
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_leads ON users ("isIdentified", "isClient", created_at)`).Error; err != nil {
		return err
	}

	// Índice para consultas em clientes (isClient=true)
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_clients ON users ("isClient", created_at)`).Error; err != nil {
		return err
	}

	// Índice BRIN para consultas por período em users (mais eficiente para grandes volumes de data sequencial)
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_users_created_at_brin ON users USING BRIN (created_at)`).Error; err != nil {
		return err
	}

	// Índices para tabela sessions

	// Índice para consultas de sessão por período
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_start_active ON sessions ("sessionStart", "isActive")`).Error; err != nil {
		return err
	}

	// Índice para relacionamentos de sessão (melhorar joins)
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_relations ON sessions (user_id, profession_id, product_id, funnel_id)`).Error; err != nil {
		return err
	}

	// Índice BRIN para consultas por período em sessions
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_date_brin ON sessions USING BRIN ("sessionStart")`).Error; err != nil {
		return err
	}

	// Índice para consultas de sessões ativas
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions ("isActive") WHERE "isActive" = true`).Error; err != nil {
		return err
	}

	// Adicionar índices para otimização da consulta de conversão por profissão
	indexes := []string{
		// Índices para a rota de conversão por profissão
		"CREATE INDEX IF NOT EXISTS idx_sessions_profession_started_landing ON sessions (profession_id, \"sessionStart\", \"landingPage\")",
		"CREATE INDEX IF NOT EXISTS idx_events_profession_time_type ON events (profession_id, event_time, event_type)",
	}

	// Executar cada índice
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			return err
		}
	}

	log.Println("Índices de performance criados com sucesso!")
	return nil
}
