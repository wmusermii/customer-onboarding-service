package adapter

import (
	"context"
	"log"
)

type EKYCRequest struct {
	VerificationID string
	CaseID         string
	KTPRef         string
	SelfieRef      string
	LivenessRef    string // kosong jika tidak ada video
}

type EKYCClient interface {
	// Dispatch fire-and-forget; hasil datang lewat event ekyc.verification.completed.
	Dispatch(ctx context.Context, req EKYCRequest) error
}

type DummyEKYC struct {
	baseURL string
	apiKey  string
}

func NewDummyEKYC(baseURL, apiKey string) *DummyEKYC {
	return &DummyEKYC{baseURL: baseURL, apiKey: apiKey}
}

func (d *DummyEKYC) Dispatch(_ context.Context, req EKYCRequest) error {
	log.Printf("[dummy-ekyc] dispatch verification=%s case=%s ktp=%s selfie=%s",
		req.VerificationID, req.CaseID, req.KTPRef, req.SelfieRef)
	return nil
}
