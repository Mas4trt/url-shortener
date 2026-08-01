package authn

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken = errors.New("authn: missing bearer token")
	ErrInvalidToken = errors.New("authn: invalid or expired token")

	// ErrMissingApplicationID/ErrMissingSecret guard against a specific
	// misconfiguration: if SSO_APPLICATION_ID is left unset, applicationID
	// defaults to 0. A token whose app_id claim is also absent/zero would
	// then pass the "app_id matches" check below — i.e. a config mistake
	// silently turns into an authentication bypass instead of a boot
	// failure. Refusing to construct a Verifier without both values closes
	// that gap.
	ErrMissingApplicationID = errors.New("authn: application id must be non-zero")
	ErrMissingSecret        = errors.New("authn: app secret must not be empty")
)

// Claims mirrors what sso puts in the access token (see sso's pkg/jwt).
type Claims struct {
	UserID uint64
	Email  string
}

type Verifier struct {
	secret        []byte
	applicationID uint64
}

// NewVerifier takes the app secret handed out when this service was
// provisioned in sso (`make new-app` in the sso repo) and the matching
// application_id. It returns an error rather than constructing a Verifier
// that would silently under-validate tokens.
func NewVerifier(secret string, applicationID uint64) (*Verifier, error) {
	if applicationID == 0 {
		return nil, ErrMissingApplicationID
	}
	if secret == "" {
		return nil, ErrMissingSecret
	}

	return &Verifier{secret: []byte(secret), applicationID: applicationID}, nil
}

func (v *Verifier) Verify(tokenStr string) (Claims, error) {
	if tokenStr == "" {
		return Claims{}, ErrMissingToken
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return Claims{}, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidToken
	}

	// Defense in depth: the secret already scopes the token to this app,
	// but check app_id explicitly too rather than trusting that alone.
	appID, _ := claims["app_id"].(float64)
	if uint64(appID) != v.applicationID {
		return Claims{}, ErrInvalidToken
	}

	uid, _ := claims["uid"].(float64)
	email, _ := claims["email"].(string)

	return Claims{UserID: uint64(uid), Email: email}, nil
}
