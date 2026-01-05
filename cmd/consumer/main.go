package main

import (
	"log"

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

	// Wait for RabbitMQ to be ready (especially important in Docker)
	if err := config.WaitForRabbitMQ(); err != nil {
		log.Fatalf("❌ RabbitMQ not ready: %v", err)
	}

	// Get RabbitMQ configuration
	rmqConfig := config.Config.RabbitMQ

	// Initialize Paota consumer
	log.Println("🎧 Initializing consumer...")
	consumerService, err := consumer.InitializeConsumer(rmqConfig)
	if err != nil {
		log.Fatalf("❌ Failed to initialize consumer: %v", err)
	}

	// Cleanup on shutdown
	defer func() {
		if consumerService != nil {
			consumerService.Close()
			log.Println("✅ Consumer closed")
		}
	}()

	// Start consuming messages
	log.Println("🚀 Starting consumer service...")
	log.Printf("📥 Listening to queues:")
	log.Printf("   - %s (routing key: %s)", rmqConfig.CreatedQueue, rmqConfig.CreatedRoutingKey)
	log.Printf("   - %s (routing key: %s)", rmqConfig.UpdatedQueue, rmqConfig.UpdatedRoutingKey)
	log.Printf("📍 Environment: %s", config.Config.Server.Env)
	log.Printf("⚙️  Prefetch Count: %d", rmqConfig.PrefetchCount)
	log.Printf("⚙️  Pool Size: %d", rmqConfig.PoolSize)

	if err := consumerService.Start(); err != nil {
		log.Fatalf("❌ Consumer failed to start: %v", err)
	}

	// Keep the application running
	log.Println("✅ Consumer is running. Press Ctrl+C to stop.")
	select {}
}
