package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/httpx"
	"customer-onboarding-service/internal/repository"
)

const maxBodyBytes = 20 << 20 // 20 MB (3 file eKYC @ max 5 MB + overhead)

type recorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}

func Idempotency(repo *repository.IdempotencyRepo, db repository.DBTX) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}

			ctx := r.Context()
			key := r.Header.Get("X-Idempotency-Key") // sudah divalidasi UUID oleh RequireHeaders
			scope := r.Method + " " + r.URL.Path + " " + httpx.Subject(ctx)

			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
			if err != nil {
				var tooBig *http.MaxBytesError
				if errors.As(err, &tooBig) {
					httpx.Fail(w, r, domain.ErrInvalidFile)
				} else {
					httpx.Fail(w, r, domain.ErrInvalidPayload)
				}
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			sum := sha256.Sum256(body)
			hash := hex.EncodeToString(sum[:])

			started, err := repo.TryStart(ctx, db, scope, key, hash)
			if err != nil {
				httpx.Fail(w, r, err)
				return
			}

			// Key sudah pernah dipakai -> replay atau tolak
			if !started {
				rec, err := repo.Get(ctx, db, scope, key)
				if err != nil || rec == nil {
					httpx.Fail(w, r, domain.ErrInternal)
					return
				}
				if rec.RequestHash != hash {
					httpx.Fail(w, r, domain.ErrIdempotencyClash)
					return
				}
				if rec.ResponseStatus == nil {
					httpx.Fail(w, r, domain.ErrIdempotencyInProgress)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Idempotent-Replayed", "true")
				w.WriteHeader(int(*rec.ResponseStatus))
				w.Write(rec.ResponseBody)
				return
			}

			// Request pertama: jalankan handler, simpan hasilnya
			finished := false
			defer func() {
				if !finished { // panic / kondisi tak terduga
					repo.Delete(context.WithoutCancel(ctx), db, scope, key)
				}
			}()

			rw := &recorder{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			bg := context.WithoutCancel(ctx)
			if rw.status == 0 || rw.status >= 500 {
				// error server tidak di-cache, supaya client boleh retry
				repo.Delete(bg, db, scope, key)
			} else {
				repo.Complete(bg, db, scope, key, rw.status, rw.buf.Bytes())
			}
			finished = true
		})
	}
}
