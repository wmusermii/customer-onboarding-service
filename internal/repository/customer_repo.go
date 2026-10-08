package repository

import (
	"context"

	"customer-onboarding-service/internal/domain"
)

type CustomerRepo struct{}

func NewCustomerRepo() *CustomerRepo { return &CustomerRepo{} }

// InsertPending membuat customer berstatus PENDING, mengembalikan customer_id.
func (r *CustomerRepo) InsertPending(ctx context.Context, db DBTX, t domain.CustomerType) (string, error) {
	var id string
	err := db.QueryRow(ctx,
		`INSERT INTO customer (type) VALUES ($1::customer_type) RETURNING customer_id::text`,
		string(t),
	).Scan(&id)
	return id, err
}
