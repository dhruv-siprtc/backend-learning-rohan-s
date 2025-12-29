package main

import (
	"log"
	"os"

	"golang-postgre/config"
	"golang-postgre/models"
	"golang-postgre/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {

	godotenv.Load()
	port := os.Getenv("SERVER_PORT")

	if port == "" {
		log.Fatal("❌ SERVER_PORT is not set")
	}

	config.ConnectDB()
	config.DB.AutoMigrate(&models.User{})

	e := echo.New()

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
		},
	}))

	routes.RegisterRoutes(e)

	log.Println("🚀 Server running at http://localhost:%s\n", port)
	log.Fatal(e.Start(":" + port))
}
