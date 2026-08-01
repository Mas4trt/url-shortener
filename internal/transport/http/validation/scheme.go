package validation

import (
	"net/url"

	"github.com/go-playground/validator/v10"
)

// allowedSchemes restricts what this service will store and later hand
// back in a redirect's Location header. Without this, `url:"required,url"`
// alone accepts any URI scheme go-playground/validator considers
// well-formed — including javascript:, data:, and file: — any of which
// becomes attacker-controlled content in a 302 response once stored.
var allowedSchemes = map[string]bool{
	"http":  true,
	"https": true,
}

func registerURLScheme(v *validator.Validate) error {
	return v.RegisterValidation("url_scheme", func(fl validator.FieldLevel) bool {
		u, err := url.Parse(fl.Field().String())
		if err != nil {
			return false
		}
		return allowedSchemes[u.Scheme]
	})
}
