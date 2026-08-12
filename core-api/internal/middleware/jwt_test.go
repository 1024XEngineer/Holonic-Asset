package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"

	appmiddleware "github.com/1024XEngineer/Holonic-Asset/internal/middleware"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
)

type tokenVerifierStub struct{}

func (tokenVerifierStub) VerifyToken(token string) (*auth.Claims, error) {
	if token != "valid-token" {
		return nil, errors.New("invalid token")
	}
	return &auth.Claims{
		Username: "login-test-user",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: "1",
		},
	}, nil
}

func TestJWTRejectsMissingToken(t *testing.T) {
	e := echo.New()
	e.GET("/protected", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, appmiddleware.JWT(tokenVerifierStub{}))
	response := httptest.NewRecorder()
	e.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestJWTRejectsInvalidToken(t *testing.T) {
	e := echo.New()
	e.GET("/protected", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, appmiddleware.JWT(tokenVerifierStub{}))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set(echo.HeaderAuthorization, "Bearer invalid-token")
	response := httptest.NewRecorder()

	e.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestJWTExposesAuthenticatedUser(t *testing.T) {
	e := echo.New()
	e.GET("/protected", func(c echo.Context) error {
		if c.Get(appmiddleware.ContextUserID) != "1" || c.Get(appmiddleware.ContextUsername) != "login-test-user" {
			t.Fatalf("unexpected authenticated context: id=%v username=%v", c.Get(appmiddleware.ContextUserID), c.Get(appmiddleware.ContextUsername))
		}
		return c.NoContent(http.StatusOK)
	}, appmiddleware.JWT(tokenVerifierStub{}))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set(echo.HeaderAuthorization, "Bearer valid-token")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}
