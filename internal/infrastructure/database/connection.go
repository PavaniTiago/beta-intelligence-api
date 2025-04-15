package database

import (
	"context"

	"gorm.io/gorm"
)

// Chave para o contexto que indica se o timezone já foi configurado
type timezoneKey struct{}

// SetTimezoneMiddleware cria um middleware GORM para definir o timezone
func SetTimezoneMiddleware() func(db *gorm.DB) {
	return func(db *gorm.DB) {
		// Verificar se já está processando uma configuração de timezone
		if _, ok := db.Statement.Context.Value(timezoneKey{}).(bool); ok {
			return // Evita recursão infinita
		}

		// Define um contexto marcado para evitar recursão
		ctx := context.WithValue(db.Statement.Context, timezoneKey{}, true)

		// Executa a configuração de timezone com o contexto marcado
		tx := db.WithContext(ctx)
		tx.Exec("SET timezone = 'America/Sao_Paulo'")
	}
}

// RegisterMiddlewares registra todos os middlewares necessários no GORM
func RegisterMiddlewares(db *gorm.DB) {
	// Adicionar middleware apenas no callback de consulta query
	// Removendo callbacks desnecessários para evitar overhead
	db.Callback().Query().Before("gorm:query").Register("set_timezone_before_query", SetTimezoneMiddleware())
}
