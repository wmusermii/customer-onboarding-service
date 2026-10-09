package handler

import (
	"io"
	"net/http"

	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/httpx"
	"customer-onboarding-service/internal/service"
)

const maxFormFileBytes = 9 << 20 // batas baca; batas per tipe dicek di service

type VerificationHandler struct{ svc *service.VerificationService }

func NewVerificationHandler(svc *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{svc: svc}
}

func readFormFile(r *http.Request, field string, required bool) ([]byte, error) {
	f, _, err := r.FormFile(field)
	if err != nil {
		if required {
			return nil, domain.ErrInvalidPayload.WithDetail(field + " is required")
		}
		return nil, nil
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxFormFileBytes+1))
	if err != nil {
		return nil, domain.ErrInvalidPayload.WithDetail("cannot read " + field)
	}
	if len(data) > maxFormFileBytes {
		return nil, domain.ErrInvalidFile.WithHTTP(http.StatusRequestEntityTooLarge).
			WithDetail(field + ": file too large")
	}
	return data, nil
}

func (h *VerificationHandler) UploadIdentity(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		httpx.Fail(w, r, domain.ErrInvalidPayload.WithDetail("expected multipart/form-data"))
		return
	}
	if side := r.FormValue("documentSide"); side != "" && side != "FRONT" {
		httpx.Fail(w, r, domain.ErrInvalidPayload.WithDetail("documentSide must be FRONT"))
		return
	}

	ktp, err := readFormFile(r, "ktpImage", true)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	selfie, err := readFormFile(r, "selfieImage", true)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	liveness, err := readFormFile(r, "livenessVideo", false)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	res, err := h.svc.UploadIdentity(r.Context(), service.UploadIdentityInput{
		CaseID:   r.PathValue("caseId"),
		Subject:  httpx.Subject(r.Context()),
		KTP:      ktp,
		Selfie:   selfie,
		Liveness: liveness,
	})
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	httpx.OK(w, r, http.StatusAccepted, map[string]any{
		"verificationId": res.VerificationID,
		"caseId":         res.CaseID,
		"status":         res.Status,
	})
}
