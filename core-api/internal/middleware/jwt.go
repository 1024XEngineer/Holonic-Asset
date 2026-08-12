package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
)

const (
	ContextUserID   = "userID"
	ContextUsername = "username"
)

type TokenVerifier interface {
	VerifyToken(token string) (*auth.Claims, error)
}

func JWT(verifier TokenVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			scheme, token, found := strings.Cut(c.Request().Header.Get(echo.HeaderAuthorization), " ")
			if !found || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid bearer token")
			}
			claims, err := verifier.VerifyToken(strings.TrimSpace(token))
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired bearer token").SetInternal(err)
			}
			c.Set(ContextUserID, claims.Subject)
			c.Set(ContextUsername, claims.Username)
			return next(c)
		}
	}
}
