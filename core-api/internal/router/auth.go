package router

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/1024XEngineer/Holonic-Asset/internal/dto"
)

const authLoginPath = "/auth/login"

type AuthRouter interface {
	Login(
		ctx context.Context,
		request dto.LoginRequest,
	) (dto.SuccessResponse[dto.LoginResponse], error)
}

type loginInput struct {
	Body dto.LoginRequest
}

type loginOutput struct {
	Body dto.SuccessResponse[dto.LoginResponse]
}

func RegisterAuthRoutes(api huma.API, authRouter AuthRouter) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        authLoginPath,
		Summary:     "Log in",
		Tags:        []string{"Authentication"},
		Errors:      []int{http.StatusUnauthorized},
	}, func(ctx context.Context, input *loginInput) (*loginOutput, error) {
		response, err := authRouter.Login(ctx, input.Body)
		if err != nil {
			return nil, openAPIError(err)
		}
		return &loginOutput{Body: response}, nil
	})
}
