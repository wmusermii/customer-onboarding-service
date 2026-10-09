package event

type ExtractedData struct {
	NIK       string `json:"nik"`
	FullName  string `json:"fullName"`
	BirthDate string `json:"birthDate"` // YYYY-MM-DD
	Address   string `json:"address"`
}

// EKYCCompleted = payload topic ekyc.verification.completed
type EKYCCompleted struct {
	VerificationID string         `json:"verificationId"`
	CaseID         string         `json:"caseId"`
	Status         string         `json:"status"` // VERIFIED | REJECTED | MANUAL_REVIEW_REQUIRED
	FaceMatchScore *float64       `json:"faceMatchScore,omitempty"`
	LivenessScore  *float64       `json:"livenessScore,omitempty"`
	DukcapilMatch  *bool          `json:"dukcapilMatch,omitempty"`
	RejectReason   string         `json:"rejectReason,omitempty"`
	Extracted      *ExtractedData `json:"extractedData,omitempty"`
}
