package main

import (
	"log"

	"golang-postgre/config"
	"golang-postgre/models"
	"golang-postgre/producer"
	"golang-postgre/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	// Initialize configuration
	if err := config.InitConfig(); err != nil {
		log.Fatalf("❌ Failed to initialize configuration: %v", err)
	}

	// Print configuration (for debugging)
	if config.Config.Server.IsDevelopment() {
		config.PrintConfig()
	}

	// Wait for RabbitMQ to be ready (especially important in Docker)
	if err := config.WaitForRabbitMQ(); err != nil {
		log.Printf("⚠️  RabbitMQ wait failed: %v (will retry on publish)", err)
		// Continue anyway - producer will retry on actual publish
	}

	// Connect to database
	if err := config.ConnectDB(); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer config.CloseDB()

	// Run database migrations
	if err := config.DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("❌ Database migration failed: %v", err)
	}
	log.Println("✅ Database migrations completed")

	// Initialize Paota producer
	_, err := producer.InitializeProducer(config.Config.RabbitMQ)
	if err != nil {
		log.Fatalf("❌ Failed to initialize producer: %v", err)
	}

	// Cleanup producer on shutdown
	defer func() {
		prod, _ := producer.GetProducer()
		if prod != nil {
			prod.Close()
			log.Println("✅ Producer closed")
		}
	}()

	// Initialize Echo server
	e := echo.New()
	e.HideBanner = true

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			echo.GET,
			echo.POST,
			echo.PUT,
			echo.DELETE,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin,
			echo.HeaderContentType,
			echo.HeaderAccept,
			echo.HeaderAuthorization,
		},
	}))

	// Register routes
	routes.RegisterRoutes(e)

	// Start server
	port := config.Config.Server.Port
	log.Printf("🚀 Server running at http://localhost:%s", port)
	log.Printf("📍 Environment: %s", config.Config.Server.Env)
	log.Println("📍 Health check: http://localhost:" + port + "/health")

	if err := e.Start(":" + port); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}
