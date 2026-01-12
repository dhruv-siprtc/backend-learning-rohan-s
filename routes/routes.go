package routes

import (
	"golang-postgre/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	// Health check endpoints
	e.GET("/health", handlers.HealthCheckHandler)
	e.GET("/readiness", handlers.ReadinessHandler)
	e.GET("/liveness", handlers.LivenessHandler)

	// User management endpoints
	e.POST("/users", handlers.CreateUser)
	e.GET("/users", handlers.GetUsers)
	e.GET("/users/:id", handlers.GetUserByID)
	e.PUT("/users/:id", handlers.UpdateUser)
	e.DELETE("/users/:id", handlers.DeleteUser)
}
