package repository

import (
	"context"
	"encoding/json"
)

type EventLogRepo struct{}

func NewEventLogRepo() *EventLogRepo { return &EventLogRepo{} }

// Append menulis audit trail (append-only) ke case_event_log.
func (r *EventLogRepo) Append(ctx context.Context, db DBTX, caseID, eventType string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `
		INSERT INTO case_event_log (case_id, event_type, payload_json)
		VALUES ($1, $2, $3::jsonb)`,
		caseID, eventType, string(b))
	return err
}
