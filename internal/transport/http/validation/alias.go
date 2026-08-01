package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

var aliasRegexp = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// New builds the shared validator instance, registering this service's
// custom tags. Called once at startup (see bootstrap.provideValidator)
// and reused for every request — validator.Validate is safe for
// concurrent use once its custom validations are registered.
func New() (*validator.Validate, error) {
	v := validator.New()

	if err := v.RegisterValidation("alias", func(fl validator.FieldLevel) bool {
		return aliasRegexp.MatchString(fl.Field().String())
	}); err != nil {
		return nil, err
	}

	if err := registerURLScheme(v); err != nil {
		return nil, err
	}

	return v, nil
}
