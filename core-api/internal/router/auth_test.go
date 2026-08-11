package router_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/handler"
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
