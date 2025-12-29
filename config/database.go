package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	// 🔐 Validate required environment variables
	requiredEnv := []string{
		"DB_HOST",
		"DB_USER",
		"DB_PASSWORD",
		"DB_NAME",
		"DB_PORT",
	}

	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("❌ %s is not set", env)
		}
	}

	// 🚨 Safety guard: prevent tests from using non-test DB
	if os.Getenv("APP_ENV") == "test" &&
		!strings.HasSuffix(os.Getenv("DB_NAME"), "_test") {
		log.Fatal("❌ APP_ENV=test but DB_NAME is not a test database")
	}

	// Build DSN
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect database:", err)
	}

	DB = db
	log.Println("✅ Database connected")
}
