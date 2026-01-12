package handlers

import (
	"net/http"
	"time"

	"golang-postgre/config"
	"golang-postgre/events"
	"golang-postgre/models"
	"golang-postgre/producer"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request"})
	}

	// Check if email already exists
	var count int64
	config.DB.
		Model(&models.User{}).
		Where("email = ? AND deleted_at IS NULL", user.Email).
		Count(&count)

	if count > 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Email already in use",
		})
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Password encryption failed"})
	}
	user.Password = string(hash)

	// Create user in database
	if err := config.DB.Create(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	// Publish USER_CREATED event using Paota
	event := events.UserEvent{
		Event:     "USER_CREATED",
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: events.UserData{
			UserID: user.ID,
			Name:   user.Name,
			Email:  user.Email,
		},
	}

	// Get producer instance and publish event
	prod, err := producer.GetProducer()
	if err != nil {
		c.Logger().Error("Failed to get producer instance:", err)
		// Continue - don't fail the request if event publishing fails
	} else {
		if err := prod.PublishUserCreated(event); err != nil {
			c.Logger().Error("Failed to publish USER_CREATED event:", err)
			// Log but don't fail the request
		}
	}

	return c.JSON(http.StatusCreated, user)
}

func UpdateUser(c echo.Context) error {
	var user models.User

	// Find existing user
	if err := config.DB.
		Where("id = ? AND deleted_at IS NULL", c.Param("id")).
		First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	// Parse update request
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	// Check email uniqueness
	var count int64
	config.DB.
		Model(&models.User{}).
		Where("email = ? AND id <> ? AND deleted_at IS NULL", input.Email, user.ID).
		Count(&count)

	if count > 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Email already in use",
		})
	}

	// Prepare updates
	updates := map[string]interface{}{
		"name":  input.Name,
		"email": input.Email,
	}

	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "Password encryption failed",
			})
		}
		updates["password"] = string(hash)
	}

	// Update user in database
	if err := config.DB.
		Model(&user).
		Updates(updates).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	// Reload user data to get updated values
	if err := config.DB.
		Where("id = ? AND deleted_at IS NULL", user.ID).
		First(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to reload user data",
		})
	}

	// Publish USER_UPDATED event using Paota
	event := events.UserEvent{
		Event:     "USER_UPDATED",
		Version:   "1.0",
		Timestamp: time.Now(),
		Data: events.UserData{
			UserID: user.ID,
			Name:   user.Name,
			Email:  user.Email,
		},
	}

	// Get producer instance and publish event
	prod, err := producer.GetProducer()
	if err != nil {
		c.Logger().Error("Failed to get producer instance:", err)
		// Continue - don't fail the request if event publishing fails
	} else {
		if err := prod.PublishUserUpdated(event); err != nil {
			c.Logger().Error("Failed to publish USER_UPDATED event:", err)
			// Log but don't fail the request
		}
	}

	return c.JSON(http.StatusOK, user)
}

func GetUsers(c echo.Context) error {
	var users []models.User
	if err := config.DB.Where("deleted_at IS NULL").Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, users)
}

func GetUserByID(c echo.Context) error {
	var user models.User
	if err := config.DB.Where("id = ? AND deleted_at IS NULL", c.Param("id")).First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, user)
}

func DeleteUser(c echo.Context) error {
	var user models.User

	if err := config.DB.
		Where("id = ? AND deleted_at IS NULL", c.Param("id")).
		First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "User not found",
		})
	}

	if err := config.DB.Delete(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}
