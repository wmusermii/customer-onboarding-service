package middleware

import (
	"net/http"
	"strings"

	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/httpx"
)

// DUMMY: token format "Bearer dummy-<subject>", mis. "Bearer dummy-user-001".
// TODO: ganti dengan validasi JWT (CIAM/Staff IAM) via JWKS.
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		sub, ok2 := strings.CutPrefix(token, "dummy-")
		if !ok || !ok2 || sub == "" {
			httpx.Fail(w, r, domain.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(httpx.WithSubject(r.Context(), sub)))
	})
}
