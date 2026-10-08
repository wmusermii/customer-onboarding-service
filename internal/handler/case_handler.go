package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"customer-onboarding-service/internal/domain"
	"customer-onboarding-service/internal/httpx"
	"customer-onboarding-service/internal/service"
)

var wib = time.FixedZone("WIB", 7*3600)

type CaseHandler struct{ svc *service.CaseService }

func NewCaseHandler(svc *service.CaseService) *CaseHandler { return &CaseHandler{svc: svc} }

type createCaseRequest struct {
	Type      string `json:"type"`
	Channel   string `json:"channel"`
	Applicant struct {
		FullName    string `json:"fullName"`
		PhoneNumber string `json:"phoneNumber"`
		Email       string `json:"email"`
	} `json:"applicant"`
}

func (h *CaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCaseRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.Fail(w, r, domain.ErrInvalidPayload.WithDetail("malformed JSON body"))
		return
	}

	if req.Channel != httpx.Channel(r.Context()) {
		httpx.Fail(w, r, domain.ErrInvalidPayload.WithDetail("channel must match X-Channel-Id"))
		return
	}

	c, err := h.svc.CreateCase(r.Context(), service.CreateCaseInput{
		Type:    domain.CustomerType(req.Type),
		Channel: domain.Channel(req.Channel),
		Name:    req.Applicant.FullName,
		Phone:   req.Applicant.PhoneNumber,
		Email:   req.Applicant.Email,
		Subject: httpx.Subject(r.Context()),
	})
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	httpx.OK(w, r, http.StatusCreated, map[string]any{
		"caseId":    c.CaseID,
		"status":    c.CurrentStage.APIStatus(),
		"createdAt": c.CreatedAt.In(wib).Format(time.RFC3339),
	})
}
