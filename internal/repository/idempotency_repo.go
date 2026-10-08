package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

type IdempotencyRecord struct {
	RequestHash    string
	ResponseStatus *int16 // nil = masih diproses
	ResponseBody   []byte
}

type IdempotencyRepo struct{}

func NewIdempotencyRepo() *IdempotencyRepo { return &IdempotencyRepo{} }

// TryStart: true jika key baru berhasil di-claim, false jika sudah ada.
func (r *IdempotencyRepo) TryStart(ctx context.Context, db DBTX, scope, key, hash string) (bool, error) {
	// buang record kadaluarsa untuk key yang sama
	if _, err := db.Exec(ctx, `
		DELETE FROM idempotency_key
		WHERE scope = $1 AND idempotency_key = $2::uuid AND expires_at < now()`,
		scope, key); err != nil {
		return false, err
	}
	tag, err := db.Exec(ctx, `
		INSERT INTO idempotency_key (scope, idempotency_key, request_hash)
		VALUES ($1, $2::uuid, $3)
		ON CONFLICT (scope, idempotency_key) DO NOTHING`,
		scope, key, hash)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *IdempotencyRepo) Get(ctx context.Context, db DBTX, scope, key string) (*IdempotencyRecord, error) {
	var rec IdempotencyRecord
	err := db.QueryRow(ctx, `
		SELECT request_hash, response_status, response_body
		FROM idempotency_key
		WHERE scope = $1 AND idempotency_key = $2::uuid`,
		scope, key).Scan(&rec.RequestHash, &rec.ResponseStatus, &rec.ResponseBody)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &rec, err
}

func (r *IdempotencyRepo) Complete(ctx context.Context, db DBTX, scope, key string, status int, body []byte) error {
	if !json.Valid(body) {
		body = []byte(`{}`)
	}
	_, err := db.Exec(ctx, `
		UPDATE idempotency_key
		SET response_status = $3, response_body = $4::jsonb
		WHERE scope = $1 AND idempotency_key = $2::uuid`,
		scope, key, status, string(body))
	return err
}

func (r *IdempotencyRepo) Delete(ctx context.Context, db DBTX, scope, key string) error {
	_, err := db.Exec(ctx,
		`DELETE FROM idempotency_key WHERE scope = $1 AND idempotency_key = $2::uuid`,
		scope, key)
	return err
}
