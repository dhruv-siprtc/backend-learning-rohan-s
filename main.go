package main

import (
	"log"

	"golang-postgre/config"
	"golang-postgre/models"
	"golang-postgre/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

func main() {
	godotenv.Load()

	config.ConnectDB()
	config.DB.AutoMigrate(&models.User{})

	e := echo.New()
	routes.RegisterRoutes(e)

	log.Fatal(e.Start(":8080"))
}
