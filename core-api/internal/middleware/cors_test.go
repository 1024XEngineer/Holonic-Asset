package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	appmiddleware "github.com/1024XEngineer/Holonic-Asset/internal/middleware"
)

func TestCORSAllowsRequestsFromAnyOrigin(t *testing.T) {
	e := echo.New()
	e.Use(appmiddleware.CORS())
	e.GET("/resource", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	request.Header.Set(echo.HeaderOrigin, "https://example.com")
	response := httptest.NewRecorder()

	e.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if origin := response.Header().Get(echo.HeaderAccessControlAllowOrigin); origin != "*" {
		t.Fatalf("expected wildcard allowed origin, got %q", origin)
	}
}

func TestCORSAllowsPreflightRequests(t *testing.T) {
	e := echo.New()
	e.Use(appmiddleware.CORS())
	e.POST("/resource", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	request.Header.Set(echo.HeaderOrigin, "https://example.com")
	request.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPost)
	request.Header.Set(echo.HeaderAccessControlRequestHeaders, "Authorization,X-Custom-Header")
	response := httptest.NewRecorder()

	e.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, response.Code)
	}
	if origin := response.Header().Get(echo.HeaderAccessControlAllowOrigin); origin != "*" {
		t.Fatalf("expected wildcard allowed origin, got %q", origin)
	}
	if methods := response.Header().Get(echo.HeaderAccessControlAllowMethods); !strings.Contains(methods, http.MethodPost) {
		t.Fatalf("expected POST in allowed methods, got %q", methods)
	}
	if headers := response.Header().Get(echo.HeaderAccessControlAllowHeaders); headers != "Authorization,X-Custom-Header" {
		t.Fatalf("expected requested headers to be allowed, got %q", headers)
	}
}
