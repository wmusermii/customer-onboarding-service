package adapter

import (
	"context"
	"log"
	"time"

	"customer-onboarding-service/internal/event"
)

type EKYCRequest struct {
	VerificationID string
	CaseID         string
	KTPRef         string
	SelfieRef      string
	LivenessRef    string
}

type EKYCClient interface {
	Dispatch(ctx context.Context, req EKYCRequest) error
}

type EKYCHandler func(ctx context.Context, ev event.EKYCCompleted) error

// DummyEKYC mensimulasikan eKYC Service + publish event hasilnya.
type DummyEKYC struct {
	baseURL string
	apiKey  string
	result  string // VERIFIED | REJECTED | MANUAL
	delay   time.Duration
	handler EKYCHandler
}

func NewDummyEKYC(baseURL, apiKey, result string) *DummyEKYC {
	return &DummyEKYC{baseURL: baseURL, apiKey: apiKey, result: result, delay: 2 * time.Second}
}

// SetHandler: pengganti Kafka consumer untuk sementara.
func (d *DummyEKYC) SetHandler(h EKYCHandler) { d.handler = h }

func (d *DummyEKYC) Dispatch(_ context.Context, req EKYCRequest) error {
	log.Printf("[dummy-ekyc] dispatch verification=%s case=%s result=%s",
		req.VerificationID, req.CaseID, d.result)

	go func() {
		time.Sleep(d.delay)
		if d.handler == nil {
			return
		}
		if err := d.handler(context.Background(), d.buildEvent(req)); err != nil {
			log.Printf("[dummy-ekyc] handler error verification=%s: %v", req.VerificationID, err)
		}
	}()
	return nil
}

func ptr[T any](v T) *T { return &v }

func (d *DummyEKYC) buildEvent(req EKYCRequest) event.EKYCCompleted {
	ev := event.EKYCCompleted{VerificationID: req.VerificationID, CaseID: req.CaseID}
	switch d.result {
	case "REJECTED":
		ev.Status = "REJECTED"
		ev.RejectReason = "FACE_MISMATCH"
		ev.FaceMatchScore = ptr(0.42)
		ev.LivenessScore = ptr(0.91)
	case "MANUAL":
		ev.Status = "MANUAL_REVIEW_REQUIRED"
		ev.FaceMatchScore = ptr(0.72)
		ev.LivenessScore = ptr(0.90)
	default:
		ev.Status = "VERIFIED"
		ev.FaceMatchScore = ptr(0.97)
		ev.LivenessScore = ptr(0.95)
		ev.DukcapilMatch = ptr(true)
		ev.Extracted = &event.ExtractedData{
			NIK:       "3273011405900001",
			FullName:  "BUDI SANTOSO",
			BirthDate: "1990-05-14",
			Address:   "JL. MERDEKA NO. 10, BANDUNG",
		}
	}
	return ev
}
