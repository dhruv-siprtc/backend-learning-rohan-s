package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang-postgre/config"
	"golang-postgre/consumer"

	"github.com/joho/godotenv"
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

	// Wait for RabbitMQ to be ready
	if err := config.WaitForRabbitMQ(); err != nil {
		log.Fatalf("❌ RabbitMQ not ready: %v", err)
	}

	// Get RabbitMQ configuration
	rmqConfig := config.Config.RabbitMQ

	// Initialize Paota consumer
	log.Println("🎧 Initializing consumer service...")
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
