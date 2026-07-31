package dto

const (
	SuccessCode    = 200
	SuccessMessage = "success"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// SuccessResponse keeps the response payload concrete so OpenAPI clients can
// derive their types from the backend contract.
type SuccessResponse[T any] struct {
	Code    int    `json:"code" enum:"200"`
	Message string `json:"message" const:"success"`
	Data    T      `json:"data"`
}

func NewSuccessResponse(data any) Response {
	return Response{
		Code:    SuccessCode,
		Message: SuccessMessage,
		Data:    data,
	}
}

func NewTypedSuccessResponse[T any](data T) SuccessResponse[T] {
	return SuccessResponse[T]{
		Code:    SuccessCode,
		Message: SuccessMessage,
		Data:    data,
	}
}
