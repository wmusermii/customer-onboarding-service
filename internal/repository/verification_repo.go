package repository

import (
	"context"
	"errors"

	"customer-onboarding-service/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

type VerificationRepo struct{}

func NewVerificationRepo() *VerificationRepo { return &VerificationRepo{} }

// AttemptInfo: attempt tertinggi dan apakah ada yang masih PROCESSING.
func (r *VerificationRepo) AttemptInfo(ctx context.Context, db DBTX, caseID string) (maxAttempt int, processing bool, err error) {
	err = db.QueryRow(ctx, `
		SELECT COALESCE(MAX(attempt_number), 0)::int,
		       COALESCE(bool_or(status = 'PROCESSING'), false)
		FROM identity_verification
		WHERE case_id = $1 AND subject_type = 'APPLICANT'`, caseID).
		Scan(&maxAttempt, &processing)
	return
}

func (r *VerificationRepo) InsertApplicant(ctx context.Context, db DBTX, caseID string, attempt int) (*domain.IdentityVerification, error) {
	var v domain.IdentityVerification
	err := db.QueryRow(ctx, `
		INSERT INTO identity_verification (case_id, attempt_number)
		VALUES ($1, $2)
		RETURNING verification_id, case_id, attempt_number, status::text, created_at`,
		caseID, attempt).
		Scan(&v.VerificationID, &v.CaseID, &v.AttemptNumber, &v.Status, &v.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return nil, domain.ErrVerificationInProgress
	}
	return &v, err
}

func (r *VerificationRepo) MarkManualReview(ctx context.Context, db DBTX, verificationID string) error {
	_, err := db.Exec(ctx, `
		UPDATE identity_verification
		SET status = 'MANUAL_REVIEW_REQUIRED', updated_at = now()
		WHERE verification_id = $1`, verificationID)
	return err
}
