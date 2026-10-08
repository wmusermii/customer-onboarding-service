package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"customer-onboarding-service/internal/domain"
)

type meta struct {
	RequestID string `json:"requestId"`
}

type successBody struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
	Meta    meta `json:"meta"`
}

type errorDetail struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details"`
}

type errorBody struct {
	Success bool        `json:"success"`
	Error   errorDetail `json:"error"`
	Meta    meta        `json:"meta"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func OK(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeJSON(w, status, successBody{
		Success: true,
		Data:    data,
		Meta:    meta{RequestID: RequestID(r.Context())},
	})
}

func Fail(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		log.Printf("[%s] internal error: %v", RequestID(r.Context()), err)
		appErr = domain.ErrInternal
	}
	details := appErr.Details
	if details == nil {
		details = []string{}
	}
	writeJSON(w, appErr.HTTP, errorBody{
		Success: false,
		Error:   errorDetail{Code: appErr.Code, Message: appErr.Message, Details: details},
		Meta:    meta{RequestID: RequestID(r.Context())},
	})
}
