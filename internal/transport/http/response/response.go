package response

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

const (
	StatusOk    = "OK"
	StatusError = "Error"
)

type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Response struct {
	Status  string                  `json:"status"`
	Error   string                  `json:"error,omitempty"`
	Details []ValidationErrorDetail `json:"details,omitempty"`
}

func Ok() Response {
	return Response{
		Status: StatusOk,
	}
}

func Error(msg string) Response {
	return Response{
		Status: StatusError,
		Error:  msg,
	}
}

func ValidationError(errs validator.ValidationErrors) Response {
	var details []ValidationErrorDetail

	for _, err := range errs {
		details = append(details, ValidationErrorDetail{
			Field:   strings.ToLower(err.Field()),
			Message: fmt.Sprintf("field %s is invalid", err.Field()),
		})
	}

	return Response{
		Status:  StatusError,
		Error:   "validation failed",
		Details: details,
	}
}

// Respond — универсальная обертка для отправки JSON-ответа
func Respond(w http.ResponseWriter, r *http.Request, code int, v any) {
	render.Status(r, code)
	render.JSON(w, r, v)
}
