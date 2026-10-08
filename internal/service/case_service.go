package service

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/event"
	"customer-onboarding-service/internal/repository"
)

const caseTTL = 7 * 24 * time.Hour

var phoneE164 = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

type CaseService struct {
	tx        *repository.TxManager
	customers *repository.CustomerRepo
	cases     *repository.CaseRepo
	eventLog  *repository.EventLogRepo
	outbox    *repository.OutboxRepo
}

func NewCaseService(
	tx *repository.TxManager,
	customers *repository.CustomerRepo,
	cases *repository.CaseRepo,
	eventLog *repository.EventLogRepo,
	outbox *repository.OutboxRepo,
) *CaseService {
	return &CaseService{tx: tx, customers: customers, cases: cases, eventLog: eventLog, outbox: outbox}
}

type CreateCaseInput struct {
	Type    domain.CustomerType
	Channel domain.Channel
	Name    string
	Phone   string
	Email   string
	Subject string // claim "sub" dari JWT
}

func invalid(detail string) *domain.AppError {
	return domain.NewError(http.StatusBadRequest, "ONB-4001",
		"Invalid payload or missing required field", detail)
}

func (in CreateCaseInput) validate() error {
	switch in.Type {
	case domain.CustomerIndividual, domain.CustomerCorporate:
	default:
		return invalid("type must be INDIVIDUAL or CORPORATE")
	}
	switch in.Channel {
	case domain.ChannelMobile, domain.ChannelCorporatePortal, domain.ChannelInternalTeller:
	default:
		return invalid("channel must be MOBILE, CORPORATE_PORTAL or INTERNAL_TELLER")
	}
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 150 {
		return invalid("applicant.fullName is required (max 150)")
	}
	if !phoneE164.MatchString(in.Phone) {
		return invalid("applicant.phoneNumber must be E.164, e.g. +6281234567890")
	}
	if in.Subject == "" {
		return invalid("missing subject")
	}
	return nil
}

func (s *CaseService) CreateCase(ctx context.Context, in CreateCaseInput) (*domain.OnboardingCase, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}

	var email *string
	if e := strings.TrimSpace(in.Email); e != "" {
		email = &e
	}

	var created *domain.OnboardingCase
	err := s.tx.Do(ctx, func(ctx context.Context, db repository.DBTX) error {
		customerID, err := s.customers.InsertPending(ctx, db, in.Type)
		if err != nil {
			return err
		}

		c, err := s.cases.Insert(ctx, db, repository.NewCase{
			CustomerID:     customerID,
			CustomerType:   in.Type,
			Subject:        in.Subject,
			ApplicantName:  strings.TrimSpace(in.Name),
			ApplicantPhone: in.Phone,
			ApplicantEmail: email,
			Channel:        in.Channel,
			ExpiresAt:      time.Now().Add(caseTTL),
		})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"caseId":       c.CaseID,
			"customerType": c.CustomerType,
			"channel":      c.Channel,
		}
		if err := s.eventLog.Append(ctx, db, c.CaseID, "CASE_CREATED", payload); err != nil {
			return err
		}
		if err := s.outbox.Insert(ctx, db, event.TopicCaseCreated, c.CaseID, c.CaseID, payload); err != nil {
			return err
		}

		created = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}
