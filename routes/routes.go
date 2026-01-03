package routes

import (
	"golang-postgre/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	e.GET("/health", handlers.HealthCheck)
	e.POST("/users", handlers.CreateUser)
	e.GET("/users", handlers.GetUsers)
	e.GET("/users/:id", handlers.GetUserByID)
	e.PUT("/users/:id", handlers.UpdateUser)
	e.DELETE("/users/:id", handlers.DeleteUser)
}
