// Package authn verifies access tokens issued by sso without calling sso
// on every request: sso signs access tokens (HS256) with the same secret
// this app was provisioned with, so we can check the signature locally.
package authn

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrMissingToken = errors.New("authn: missing bearer token")
	ErrInvalidToken = errors.New("authn: invalid or expired token")
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
// application_id.
func NewVerifier(secret string, applicationID uint64) *Verifier {
	return &Verifier{secret: []byte(secret), applicationID: applicationID}
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
