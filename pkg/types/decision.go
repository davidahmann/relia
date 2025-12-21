package types

type DecisionRecord struct {
	Schema           string         `json:"schema"`
	DecisionID       string         `json:"decision_id"`
	CreatedAt        string         `json:"created_at"`
	ContextID        string         `json:"context_id"`
	Policy           DecisionPolicy `json:"policy"`
	Verdict          string         `json:"verdict"`
	ReasonCodes      []string       `json:"reason_codes,omitempty"`
	RequiresApproval bool           `json:"requires_approval"`
	Risk             string         `json:"risk,omitempty"`
}

type DecisionPolicy struct {
	PolicyID      string `json:"policy_id"`
	PolicyVersion string `json:"policy_version"`
	PolicyHash    string `json:"policy_hash"`
}
