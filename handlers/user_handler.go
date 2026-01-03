package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"golang-postgre/config"
	"golang-postgre/events"
	"golang-postgre/models"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request"})
	}

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

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Password encryption failed"})
	}
	user.Password = string(hash)

	if err := config.DB.Create(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

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

	eventBody, err := json.Marshal(event)
	if err != nil {
		c.Logger().Error("Failed to marshal user created event:", err)
	} else {
		if err := config.PublishUserCreated(eventBody); err != nil {

			c.Logger().Error("Failed to publish user created event:", err)
		}
	}

	return c.JSON(http.StatusCreated, user)
}

func UpdateUser(c echo.Context) error {
	var user models.User

	if err := config.DB.
		Where("id = ? AND deleted_at IS NULL", c.Param("id")).
		First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

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

	if err := config.DB.
		Model(&user).
		Updates(updates).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	if err := config.DB.
		Where("id = ? AND deleted_at IS NULL", user.ID).
		First(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to reload user data",
		})
	}

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

	eventBody, err := json.Marshal(event)
	if err != nil {
		c.Logger().Error("Failed to marshal user updated event:", err)
	} else {
		if err := config.PublishUserUpdated(eventBody); err != nil {
			c.Logger().Error("Failed to publish user updated event:", err)
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
