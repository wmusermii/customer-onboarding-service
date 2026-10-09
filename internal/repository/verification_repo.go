package repository

import (
	"context"
	"errors"
	"time"

	"customer-onboarding-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type VerificationResult struct {
	Status         domain.VerificationStatus
	FaceMatchScore *float64
	LivenessScore  *float64
	DukcapilMatch  *bool
	NIK            *string
	FullName       *string
	BirthDate      *time.Time
	Address        *string
	RejectReason   *string
}

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

// Lock: SELECT ... FOR UPDATE. Mengembalikan nil jika tidak ada.
func (r *VerificationRepo) Lock(ctx context.Context, db DBTX, id string) (*domain.IdentityVerification, error) {
	var v domain.IdentityVerification
	err := db.QueryRow(ctx, `
		SELECT verification_id, case_id, status::text
		FROM identity_verification
		WHERE verification_id = $1
		FOR UPDATE`, id).Scan(&v.VerificationID, &v.CaseID, &v.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &v, err
}

func (r *VerificationRepo) Complete(ctx context.Context, db DBTX, id string, res VerificationResult) error {
	_, err := db.Exec(ctx, `
		UPDATE identity_verification SET
			status = $2::verification_status,
			face_match_score = $3, liveness_score = $4, dukcapil_match = $5,
			extracted_nik = $6, extracted_full_name = $7,
			extracted_birth_date = $8, extracted_address = $9,
			reject_reason = $10, completed_at = now()
		WHERE verification_id = $1`,
		id, string(res.Status), res.FaceMatchScore, res.LivenessScore, res.DukcapilMatch,
		res.NIK, res.FullName, res.BirthDate, res.Address, res.RejectReason)
	return err
}

func (r *VerificationRepo) GetForCase(ctx context.Context, db DBTX, caseID, verificationID string) (*domain.IdentityVerification, error) {
	var v domain.IdentityVerification
	err := db.QueryRow(ctx, `
		SELECT verification_id, case_id, attempt_number, status::text,
		       face_match_score, liveness_score, dukcapil_match,
		       extracted_nik, extracted_full_name, extracted_birth_date, extracted_address,
		       reject_reason, created_at, completed_at
		FROM identity_verification
		WHERE verification_id = $1 AND case_id = $2`, verificationID, caseID).
		Scan(&v.VerificationID, &v.CaseID, &v.AttemptNumber, &v.Status,
			&v.FaceMatchScore, &v.LivenessScore, &v.DukcapilMatch,
			&v.ExtractedNIK, &v.ExtractedFullName, &v.ExtractedBirthDate, &v.ExtractedAddress,
			&v.RejectReason, &v.CreatedAt, &v.CompletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrVerificationNotFound
	}
	return &v, err
}
