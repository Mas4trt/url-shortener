package auth

import (
	"context"
	"net/http"
	"strings"

	"url-shortener/internal/authn"
	"url-shortener/internal/transport/http/response"
)

type ctxKey struct{}

// RequireAuth rejects requests without a valid "Authorization: Bearer
// <token>" header and attaches the resulting claims to the request context.
func RequireAuth(verifier *authn.Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "

			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, prefix) {
				response.Respond(w, r, http.StatusUnauthorized, response.Error("missing bearer token"))
				return
			}

			claims, err := verifier.Verify(strings.TrimPrefix(header, prefix))
			if err != nil {
				response.Respond(w, r, http.StatusUnauthorized, response.Error("invalid or expired token"))
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)))
		})
	}
}

// User returns the authenticated user's claims from a request context
// populated by RequireAuth.
func User(ctx context.Context) (authn.Claims, bool) {
	claims, ok := ctx.Value(ctxKey{}).(authn.Claims)
	return claims, ok
}
