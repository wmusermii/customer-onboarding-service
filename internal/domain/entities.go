package domain

import "time"

type Customer struct {
	CustomerID string
	Type       CustomerType
	Status     CustomerStatus
	RiskRating *RiskLevel
	CIFNumber  *string
	NIKHash    *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CorporateEntity struct {
	CompanyID   string
	CustomerID  string
	NIB         *string
	AktaNumber  *string
	NPWPBadan   *string
	LegalStatus *string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type OnboardingCase struct {
	CaseID           string
	CustomerID       string
	CustomerType     CustomerType
	CreatedBySubject string
	ApplicantName    string
	ApplicantPhone   string
	ApplicantEmail   *string
	Channel          Channel
	CurrentStage     CaseStage
	SagaState        SagaState
	RiskLevel        *RiskLevel
	Version          int
	ExpiresAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type IdentityVerification struct {
	VerificationID     string
	CaseID             string
	SubjectType        VerificationSubject
	AttemptNumber      int
	Status             VerificationStatus
	FaceMatchScore     *float64
	LivenessScore      *float64
	DukcapilMatch      *bool
	ExtractedNIK       *string
	ExtractedFullName  *string
	ExtractedBirthDate *time.Time
	ExtractedAddress   *string
	RejectReason       *string
	CreatedAt          time.Time
	CompletedAt        *time.Time
	UpdatedAt          time.Time
}

type BeneficialOwner struct {
	BOID                string
	CompanyID           string
	FullName            string
	VerificationID      *string
	OwnershipPercentage float64
	Status              ReviewStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type AuthorizedSignatory struct {
	SignatoryID    string
	CompanyID      string
	FullName       string
	Role           SignatoryRole
	AuthorityLevel AuthorityLevel
	VerificationID *string
	CreatedAt      time.Time
}

type DocumentRef struct {
	DocumentID         string
	CaseID             string
	VerificationID     *string
	DocType            string
	StorageRef         string
	VerificationStatus ReviewStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ApprovalTaskRef struct {
	TaskID        string
	CaseID        string
	ApprovalLevel int
	ApproverRole  *string
	MakerID       string
	CheckerID     *string
	Decision      ApprovalDecision
	DecidedAt     *time.Time
	CreatedAt     time.Time
}

type OutboxEvent struct {
	OutboxID    int64
	EventID     string
	AggregateID string
	Topic       string
	EventKey    string
	Payload     []byte
	Headers     []byte
	CreatedAt   time.Time
}
