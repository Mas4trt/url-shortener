package authn_test

import (
	"testing"
	"time"

	"url-shortener/internal/authn"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testSecret = "test-signing-secret-value"
	testAppID  = uint64(42)
)

func signToken(t *testing.T, method jwt.SigningMethod, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token := jwt.NewWithClaims(method, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func TestNewVerifier(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		applicationID uint64
		wantErr       error
	}{
		{"valid", testSecret, testAppID, nil},
		{
			name:          "zero application id refused",
			secret:        testSecret,
			applicationID: 0,
			wantErr:       authn.ErrMissingApplicationID,
		},
		{
			name:          "empty secret refused",
			secret:        "",
			applicationID: testAppID,
			wantErr:       authn.ErrMissingSecret,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := authn.NewVerifier(tt.secret, tt.applicationID)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, v)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, v)
		})
	}
}

func TestVerifier_Verify(t *testing.T) {
	v, err := authn.NewVerifier(testSecret, testAppID)
	require.NoError(t, err)

	validClaims := jwt.MapClaims{
		"app_id": float64(testAppID),
		"uid":    float64(7),
		"email":  "user@example.com",
		"exp":    time.Now().Add(time.Hour).Unix(),
	}

	t.Run("valid token", func(t *testing.T) {
		tok := signToken(t, jwt.SigningMethodHS256, validClaims, testSecret)

		claims, err := v.Verify(tok)

		require.NoError(t, err)
		require.Equal(t, uint64(7), claims.UserID)
		require.Equal(t, "user@example.com", claims.Email)
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := v.Verify("")
		require.ErrorIs(t, err, authn.ErrMissingToken)
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := v.Verify("not-a-jwt")
		require.ErrorIs(t, err, authn.ErrInvalidToken)
	})

	t.Run("expired token", func(t *testing.T) {
		claims := jwt.MapClaims{
			"app_id": float64(testAppID),
			"uid":    float64(7),
			"exp":    time.Now().Add(-time.Hour).Unix(),
		}
		tok := signToken(t, jwt.SigningMethodHS256, claims, testSecret)

		_, err := v.Verify(tok)

		require.ErrorIs(t, err, authn.ErrInvalidToken)
	})

	t.Run("wrong signing secret", func(t *testing.T) {
		tok := signToken(t, jwt.SigningMethodHS256, validClaims, "a-different-secret")

		_, err := v.Verify(tok)

		require.ErrorIs(t, err, authn.ErrInvalidToken)
	})

	t.Run("wrong signing method rejected even with the right secret", func(t *testing.T) {
		tok := signToken(t, jwt.SigningMethodHS512, validClaims, testSecret)

		_, err := v.Verify(tok)

		require.ErrorIs(t, err, authn.ErrInvalidToken)
	})

	t.Run("app_id mismatch", func(t *testing.T) {
		claims := jwt.MapClaims{
			"app_id": float64(999),
			"uid":    float64(7),
			"exp":    time.Now().Add(time.Hour).Unix(),
		}
		tok := signToken(t, jwt.SigningMethodHS256, claims, testSecret)

		_, err := v.Verify(tok)

		require.ErrorIs(t, err, authn.ErrInvalidToken)
	})

	// Regression test for the exact bypass ErrMissingApplicationID exists
	// to prevent: a token with no app_id claim at all (appID defaults to
	// 0 via the type assertion) must not be treated as matching a
	// correctly configured, non-zero application ID.
	t.Run("missing app_id claim does not default to a match", func(t *testing.T) {
		claims := jwt.MapClaims{
			"uid": float64(7),
			"exp": time.Now().Add(time.Hour).Unix(),
		}
		tok := signToken(t, jwt.SigningMethodHS256, claims, testSecret)

		_, err := v.Verify(tok)

		require.ErrorIs(t, err, authn.ErrInvalidToken)
	})
}
