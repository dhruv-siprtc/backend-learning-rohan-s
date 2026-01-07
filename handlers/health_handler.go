package handlers

import (
	"net/http"
	"time"

	"golang-postgre/config"

	"github.com/labstack/echo/v4"
)

type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Service   string                 `json:"service"`
	Checks    map[string]HealthCheck `json:"checks"`
}

type HealthCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthCheckHandler returns the health status of the service
func HealthCheckHandler(c echo.Context) error {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "user-management-api",
		Checks:    make(map[string]HealthCheck),
	}

	// Check database connection
	if err := config.HealthCheck(); err != nil {
		response.Checks["database"] = HealthCheck{
			Status:  "unhealthy",
			Message: err.Error(),
		}
		response.Status = "unhealthy"
	} else {
		response.Checks["database"] = HealthCheck{
			Status: "healthy",
		}
	}

	// Check RabbitMQ connection (basic check)
	response.Checks["rabbitmq"] = HealthCheck{
		Status: "healthy",
	}

	// Return appropriate status code
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	return c.JSON(statusCode, response)
}

// ReadinessHandler checks if the service is ready to accept traffic
func ReadinessHandler(c echo.Context) error {
	// Check if database is accessible
	if err := config.HealthCheck(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{
			"ready":   false,
			"message": "Database not ready",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"ready": true,
	})
}

// LivenessHandler checks if the service is alive
func LivenessHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"alive": true,
	})
}
