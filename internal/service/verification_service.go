package service

import (
	"context"
	"log"
	"time"

	"customer-onboarding-service/internal/adapter"
	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/repository"
)

const maxVerificationAttempts = 3

type VerificationService struct {
	db       repository.DBTX
	tx       *repository.TxManager
	cases    *repository.CaseRepo
	verifs   *repository.VerificationRepo
	docs     *repository.DocumentRepo
	eventLog *repository.EventLogRepo
	store    adapter.DocumentStore
	ekyc     adapter.EKYCClient
}

func NewVerificationService(
	db repository.DBTX,
	tx *repository.TxManager,
	cases *repository.CaseRepo,
	verifs *repository.VerificationRepo,
	docs *repository.DocumentRepo,
	eventLog *repository.EventLogRepo,
	store adapter.DocumentStore,
	ekyc adapter.EKYCClient,
) *VerificationService {
	return &VerificationService{db: db, tx: tx, cases: cases, verifs: verifs,
		docs: docs, eventLog: eventLog, store: store, ekyc: ekyc}
}

type UploadIdentityInput struct {
	CaseID   string
	Subject  string
	KTP      []byte
	Selfie   []byte
	Liveness []byte // opsional
}

type UploadIdentityResult struct {
	VerificationID string
	CaseID         string
	Status         domain.VerificationStatus
}

func (s *VerificationService) UploadIdentity(ctx context.Context, in UploadIdentityInput) (*UploadIdentityResult, error) {
	// 1. Case harus ada, milik pemanggil, belum expired
	c, err := s.cases.GetByID(ctx, s.db, in.CaseID)
	if err != nil {
		return nil, err
	}
	if c.CreatedBySubject != in.Subject ||
		(c.ExpiresAt != nil && time.Now().After(*c.ExpiresAt)) {
		return nil, domain.ErrCaseNotFound
	}
	if c.CurrentStage != domain.StageInitiated {
		return nil, domain.ErrCaseStateInvalid
	}

	// 2. Batas percobaan
	maxAttempt, processing, err := s.verifs.AttemptInfo(ctx, s.db, in.CaseID)
	if err != nil {
		return nil, err
	}
	if processing {
		return nil, domain.ErrVerificationInProgress
	}
	if maxAttempt >= maxVerificationAttempts {
		return nil, domain.ErrMaxAttempts
	}

	// 3. Validasi file (percobaan belum terpakai kalau file ditolak)
	ktp, ktpExt, err := sanitizeImage("ktpImage", in.KTP)
	if err != nil {
		return nil, err
	}
	selfie, selfieExt, err := sanitizeImage("selfieImage", in.Selfie)
	if err != nil {
		return nil, err
	}
	if len(in.Liveness) > 0 {
		if err := validateVideo("livenessVideo", in.Liveness); err != nil {
			return nil, err
		}
	}

	// 4. Simpan ke Document Management Service
	ktpRef, err := s.store.Put(ctx, in.CaseID, "KTP", ktpExt, ktp)
	if err != nil {
		return nil, err
	}
	selfieRef, err := s.store.Put(ctx, in.CaseID, "SELFIE", selfieExt, selfie)
	if err != nil {
		return nil, err
	}
	var liveRef string
	if len(in.Liveness) > 0 {
		if liveRef, err = s.store.Put(ctx, in.CaseID, "LIVENESS_VIDEO", "mp4", in.Liveness); err != nil {
			return nil, err
		}
	}

	// 5. Satu transaksi: verification + document_ref + saga_state + event log
	var v *domain.IdentityVerification
	err = s.tx.Do(ctx, func(ctx context.Context, db repository.DBTX) error {
		ins, err := s.verifs.InsertApplicant(ctx, db, in.CaseID, maxAttempt+1)
		if err != nil {
			return err
		}
		v = ins

		docs := []struct{ docType, ref string }{{"KTP", ktpRef}, {"SELFIE", selfieRef}}
		if liveRef != "" {
			docs = append(docs, struct{ docType, ref string }{"LIVENESS_VIDEO", liveRef})
		}
		for _, d := range docs {
			if err := s.docs.Insert(ctx, db, in.CaseID, v.VerificationID, d.docType, d.ref); err != nil {
				return err
			}
		}
		if err := s.cases.SetSagaState(ctx, db, in.CaseID, domain.SagaWaitingExternal); err != nil {
			return err
		}
		return s.eventLog.Append(ctx, db, in.CaseID, "IDENTITY_VERIFICATION_STARTED", map[string]any{
			"verificationId": v.VerificationID,
			"attempt":        v.AttemptNumber,
		})
	})
	if err != nil {
		return nil, err
	}

	// 6. Dispatch ke eKYC SETELAH commit
	if err := s.ekyc.Dispatch(ctx, adapter.EKYCRequest{
		VerificationID: v.VerificationID,
		CaseID:         in.CaseID,
		KTPRef:         ktpRef,
		SelfieRef:      selfieRef,
		LivenessRef:    liveRef,
	}); err != nil {
		log.Printf("ekyc dispatch failed verification=%s: %v", v.VerificationID, err)
		s.verifs.MarkManualReview(context.WithoutCancel(ctx), s.db, v.VerificationID)
		return nil, domain.ErrEKYCDown
	}

	return &UploadIdentityResult{
		VerificationID: v.VerificationID,
		CaseID:         in.CaseID,
		Status:         v.Status,
	}, nil
}
