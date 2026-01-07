package config

import (
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v7"
)

// ServerConf holds server configuration
type ServerConf struct {
	Port string `env:"SERVER_PORT" envDefault:"8080"`
	Env  string `env:"APP_ENV" envDefault:"development"`
}

// DatabaseConf holds database configuration
type DatabaseConf struct {
	Host     string `env:"DB_HOST" envDefault:"localhost"`
	Port     string `env:"DB_PORT" envDefault:"5432"`
	User     string `env:"DB_USER" envDefault:"postgres"`
	Password string `env:"DB_PASSWORD" envDefault:""`
	Name     string `env:"DB_NAME" envDefault:"postgis_36_sample"`
}

// AppConfig holds all application configuration
type AppConfig struct {
	Server   ServerConf
	Database DatabaseConf
	RabbitMQ RabbitMQConf
}

// Config is the global configuration instance
var Config AppConfig

// InitConfig initializes application configuration from environment variables
func InitConfig() error {
	log.Println("🔧 Initializing application configuration...")

	// Parse environment variables into Config struct
	if err := env.Parse(&Config); err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate configuration
	if err := validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	log.Println("✅ Configuration initialized successfully")
	return nil
}

// validateConfig validates the loaded configuration
func validateConfig() error {
	// Validate Server configuration
	if Config.Server.Port == "" {
		return fmt.Errorf("SERVER_PORT is required")
	}

	// Validate Database configuration
	requiredDBFields := map[string]string{
		"DB_HOST":     Config.Database.Host,
		"DB_PORT":     Config.Database.Port,
		"DB_USER":     Config.Database.User,
		"DB_PASSWORD": Config.Database.Password,
		"DB_NAME":     Config.Database.Name,
	}

	for field, value := range requiredDBFields {
		if value == "" {
			return fmt.Errorf("%s is required", field)
		}
	}

	// Validate RabbitMQ configuration
	if err := Config.RabbitMQ.ValidateRabbitMQConfig(); err != nil {
		return err
	}

	return nil
}

// GetDatabaseDSN returns the database connection string
func (d *DatabaseConf) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		d.Host,
		d.User,
		d.Password,
		d.Name,
		d.Port,
	)
}

// IsProduction returns true if running in production environment
func (s *ServerConf) IsProduction() bool {
	return s.Env == "production" || s.Env == "prod"
}

// IsDevelopment returns true if running in development environment
func (s *ServerConf) IsDevelopment() bool {
	return s.Env == "development" || s.Env == "dev"
}

// IsTest returns true if running in test environment
func (s *ServerConf) IsTest() bool {
	return s.Env == "test"
}

// PrintConfig prints the current configuration (excluding sensitive data)
func PrintConfig() {
	log.Println("📋 Current Configuration:")
	log.Printf("   Environment: %s", Config.Server.Env)
	log.Printf("   Server Port: %s", Config.Server.Port)
	log.Printf("   Database Host: %s:%s", Config.Database.Host, Config.Database.Port)
	log.Printf("   Database Name: %s", Config.Database.Name)
	log.Printf("   RabbitMQ Host: %s:%s", Config.RabbitMQ.Host, Config.RabbitMQ.Port)
	log.Printf("   RabbitMQ Exchange: %s", Config.RabbitMQ.Exchange)
	log.Printf("   Prefetch Count: %d", Config.RabbitMQ.PrefetchCount)
	log.Printf("   Connection Pool Size: %d", Config.RabbitMQ.PoolSize)
}

// GetRabbitMQURL constructs the RabbitMQ connection URL

// ValidateRabbitMQConfig validates RabbitMQ configuration

// WaitForRabbitMQ waits for RabbitMQ to be ready

// GetEnv returns environment variable value or default
func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// MustGetEnv returns environment variable value or panics if not set
func MustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("❌ Required environment variable %s is not set", key)
	}
	return value
}
