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

func NewSuccessResponse(data any) Response {
	return Response{
		Code:    SuccessCode,
		Message: SuccessMessage,
		Data:    data,
	}
}
