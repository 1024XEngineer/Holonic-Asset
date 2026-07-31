package echox

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const UserClaimsKey = "user_claims"

type Context struct {
	echo.Context
	requestContext context.Context
}

func newContext(c echo.Context) *Context {
	reqCtx := context.Background()
	if req := c.Request(); req != nil && req.Context() != nil {
		reqCtx = req.Context()
	}
	return &Context{
		Context:        c,
		requestContext: reqCtx,
	}
}

// FromContext adapts a standard request context for handlers registered through
// OpenAPI-aware transports that do not expose an Echo context directly.
func FromContext(ctx context.Context) *Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Context{requestContext: ctx}
}

func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c.requestContext != nil {
		return c.requestContext.Deadline()
	}
	return c.Request().Context().Deadline()
}

func (c *Context) Done() <-chan struct{} {
	if c.requestContext != nil {
		return c.requestContext.Done()
	}
	return c.Request().Context().Done()
}

func (c *Context) Err() error {
	if c.requestContext != nil {
		return c.requestContext.Err()
	}
	return c.Request().Context().Err()
}

func (c *Context) Value(key any) any {
	if c.requestContext != nil {
		return c.requestContext.Value(key)
	}
	return c.Request().Context().Value(key)
}

type HandlerFunc[Resp any] func(*Context) (Resp, error)

type ReqHandlerFunc[Req any, Resp any] func(
	*Context,
	Req,
) (Resp, error)

type ClaimsHandlerFunc[Claims any, Resp any] func(
	*Context,
	Claims,
) (Resp, error)

type ClaimsReqHandlerFunc[
	Req any,
	Claims any,
	Resp any,
] func(
	*Context,
	Req,
	Claims,
) (Resp, error)

func Wrap[Resp any](fn HandlerFunc[Resp]) echo.HandlerFunc {
	return func(c echo.Context) error {
		resp, err := fn(newContext(c))
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, resp)
	}
}

func WrapReq[Req any, Resp any](fn ReqHandlerFunc[Req, Resp]) echo.HandlerFunc {
	return func(c echo.Context) error {

		var req Req

		if err := Bind(c, &req); err != nil {
			return err
		}

		resp, err := fn(newContext(c), req)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	}
}

func WrapClaims[Claims any, Resp any](fn ClaimsHandlerFunc[Claims, Resp]) echo.HandlerFunc {
	return func(c echo.Context) error {

		claims, err := GetClaims[Claims](c)
		if err != nil {
			return err
		}

		resp, err := fn(newContext(c), claims)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	}
}

func WrapClaimsAndReq[
	Req any,
	Claims any,
	Resp any,
](fn ClaimsReqHandlerFunc[Req, Claims, Resp]) echo.HandlerFunc {

	return func(c echo.Context) error {

		var req Req

		if err := Bind(c, &req); err != nil {
			return err
		}

		claims, err := GetClaims[Claims](c)
		if err != nil {
			return err
		}

		resp, err := fn(newContext(c), req, claims)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, resp)
	}
}

func Bind(c echo.Context, req any) error {

	if err := c.Bind(req); err != nil {
		return err
	}

	if c.Echo().Validator != nil {
		if err := c.Validate(req); err != nil {
			return err
		}
	}

	return nil
}

func SetClaims[T any](c echo.Context, claims T) {
	c.Set(UserClaimsKey, claims)
}

func GetClaims[T any](c echo.Context) (T, error) {

	var claims T

	raw := c.Get(UserClaimsKey)
	if raw == nil {
		return claims, errors.New("claims not found")
	}

	v, ok := raw.(T)
	if !ok {
		return claims, errors.New("claims type mismatch")
	}

	return v, nil
}
