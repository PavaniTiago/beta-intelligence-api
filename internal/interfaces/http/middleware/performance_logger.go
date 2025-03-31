package middleware

import (
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// PerformanceLogger é um middleware que mede o tempo de resposta das rotas críticas
func PerformanceLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Verificar se é uma rota que queremos monitorar
		path := c.Path()

		// Lista de rotas para monitorar performance
		monitoredRoutes := []string{
			"/session",
			"/lead",
		}

		shouldMonitor := false
		for _, route := range monitoredRoutes {
			if strings.HasPrefix(path, route) {
				shouldMonitor = true
				break
			}
		}

		if shouldMonitor {
			// Registrar o tempo de início
			start := time.Now()

			// Processar a requisição
			err := c.Next()

			// Calcular duração
			duration := time.Since(start)

			// Registrar informações de performance
			log.Printf(
				"[PERFORMANCE] %s %s - %d - Duration: %v - Query params: %s",
				c.Method(),
				path,
				c.Response().StatusCode(),
				duration,
				c.Request().URI().QueryArgs().String(),
			)

			return err
		}

		// Se não for uma rota monitorada, apenas continua
		return c.Next()
	}
}
