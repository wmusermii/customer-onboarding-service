package domain

type CustomerType string
type CustomerStatus string
type RiskLevel string
type Channel string
type CaseStage string
type SagaState string
type VerificationStatus string
type VerificationSubject string
type ReviewStatus string
type ApprovalDecision string
type SignatoryRole string
type AuthorityLevel string

const (
	CustomerIndividual CustomerType = "INDIVIDUAL"
	CustomerCorporate  CustomerType = "CORPORATE"
)

const (
	CustomerPending   CustomerStatus = "PENDING"
	CustomerActive    CustomerStatus = "ACTIVE"
	CustomerSuspended CustomerStatus = "SUSPENDED"
	CustomerClosed    CustomerStatus = "CLOSED"
)

const (
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)

const (
	ChannelMobile          Channel = "MOBILE"
	ChannelCorporatePortal Channel = "CORPORATE_PORTAL"
	ChannelInternalTeller  Channel = "INTERNAL_TELLER"
)

const (
	StageInitiated           CaseStage = "INITIATED"
	StageDocsSubmitted       CaseStage = "DOCS_SUBMITTED"
	StageIdentityVerified    CaseStage = "IDENTITY_VERIFIED"
	StageRiskAssessed        CaseStage = "RISK_ASSESSED"
	StageVerificationDone    CaseStage = "VERIFICATION_DONE"
	StagePendingReview       CaseStage = "PENDING_REVIEW"
	StagePendingApproval     CaseStage = "PENDING_APPROVAL"
	StageCIFCreation         CaseStage = "CIF_CREATION"
	StageAccountProvisioning CaseStage = "ACCOUNT_PROVISIONING"
	StageCompleted           CaseStage = "COMPLETED"
	StageRejected            CaseStage = "REJECTED"
	StageCancelled           CaseStage = "CANCELLED"
	StageExpired             CaseStage = "EXPIRED"
)

const (
	SagaStarted         SagaState = "STARTED"
	SagaInProgress      SagaState = "IN_PROGRESS"
	SagaWaitingExternal SagaState = "WAITING_EXTERNAL"
	SagaCompensating    SagaState = "COMPENSATING"
	SagaCompensated     SagaState = "COMPENSATED"
	SagaCompleted       SagaState = "COMPLETED"
	SagaFailed          SagaState = "FAILED"
)

const (
	VerProcessing   VerificationStatus = "PROCESSING"
	VerVerified     VerificationStatus = "VERIFIED"
	VerRejected     VerificationStatus = "REJECTED"
	VerManualReview VerificationStatus = "MANUAL_REVIEW_REQUIRED"
)

const (
	SubjectApplicant       VerificationSubject = "APPLICANT"
	SubjectBeneficialOwner VerificationSubject = "BENEFICIAL_OWNER"
	SubjectSignatory       VerificationSubject = "SIGNATORY"
)

const (
	ReviewPending  ReviewStatus = "PENDING"
	ReviewVerified ReviewStatus = "VERIFIED"
	ReviewRejected ReviewStatus = "REJECTED"
)

const (
	DecisionPending  ApprovalDecision = "PENDING"
	DecisionApproved ApprovalDecision = "APPROVED"
	DecisionRejected ApprovalDecision = "REJECTED"
)

const (
	RoleDirector        SignatoryRole = "DIRECTOR"
	RoleCommissioner    SignatoryRole = "COMMISSIONER"
	RoleAuthorizedProxy SignatoryRole = "AUTHORIZED_PROXY"
)

const (
	AuthoritySingle AuthorityLevel = "SINGLE_SIGNATURE"
	AuthorityJoint  AuthorityLevel = "JOINT_SIGNATURE"
)

// IsTerminal: stage yang sudah tidak bisa berubah lagi.
func (s CaseStage) IsTerminal() bool {
	switch s {
	case StageCompleted, StageRejected, StageCancelled, StageExpired:
		return true
	}
	return false
}

// APIStatus: pemetaan stage (DB) -> status ringkas untuk response API.
func (s CaseStage) APIStatus() string {
	switch s {
	case StageInitiated:
		return "INITIATED"
	case StagePendingReview, StagePendingApproval:
		return "PENDING_APPROVAL"
	case StageCompleted, StageRejected, StageCancelled, StageExpired:
		return string(s)
	default:
		return "PROCESSING"
	}
}
