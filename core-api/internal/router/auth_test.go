package router_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/upload"
	"github.com/1024XEngineer/Holonic-Asset/internal/router"
)

type authRouterStub struct{}

func (authRouterStub) Login(
	_ context.Context,
	request dto.LoginRequest,
) (dto.SuccessResponse[dto.LoginResponse], error) {
	return dto.NewTypedSuccessResponse(dto.LoginResponse{
		AccessToken: "token-for-" + request.Username,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		User:        dto.LoginUser{ID: 1, Username: request.Username},
	}), nil
}

type authManagerErrorStub struct {
	err error
}

func (s authManagerErrorStub) Login(context.Context, string, string) (*auth.LoginResult, error) {
	return nil, s.err
}

func (authManagerErrorStub) VerifyToken(string) (*auth.Claims, error) {
	return &auth.Claims{}, nil
}

func TestAuthenticationKeepsLoginPublicAndProtectsAPIRoutes(t *testing.T) {
	server := router.Register(
		nil,
		nil,
		nil,
		handler.NewUploadHandler(upload.NewManager(nil)),
		router.Authentication{
			Router: authRouterStub{},
			Middleware: func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c echo.Context) error {
					return echo.NewHTTPError(http.StatusUnauthorized)
				}
			},
		},
	)

	loginRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"username":"login-test-user","password":"login-test-password"}`),
	)
	loginRequest.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	loginResponse := httptest.NewRecorder()
	server.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("expected public login status %d, got %d: %s", http.StatusOK, loginResponse.Code, loginResponse.Body.String())
	}

	protectedResponse := httptest.NewRecorder()
	server.ServeHTTP(protectedResponse, httptest.NewRequest(http.MethodPost, "/api/v1/uploads", nil))
	if protectedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("expected protected route status %d, got %d", http.StatusUnauthorized, protectedResponse.Code)
	}
}

func TestAuthenticationUsesConfiguredAllowedOrigins(t *testing.T) {
	server := router.Register(
		nil,
		nil,
		nil,
		nil,
		router.Authentication{AllowedOrigins: []string{"https://example.com"}},
	)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/openapi.json", nil)
	request.Header.Set(echo.HeaderOrigin, "https://example.com")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if origin := response.Header().Get(echo.HeaderAccessControlAllowOrigin); origin != "https://example.com" {
		t.Fatalf("expected configured allowed origin, got %q", origin)
	}
}

func TestLoginResponseDoesNotExposeInternalErrors(t *testing.T) {
	internalMessage := "auth: find user: database connection details"
	server := router.Register(
		nil,
		nil,
		nil,
		nil,
		router.Authentication{
			Router: handler.NewAuthHandler(authManagerErrorStub{err: errors.New(internalMessage)}),
		},
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(`{"username":"login-test-user","password":"login-test-password"}`),
	)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d: %s", http.StatusInternalServerError, response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), internalMessage) {
		t.Fatalf("response exposed internal error: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), http.StatusText(http.StatusInternalServerError)) {
		t.Fatalf("expected generic internal server error response, got %s", response.Body.String())
	}
}
