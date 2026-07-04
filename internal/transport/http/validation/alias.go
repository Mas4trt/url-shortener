package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var aliasRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func New() *validator.Validate {
	v := validator.New()

	v.RegisterValidation("alias", func(fl validator.FieldLevel) bool {
		return aliasRegexp.MatchString(fl.Field().String())
	})

	return v
}
