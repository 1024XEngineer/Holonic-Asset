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
	loginErr error
}

func (s authManagerStub) Login(context.Context, string, string) (*auth.LoginResult, error) {
	return nil, s.loginErr
}

func (authManagerStub) VerifyToken(string) (*auth.Claims, error) {
	return nil, nil
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
