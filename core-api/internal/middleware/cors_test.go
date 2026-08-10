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
	if vary := response.Header().Values(echo.HeaderVary); !contains(vary, echo.HeaderOrigin) {
		t.Fatalf("expected response to vary by Origin, got %q", vary)
	}
	if exposed := response.Header().Get(echo.HeaderAccessControlExposeHeaders); exposed != "" {
		t.Fatalf("expected no exposed headers, got %q", exposed)
	}
}

func TestCORSMarksResponsesWithoutOriginAsVariant(t *testing.T) {
	e := echo.New()
	e.Use(appmiddleware.CORS())
	e.GET("/resource", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/resource", nil)
	response := httptest.NewRecorder()

	e.ServeHTTP(response, request)

	if vary := response.Header().Values(echo.HeaderVary); !contains(vary, echo.HeaderOrigin) {
		t.Fatalf("expected response to vary by Origin, got %q", vary)
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
	request.Header.Set(echo.HeaderAccessControlRequestHeaders, "Content-Type,X-Custom-Header")
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
	if headers := response.Header().Get(echo.HeaderAccessControlAllowHeaders); headers != echo.HeaderContentType {
		t.Fatalf("expected only Content-Type to be allowed, got %q", headers)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
