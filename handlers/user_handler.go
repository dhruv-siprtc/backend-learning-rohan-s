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

	return c.JSON(http.StatusCreated, user)
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

func UpdateUser(c echo.Context) error {
	var user models.User

	// 1️⃣ Ensure ACTIVE user exists
	if err := config.DB.
		Where("id = ? AND deleted_at IS NULL", c.Param("id")).
		First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "User not found"})
	}

	// Input struct (now includes password)
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	// 2️⃣ Prevent duplicate active emails
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

	// 3️⃣ Prepare update map
	updates := map[string]interface{}{
		"name":  input.Name,
		"email": input.Email,
	}

	// 4️⃣ Hash password if provided
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "Password encryption failed",
			})
		}
		updates["password"] = string(hash)
	}

	// 5️⃣ Explicit update + error check
	if err := config.DB.
		Model(&user).
		Updates(updates).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, user)
}

func DeleteUser(c echo.Context) error {
	var user models.User

	// 1️⃣ Check if ACTIVE user exists
	if err := config.DB.
		Where("id = ? AND deleted_at IS NULL", c.Param("id")).
		First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "User not found",
		})
	}

	// 2️⃣ Perform SOFT delete
	if err := config.DB.Delete(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": err.Error(),
		})
	}

	return c.NoContent(http.StatusNoContent)
}
