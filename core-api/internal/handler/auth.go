package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
	"github.com/1024XEngineer/Holonic-Asset/internal/module/auth"
)

type AuthHandler struct {
	manager auth.Manager
}

func NewAuthHandler(manager auth.Manager) *AuthHandler {
	return &AuthHandler{manager: manager}
}

func (h *AuthHandler) Login(
	ctx context.Context,
	request dto.LoginRequest,
) (dto.SuccessResponse[dto.LoginResponse], error) {
	result, err := h.manager.Login(ctx, request.Username, request.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			return dto.SuccessResponse[dto.LoginResponse]{}, echo.NewHTTPError(
				http.StatusUnauthorized,
				auth.ErrInvalidCredentials.Error(),
			).SetInternal(err)
		}
		return dto.SuccessResponse[dto.LoginResponse]{}, err
	}
	return dto.NewTypedSuccessResponse(dto.LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   result.ExpiresIn,
		User: dto.LoginUser{
			ID:       result.User.ID,
			Username: result.User.Username,
			Email:    result.User.Email,
		},
	}), nil
}
