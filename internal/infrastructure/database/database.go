package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/infrastructure/database/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupDatabase() (*gorm.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is not defined in the environment")
	}

	// Configure GORM with performance optimizations
	config := &gorm.Config{
		// Skip default transaction for better performance
		SkipDefaultTransaction: true,
		// Prepare statements for better performance
		PrepareStmt: true,
		// Configure logger to reduce overhead
		Logger: logger.Default.LogMode(logger.Error),
	}

	db, err := gorm.Open(postgres.Open(dbURL), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Configure connection pool for better performance
	sqlDB.SetMaxIdleConns(20)           // Increased from 10
	sqlDB.SetMaxOpenConns(150)          // Increased from 100
	sqlDB.SetConnMaxLifetime(time.Hour) // Reuse connections for up to an hour

	// Configurar timezone padrão para todas as conexões - configuração inicial
	if err := db.Exec("SET timezone = 'America/Sao_Paulo'").Error; err != nil {
		log.Printf("⚠️ Aviso: não foi possível configurar timezone do banco: %v", err)
	} else {
		log.Printf("🕒 Timezone do banco configurado para America/Sao_Paulo (Brasília)")
	}

	// Registrar middlewares para garantir que o timezone seja definido em todas as consultas
	// O middleware será aplicado apenas quando necessário e com proteção contra recursão
	RegisterMiddlewares(db)

	// Apply database migrations and indexes
	if err := migrations.Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Add indexes for better query performance
	if err := migrations.AddIndexes(db); err != nil {
		return nil, fmt.Errorf("failed to add indexes: %w", err)
	}

	// Add optimized performance indexes
	if err := migrations.OptimizePerformanceIndexes(db); err != nil {
		return nil, fmt.Errorf("failed to add optimized indexes: %w", err)
	}

	return db, nil
}
