package handlers

import (
	"net/http"

	"golang-postgre/config"

	"github.com/labstack/echo/v4"
)

func HealthCheck(c echo.Context) error {

	sqlDB, err := config.DB.DB()
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{
			"status":   "unhealthy",
			"database": "disconnected",
			"error":    err.Error(),
		})
	}

	if err := sqlDB.Ping(); err != nil {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{
			"status":   "unhealthy",
			"database": "ping failed",
			"error":    err.Error(),
		})
	}

	if config.RabbitConn == nil || config.RabbitConn.IsClosed() {
		return c.JSON(http.StatusServiceUnavailable, echo.Map{
			"status":   "unhealthy",
			"database": "connected",
			"rabbitmq": "disconnected",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":   "healthy",
		"database": "connected",
		"rabbitmq": "connected",
	})
}
