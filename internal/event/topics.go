package event

const (
	TopicCaseCreated       = "onboarding.case.created"
	TopicCaseStatusChanged = "onboarding.case.status.changed"
	TopicCIFCreated        = "onboarding.cif.created"
	TopicCaseApproved      = "onboarding.case.approved"
	TopicCaseRejected      = "onboarding.case.rejected"

	// Subscribe
	TopicApprovalDecided = "workflow.approval.decided"
	TopicEKYCCompleted   = "ekyc.verification.completed"
)
