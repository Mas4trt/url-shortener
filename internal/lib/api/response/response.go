package response

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

const (
	StatusOk   = "OK"
	StatusEror = "error"
)

type Response struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func Ok() Response {
	return Response{
		Status: StatusOk,
	}
}

func Error(msg string) Response {
	return Response{
		Status: StatusEror,
		Error:  msg,
	}
}

func ValidationError(errs validator.ValidationErrors) Response {
	var errMsqs []string

	for _, err := range errs {
		switch err.ActualTag() {
		case "required":
			errMsqs = append(errMsqs, fmt.Sprintf("field %s is a required field", err.Field()))
		case "url":
			errMsqs = append(errMsqs, fmt.Sprintf("field %s is not valid URL", err.Field()))
		default:
			errMsqs = append(errMsqs, fmt.Sprintf("field %s is not valid", err.Field()))
		}
	}

	return Response{
		Status: StatusEror,
		Error:  strings.Join(errMsqs, ", "),
	}
}
