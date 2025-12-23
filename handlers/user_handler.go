package handlers

import (
	"net/http"

	"golang-postgre/config"
	"golang-postgre/models"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Password encryption failed"})
	}
	user.Password = string(hash)

	if err := config.DB.Create(&user).Error; err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, user)
}

func GetUsers(c echo.Context) error {
	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, users)
}

func GetUserByID(c echo.Context) error {
	var user models.User
	if err := config.DB.First(&user, c.Param("id")).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}
	return c.JSON(http.StatusOK, user)
}

func UpdateUser(c echo.Context) error {
	var user models.User
	if err := config.DB.First(&user, c.Param("id")).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	user.Name = input.Name
	user.Email = input.Email

	config.DB.Save(&user)
	return c.JSON(http.StatusOK, user)
}

func DeleteUser(c echo.Context) error {
	if err := config.DB.Delete(&models.User{}, c.Param("id")).Error; err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}
