package types

type OutcomeStatus string

// #nosec G101 -- outcome status labels are not credentials.
const (
	OutcomeApprovalPending    OutcomeStatus = "approval_pending"
	OutcomeApprovalApproved   OutcomeStatus = "approval_approved"
	OutcomeApprovalDenied     OutcomeStatus = "approval_denied"
	OutcomeIssuingCredentials OutcomeStatus = "issuing_credentials"
	OutcomeIssuedCredentials  OutcomeStatus = "issued_credentials"
	OutcomeDenied             OutcomeStatus = "denied"
	OutcomeIssueFailed        OutcomeStatus = "issue_failed"
)

type ReceiptActor struct {
	Kind     string `json:"kind"`
	Subject  string `json:"subject"`
	Issuer   string `json:"issuer"`
	Repo     string `json:"repo"`
	Workflow string `json:"workflow"`
	RunID    string `json:"run_id"`
	SHA      string `json:"sha"`
}

type ReceiptRequest struct {
	RequestID string         `json:"request_id"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Env       string         `json:"env"`
	Intent    map[string]any `json:"intent,omitempty"`
}

type ReceiptPolicy struct {
	PolicyID      string `json:"policy_id"`
	PolicyVersion string `json:"policy_version"`
	PolicyHash    string `json:"policy_hash"`
}

type ReceiptApproval struct {
	Required   bool   `json:"required"`
	ApprovalID string `json:"approval_id,omitempty"`
	Status     string `json:"status,omitempty"`
	ApprovedAt string `json:"approved_at,omitempty"`
	Approver   *struct {
		Kind    string `json:"kind"`
		ID      string `json:"id"`
		Display string `json:"display"`
	} `json:"approver,omitempty"`
}

type ReceiptCredentialGrant struct {
	Provider    string `json:"provider"`
	Method      string `json:"method"`
	RoleARN     string `json:"role_arn"`
	Region      string `json:"region"`
	TTLSeconds  int64  `json:"ttl_seconds"`
	ScopeDigest string `json:"scope_digest"`
}

type ReceiptOutcome struct {
	Status    OutcomeStatus `json:"status"`
	IssuedAt  string        `json:"issued_at,omitempty"`
	ExpiresAt string        `json:"expires_at,omitempty"`
	Error     *struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
	} `json:"error,omitempty"`
}
