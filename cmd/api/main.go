package main

import (
	"log"
	"os"

	"github.com/PavaniTiago/beta-intelligence-api/internal/infrastructure/database"
	"github.com/PavaniTiago/beta-intelligence-api/internal/interfaces/http/middleware"
	"github.com/PavaniTiago/beta-intelligence-api/internal/interfaces/http/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found, using system environment variables")
	}

	// Initialize database
	db, err := database.SetupDatabase()
	if err != nil {
		log.Fatalf("❌ Error setting up database: %v", err)
	}

	// Create Fiber app
	app := fiber.New()

	// Configure CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "https://beta-intelligence.vercel.app",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

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
