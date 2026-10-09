package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/event"
	"customer-onboarding-service/internal/repository"
)

func toResult(ev event.EKYCCompleted) (repository.VerificationResult, error) {
	r := repository.VerificationResult{
		Status:         domain.VerificationStatus(ev.Status),
		FaceMatchScore: ev.FaceMatchScore,
		LivenessScore:  ev.LivenessScore,
		DukcapilMatch:  ev.DukcapilMatch,
	}

	switch r.Status {
	case domain.VerVerified:
		if ev.FaceMatchScore == nil || ev.LivenessScore == nil || ev.DukcapilMatch == nil || ev.Extracted == nil {
			return r, fmt.Errorf("incomplete VERIFIED event for %s", ev.VerificationID)
		}
	case domain.VerRejected:
		if ev.RejectReason == "" {
			return r, fmt.Errorf("REJECTED event without reason for %s", ev.VerificationID)
		}
	case domain.VerManualReview:
	default:
		return r, fmt.Errorf("unknown ekyc status %q", ev.Status)
	}

	if ev.RejectReason != "" {
		rr := ev.RejectReason
		r.RejectReason = &rr
	}
	if d := ev.Extracted; d != nil {
		nik, name, addr := d.NIK, d.FullName, d.Address
		r.NIK, r.FullName, r.Address = &nik, &name, &addr
		if d.BirthDate != "" {
			t, err := time.Parse("2006-01-02", d.BirthDate)
			if err != nil {
				return r, fmt.Errorf("invalid birthDate: %w", err)
			}
			r.BirthDate = &t
		}
	}
	return r, nil
}

// HandleEKYCCompleted dipanggil untuk event ekyc.verification.completed.
// Aman dipanggil berulang (at-least-once): event duplikat diabaikan.
func (s *VerificationService) HandleEKYCCompleted(ctx context.Context, ev event.EKYCCompleted) error {
	res, err := toResult(ev)
	if err != nil {
		return err
	}

	return s.tx.Do(ctx, func(ctx context.Context, db repository.DBTX) error {
		v, err := s.verifs.Lock(ctx, db, ev.VerificationID)
		if err != nil {
			return err
		}
		if v == nil || v.CaseID != ev.CaseID {
			log.Printf("ekyc event ignored: unknown verification %s", ev.VerificationID)
			return nil
		}
		if v.Status != domain.VerProcessing {
			return nil // duplikat
		}

		if err := s.verifs.Complete(ctx, db, v.VerificationID, res); err != nil {
			return err
		}

		switch res.Status {
		case domain.VerVerified:
			if err := s.docs.SetStatusByVerification(ctx, db, v.VerificationID, domain.ReviewVerified); err != nil {
				return err
			}
			if err := s.cases.AdvanceStage(ctx, db, ev.CaseID,
				domain.StageInitiated, domain.StageIdentityVerified, domain.SagaInProgress); err != nil {
				return err
			}
			if err := s.eventLog.Append(ctx, db, ev.CaseID, "IDENTITY_VERIFIED",
				map[string]any{"verificationId": v.VerificationID}); err != nil {
				return err
			}
			return s.outbox.Insert(ctx, db, event.TopicCaseStatusChanged, ev.CaseID, ev.CaseID,
				map[string]any{
					"caseId":        ev.CaseID,
					"previousStage": domain.StageInitiated,
					"newStage":      domain.StageIdentityVerified,
				})

		case domain.VerRejected:
			if err := s.docs.SetStatusByVerification(ctx, db, v.VerificationID, domain.ReviewRejected); err != nil {
				return err
			}
			// stage tetap INITIATED, nasabah boleh upload ulang (maks 3x)
			if err := s.cases.SetSagaState(ctx, db, ev.CaseID, domain.SagaStarted); err != nil {
				return err
			}
			return s.eventLog.Append(ctx, db, ev.CaseID, "IDENTITY_REJECTED",
				map[string]any{"verificationId": v.VerificationID, "reason": ev.RejectReason})

		default: // MANUAL_REVIEW_REQUIRED
			return s.eventLog.Append(ctx, db, ev.CaseID, "IDENTITY_MANUAL_REVIEW",
				map[string]any{"verificationId": v.VerificationID})
		}
	})
}
