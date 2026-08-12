package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
)

type authManagerStub struct {
	result   *auth.LoginResult
	loginErr error
}

func (s authManagerStub) Login(context.Context, string, string) (*auth.LoginResult, error) {
	return s.result, s.loginErr
}

func (authManagerStub) VerifyToken(string) (*auth.Claims, error) {
	return &auth.Claims{}, nil
}

func TestAuthHandlerLoginMapsInvalidCredentialsToUnauthorized(t *testing.T) {
	cause := fmt.Errorf("wrapped credentials error: %w", auth.ErrInvalidCredentials)
	handler := NewAuthHandler(authManagerStub{loginErr: cause})

	_, err := handler.Login(context.Background(), dto.LoginRequest{})
	assertHTTPError(t, err, http.StatusUnauthorized, auth.ErrInvalidCredentials.Error(), cause)
}

func TestAuthHandlerLoginHidesInternalErrors(t *testing.T) {
	cause := errors.New("auth: find user: database connection details")
	handler := NewAuthHandler(authManagerStub{loginErr: cause})

	_, err := handler.Login(context.Background(), dto.LoginRequest{})
	assertHTTPError(t, err, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError), cause)
}

func TestAuthHandlerLoginReturnsTokenAndUser(t *testing.T) {
	handler := NewAuthHandler(authManagerStub{result: &auth.LoginResult{
		AccessToken: "access-token",
		ExpiresIn:   3600,
		User: auth.User{
			ID:       7,
			Username: "login-test-user",
			Email:    "login-test-user@example.com",
		},
	}})

	response, err := handler.Login(context.Background(), dto.LoginRequest{
		Username: "login-test-user",
		Password: "login-test-password",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if response.Data.AccessToken != "access-token" || response.Data.TokenType != "Bearer" || response.Data.ExpiresIn != 3600 {
		t.Fatalf("unexpected token response: %+v", response.Data)
	}
	if response.Data.User.ID != 7 || response.Data.User.Username != "login-test-user" || response.Data.User.Email != "login-test-user@example.com" {
		t.Fatalf("unexpected user response: %+v", response.Data.User)
	}
}

func assertHTTPError(t *testing.T, err error, status int, message string, internal error) {
	t.Helper()

	var httpErr *echo.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected *echo.HTTPError, got %T: %v", err, err)
	}
	if httpErr.Code != status {
		t.Fatalf("expected status %d, got %d", status, httpErr.Code)
	}
	if httpErr.Message != message {
		t.Fatalf("expected message %q, got %q", message, httpErr.Message)
	}
	if !errors.Is(httpErr.Internal, internal) {
		t.Fatalf("expected internal error %v, got %v", internal, httpErr.Internal)
	}
}
