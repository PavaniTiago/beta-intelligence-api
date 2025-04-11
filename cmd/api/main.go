package main

import (
	"log"
	"os"
	"time"

	"github.com/PavaniTiago/beta-intelligence-api/internal/infrastructure/database"
	"github.com/PavaniTiago/beta-intelligence-api/internal/interfaces/http/middleware"
	"github.com/PavaniTiago/beta-intelligence-api/internal/interfaces/http/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Definir localização padrão para Brasília (UTC-3)
	brasilLocation, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Fallback para UTC-3 se não conseguir carregar a localização
		brasilLocation = time.FixedZone("BRT", -3*60*60)
		log.Printf("⚠️ Usando timezone fixo BRT (UTC-3): %v", err)
	} else {
		log.Printf("🕒 Timezone configurado para America/Sao_Paulo (Brasília)")
	}
	// Configurar timezone padrão globalmente
	time.Local = brasilLocation

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found, using system environment variables")
	}

	// Initialize database
	db, err := database.SetupDatabase()
	if err != nil {
		log.Fatalf("❌ Error setting up database: %v", err)
	}

	// Executar a contagem de sessões
	log.Println("📊 Contando sessões...")

	// Configure Fiber for better performance
	app := fiber.New(fiber.Config{
		// Increase concurrency for better performance
		Concurrency: 256 * 1024,
		// Desabilitado modo Prefork pois causa instabilidade no container
		Prefork: false,
		// Set reasonable body limit
		BodyLimit: 10 * 1024 * 1024, // 10MB
		// Configure server for better performance - aumentados timeouts para resolver 504 Gateway Timeout
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  240 * time.Second,
	})

	// Setup middleware
	middleware.SetupMiddlewares(app)

	// Setup routes
	routes.SetupRoutes(app, db)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
