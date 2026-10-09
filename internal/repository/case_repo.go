package repository

import (
	"context"
	"errors"
	"time"

	"customer-onboarding-service/internal/domain"

	"github.com/jackc/pgx/v5"
)

type NewCase struct {
	CustomerID     string
	CustomerType   domain.CustomerType
	Subject        string
	ApplicantName  string
	ApplicantPhone string
	ApplicantEmail *string
	Channel        domain.Channel
	ExpiresAt      time.Time
}

type CaseRepo struct{}

func NewCaseRepo() *CaseRepo { return &CaseRepo{} }

const caseColumns = `
	case_id, customer_id::text, customer_type::text, created_by_subject,
	applicant_name, applicant_phone, applicant_email, channel::text,
	current_stage::text, saga_state::text, risk_level::text, version,
	expires_at, created_at, updated_at`

func scanCase(row pgx.Row) (*domain.OnboardingCase, error) {
	var c domain.OnboardingCase
	err := row.Scan(
		&c.CaseID, &c.CustomerID, &c.CustomerType, &c.CreatedBySubject,
		&c.ApplicantName, &c.ApplicantPhone, &c.ApplicantEmail, &c.Channel,
		&c.CurrentStage, &c.SagaState, &c.RiskLevel, &c.Version,
		&c.ExpiresAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrCaseNotFound
	}
	return &c, err
}

func (r *CaseRepo) Insert(ctx context.Context, db DBTX, in NewCase) (*domain.OnboardingCase, error) {
	row := db.QueryRow(ctx, `
		INSERT INTO onboarding_case
			(customer_id, customer_type, created_by_subject, applicant_name,
			 applicant_phone, applicant_email, channel, expires_at)
		VALUES ($1::uuid, $2::customer_type, $3, $4, $5, $6, $7::onboarding_channel, $8)
		RETURNING `+caseColumns,
		in.CustomerID, string(in.CustomerType), in.Subject, in.ApplicantName,
		in.ApplicantPhone, in.ApplicantEmail, string(in.Channel), in.ExpiresAt,
	)
	return scanCase(row)
}

func (r *CaseRepo) GetByID(ctx context.Context, db DBTX, caseID string) (*domain.OnboardingCase, error) {
	row := db.QueryRow(ctx,
		`SELECT `+caseColumns+` FROM onboarding_case WHERE case_id = $1`, caseID)
	return scanCase(row)
}

// =========================================== This for transaction start =====================
func (r *CaseRepo) SetSagaState(ctx context.Context, db DBTX, caseID string, s domain.SagaState) error {
	_, err := db.Exec(ctx, `
		UPDATE onboarding_case
		SET saga_state = $2::saga_state, version = version + 1, updated_at = now()
		WHERE case_id = $1`, caseID, string(s))
	return err
}
