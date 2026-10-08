package handler

import (
	"net/http"

	"customer-onboarding-service/internal/middleware"
	"customer-onboarding-service/internal/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool, cases *CaseHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "db down", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})

	idem := repository.NewIdempotencyRepo()
	v1 := middleware.Chain(
		middleware.RequestID,
		middleware.Auth,
		middleware.RequireHeaders,
		middleware.Idempotency(idem, pool),
	)

	mux.Handle("POST /v1/onboarding/cases", v1(http.HandlerFunc(cases.Create)))

	return mux
}
