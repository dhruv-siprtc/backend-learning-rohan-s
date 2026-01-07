package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDB initializes the database connection with retry logic
func ConnectDB() error {
	// Validate environment for test databases
	if err := validateTestEnvironment(); err != nil {
		return err
	}

	// Configure GORM logger based on environment
	gormLogger := logger.Default.LogMode(logger.Info)
	if Config.Server.IsProduction() {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	// Build DSN using Config structure
	dsn := Config.Database.GetDatabaseDSN()

	// Connect to database with retry logic
	var db *gorm.DB
	var err error
	maxAttempts := 10
	retryInterval := 3 * time.Second

	log.Println("🔌 Connecting to database...")

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: gormLogger,
			NowFunc: func() time.Time {
				return time.Now().UTC()
			},
		})

		if err == nil {
			// Test connection
			sqlDB, err := db.DB()
			if err == nil {
				if err = sqlDB.Ping(); err == nil {
					break
				}
			}
		}

		if attempt < maxAttempts {
			log.Printf("⏳ Database not ready (attempt %d/%d). Retrying in %v...",
				attempt, maxAttempts, retryInterval)
			time.Sleep(retryInterval)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect to database after %d attempts: %w", maxAttempts, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	log.Println("✅ Database connected successfully")
	log.Printf("   Host: %s:%s", Config.Database.Host, Config.Database.Port)
	log.Printf("   Database: %s", Config.Database.Name)

	return nil
}

// validateTestEnvironment ensures test databases are properly configured
func validateTestEnvironment() error {
	if Config.Server.IsTest() {
		if !strings.HasSuffix(Config.Database.Name, "_test") {
			return fmt.Errorf(
				"APP_ENV=test but DB_NAME (%s) is not a test database. "+
					"Test database names must end with '_test'",
				Config.Database.Name,
			)
		}
	}

	// Prevent accidental production database usage in non-production
	if !Config.Server.IsProduction() {
		productionIndicators := []string{"prod", "production"}
		for _, indicator := range productionIndicators {
			if strings.Contains(strings.ToLower(Config.Database.Name), indicator) {
				log.Printf("⚠️  WARNING: Using production-like database name (%s) in %s environment",
					Config.Database.Name, Config.Server.Env)
			}
		}
	}

	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	log.Println("🔌 Closing database connection...")
	return sqlDB.Close()
}

// HealthCheck checks if the database is accessible
func HealthCheck() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// TruncateAllTables truncates all tables (for testing only)
func TruncateAllTables() error {
	if !Config.Server.IsTest() {
		return fmt.Errorf("truncate operation only allowed in test environment")
	}

	log.Println("🗑️  Truncating all tables...")

	tables := []string{"users"} // Add more tables as needed

	for _, table := range tables {
		if err := DB.Exec(fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table)).Error; err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}

	log.Println("✅ All tables truncated")
	return nil
}

// InitTestDB initializes database for testing
func InitTestDB() error {
	// Initialize config
	if err := InitConfig(); err != nil {
		return err
	}

	// Connect to database
	if err := ConnectDB(); err != nil {
		return err
	}

	// Truncate tables
	if err := TruncateAllTables(); err != nil {
		return err
	}

	return nil
}
