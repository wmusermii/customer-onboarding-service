package repository

import (
	"context"
	"encoding/json"
)

type OutboxRepo struct{}

func NewOutboxRepo() *OutboxRepo { return &OutboxRepo{} }

// Insert menulis event ke outbox dalam transaksi yang sama dengan perubahan data.
func (r *OutboxRepo) Insert(ctx context.Context, db DBTX, topic, aggregateID, key string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
		INSERT INTO outbox_event (aggregate_id, topic, event_key, payload)
		VALUES ($1, $2, $3, $4::jsonb)`,
		aggregateID, topic, key, string(b))
	return err
}
