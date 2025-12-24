package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang-postgre/config"
	"golang-postgre/handlers"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

func TestCreateUser(t *testing.T) {
	// Load TEST environment variables (not dev .env)
	if err := godotenv.Load("../.env.test"); err != nil {
		t.Fatal("❌ Failed to load .env.test file")
	}

	// 🚨 Safety check: never allow dev DB in tests
	if os.Getenv("DB_NAME") == "postgis_36_sample" {
		t.Fatal("❌ Tests are using the development database")
	}

	// Connect to TEST database
	config.ConnectDB()

	// Clean test data before running test
	config.DB.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE")

	e := echo.New()
	payload := `{"name":"naruto","email":"roh@test.com","password":"12356"}`

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(payload))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	if err := handlers.CreateUser(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
}
