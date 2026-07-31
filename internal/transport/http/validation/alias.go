package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var aliasRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func New() (*validator.Validate, error) {
	v := validator.New()

	if err := v.RegisterValidation("alias", func(fl validator.FieldLevel) bool {
		return aliasRegexp.MatchString(fl.Field().String())
	}); err != nil {
		return nil, err
	}

	return v, nil
}
