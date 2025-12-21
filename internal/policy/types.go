package policy

type Policy struct {
	PolicyID      string         `yaml:"policy_id"`
	PolicyVersion string         `yaml:"policy_version"`
	Defaults      PolicyDefaults `yaml:"defaults"`
	Rules         []PolicyRule   `yaml:"rules"`
}

type PolicyDefaults struct {
	TTLSeconds      int  `yaml:"ttl_seconds"`
	RequireApproval bool `yaml:"require_approval"`
	Deny            bool `yaml:"deny"`
}

type PolicyRule struct {
	ID     string       `yaml:"id"`
	Match  PolicyMatch  `yaml:"match"`
	Effect PolicyEffect `yaml:"effect"`
}

type PolicyMatch struct {
	Action   string `yaml:"action"`
	Resource string `yaml:"resource"`
	Env      string `yaml:"env"`
}

type PolicyEffect struct {
	RequireApproval *bool  `yaml:"require_approval"`
	Deny            *bool  `yaml:"deny"`
	TTLSeconds      *int   `yaml:"ttl_seconds"`
	AWSRoleARN      string `yaml:"aws_role_arn"`
	Risk            string `yaml:"risk"`
	Reason          string `yaml:"reason"`
}
