package tests

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"golang-postgre/config"
	"golang-postgre/handlers"

	"github.com/labstack/echo/v4"
)

func TestCreateUser(t *testing.T) {
	// Load env manually for tests
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "123")
	os.Setenv("DB_NAME", "postgis_36_sample")
	os.Setenv("DB_PORT", "5432")

	config.ConnectDB()

	e := echo.New()
	payload := `{"name":"naruto","email":"roh@test.com","password":"12356"}`

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBuffer([]byte(payload)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	c := e.NewContext(req, rec)
	if err := handlers.CreateUser(c); err != nil {
		t.Fatal(err)
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", rec.Code)
	}
}
