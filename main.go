package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang-postgre/config"
	"golang-postgre/consumer"
	"golang-postgre/models"
	"golang-postgre/producer"
	"golang-postgre/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Define command-line flags
	mode := flag.String("mode", "api", "Run mode: 'api' or 'consumer'")
	flag.Parse()

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

	// Run based on mode
	switch *mode {
	case "api":
		runAPIServer()
	case "consumer":
		runConsumer()
	default:
		log.Fatalf("❌ Invalid mode: %s. Use 'api' or 'consumer'", *mode)
	}
}

func runAPIServer() {
	log.Println("🚀 Starting in API mode...")

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

	// Wait for RabbitMQ to be ready
	if err := config.WaitForRabbitMQ(); err != nil {
		log.Printf("⚠️  RabbitMQ wait failed: %v (will retry on publish)", err)
	}

	// Initialize Paota producer
	log.Println("🔧 Initializing producer...")
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

	// Setup graceful shutdown
	go func() {
		port := config.Config.Server.Port
		log.Printf("🚀 API Server running at http://localhost:%s", port)
		log.Printf("📍 Environment: %s", config.Config.Server.Env)
		log.Println("📍 Health check: http://localhost:" + port + "/health")
		log.Println("📍 Press Ctrl+C to stop")

		if err := e.Start(":" + port); err != nil {
			log.Printf("❌ Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down API server gracefully...")

	// Cleanup
	if err := e.Close(); err != nil {
		log.Printf("⚠️  Error closing server: %v", err)
	}

	log.Println("✅ API Server stopped")
}

func runConsumer() {
	log.Println("🎧 Starting in Consumer mode...")

	// Wait for RabbitMQ to be ready
	if err := config.WaitForRabbitMQ(); err != nil {
		log.Fatalf("❌ RabbitMQ not ready: %v", err)
	}

	// Get RabbitMQ configuration
	rmqConfig := config.Config.RabbitMQ

	// Initialize Paota consumer
	log.Println("🔧 Initializing consumer service...")
	consumerService, err := consumer.InitializeConsumer(rmqConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize consumer: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start consuming in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Println("🚀 Starting consumer service...")
		log.Printf("📥 Listening to queues:")
		log.Printf("   - %s (routing key: %s)", rmqConfig.CreatedQueue, rmqConfig.CreatedRoutingKey)
		log.Printf("   - %s (routing key: %s)", rmqConfig.UpdatedQueue, rmqConfig.UpdatedRoutingKey)
		log.Printf("📍 Environment: %s", config.Config.Server.Env)
		log.Printf("⚙️  Prefetch Count: %d", rmqConfig.PrefetchCount)
		log.Printf("⚙️  Pool Size: %d", rmqConfig.PoolSize)
		log.Println("📍 Press Ctrl+C to stop")

		if err := consumerService.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("⚠️  Received signal: %v. Shutting down gracefully...", sig)
	case err := <-errChan:
		log.Printf("❌ Consumer error: %v. Shutting down...", err)
	}

	// Cleanup
	if consumerService != nil {
		if err := consumerService.Close(); err != nil {
			log.Printf("⚠️  Error during cleanup: %v", err)
		}
	}

	log.Println("✅ Consumer service stopped")
}
