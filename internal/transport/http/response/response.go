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
	details := make([]ValidationErrorDetail, 0, len(errs))

	for _, err := range errs {
		details = append(details, ValidationErrorDetail{
			Field:   strings.ToLower(err.Field()),
			Message: validationMessage(err),
		})
	}

	return Response{
		Status:  StatusError,
		Error:   "validation failed",
		Details: details,
	}
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"

	case "url":
		return "must be a valid URL"

	case "min":
		return fmt.Sprintf("must be at least %s characters", err.Param())

	case "max":
		return fmt.Sprintf("must be at most %s characters", err.Param())

	case "alias":
		return "may contain only letters, numbers, '-' and '_'"

	default:
		return "is invalid"
	}
}

// Respond — универсальная обертка для отправки JSON-ответа
func Respond(w http.ResponseWriter, r *http.Request, code int, v any) {
	render.Status(r, code)
	render.JSON(w, r, v)
}
