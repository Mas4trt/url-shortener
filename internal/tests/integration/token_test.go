//go:build integration
// +build integration

package integration

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func issueTestAccessToken(secret string, appID uint64) (string, error) {
	claims := jwt.MapClaims{
		"app_id": float64(appID),
		"uid":    float64(1),
		"email":  "integration-test@example.com",
		"exp":    time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
