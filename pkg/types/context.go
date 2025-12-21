package types

type ContextRecord struct {
	Schema    string          `json:"schema"`
	ContextID string          `json:"context_id"`
	CreatedAt string          `json:"created_at"`
	Source    ContextSource   `json:"source"`
	Inputs    ContextInputs   `json:"inputs"`
	Evidence  ContextEvidence `json:"evidence,omitempty"`
}

type ContextSource struct {
	Kind     string `json:"kind"`
	Repo     string `json:"repo"`
	Workflow string `json:"workflow"`
	RunID    string `json:"run_id"`
	Actor    string `json:"actor"`
	Ref      string `json:"ref"`
	SHA      string `json:"sha"`
}

type ContextInputs struct {
	Action   string         `json:"action"`
	Resource string         `json:"resource"`
	Env      string         `json:"env"`
	Intent   map[string]any `json:"intent,omitempty"`
}

type ContextEvidence struct {
	PlanDigest string `json:"plan_digest,omitempty"`
	DiffURL    string `json:"diff_url,omitempty"`
}
