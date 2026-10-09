package domain

import "net/http"

type AppError struct {
	Code    string
	Message string
	HTTP    int
	Details []string
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

func NewError(httpStatus int, code, msg string, details ...string) *AppError {
	return &AppError{Code: code, Message: msg, HTTP: httpStatus, Details: details}
}

var (
	ErrInvalidPayload         = NewError(http.StatusBadRequest, "ONB-4001", "Invalid payload or missing required field")
	ErrCaseNotFound           = NewError(http.StatusNotFound, "ONB-4041", "Onboarding case not found")
	ErrIdempotencyClash       = NewError(http.StatusConflict, "ONB-4090", "Idempotency key already used with different payload")
	ErrInvalidFile            = NewError(http.StatusRequestEntityTooLarge, "ONB-4131", "Invalid image size or format")
	ErrEKYCRejected           = NewError(http.StatusUnprocessableEntity, "ONB-4221", "Face match or liveness failed")
	ErrEKYCDown               = NewError(http.StatusBadGateway, "ONB-5001", "eKYC service unreachable")
	ErrUnauthorized           = NewError(http.StatusUnauthorized, "ONB-4010", "Unauthorized")
	ErrInternal               = NewError(http.StatusInternalServerError, "ONB-5000", "Internal server error")
	ErrIdempotencyInProgress  = NewError(http.StatusConflict, "ONB-4091", "A request with this idempotency key is still being processed")
	ErrCaseStateInvalid       = NewError(http.StatusConflict, "ONB-4092", "Case state does not allow this operation")
	ErrVerificationInProgress = NewError(http.StatusConflict, "ONB-4093", "A verification is already in progress for this case")
	ErrMaxAttempts            = NewError(http.StatusUnprocessableEntity, "ONB-4222", "Maximum verification attempts reached, continue with assisted verification (video call)")
	ErrVerificationNotFound   = NewError(http.StatusNotFound, "ONB-4042", "Verification not found")
)

func (e *AppError) WithDetail(d string) *AppError {
	c := *e
	c.Details = []string{d}
	return &c
}
func (e *AppError) WithHTTP(status int) *AppError {
	c := *e
	c.HTTP = status
	return &c
}
