package middleware

import (
	"net/http"

	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/httpx"

	"github.com/google/uuid"
)

func RequireHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch := domain.Channel(r.Header.Get("X-Channel-Id"))
		switch ch {
		case domain.ChannelMobile, domain.ChannelCorporatePortal, domain.ChannelInternalTeller:
		default:
			httpx.Fail(w, r, domain.ErrInvalidPayload.WithDetail("X-Channel-Id must be MOBILE, CORPORATE_PORTAL or INTERNAL_TELLER"))
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if _, err := uuid.Parse(r.Header.Get("X-Idempotency-Key")); err != nil {
				httpx.Fail(w, r, domain.ErrInvalidPayload.WithDetail("X-Idempotency-Key must be a UUID"))
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(httpx.WithChannel(r.Context(), string(ch))))
	})
}
