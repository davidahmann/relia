package policy

import "testing"

func TestEvaluatePolicyDefaults(t *testing.T) {
	p := Policy{
		PolicyID:      "relia-default",
		PolicyVersion: "2025-12-20",
		Defaults: PolicyDefaults{
			TTLSeconds:      900,
			RequireApproval: false,
			Deny:            false,
		},
	}

	decision := Evaluate(p, "sha256:policy", Input{Action: "noop"})
	if decision.Verdict != "allow" {
		t.Fatalf("expected allow, got %s", decision.Verdict)
	}
	if decision.TTLSeconds != 900 {
		t.Fatalf("expected ttl 900, got %d", decision.TTLSeconds)
	}
}

func TestEvaluatePolicyRuleMatch(t *testing.T) {
	requireApproval := true
	deny := true
	ttl := 600

	p := Policy{
		PolicyID:      "relia-default",
		PolicyVersion: "2025-12-20",
		Defaults: PolicyDefaults{
			TTLSeconds:      900,
			RequireApproval: false,
			Deny:            false,
		},
		Rules: []PolicyRule{
			{
				ID: "rule-1",
				Match: PolicyMatch{
					Action: "terraform.apply",
					Env:    "prod",
				},
				Effect: PolicyEffect{
					RequireApproval: &requireApproval,
					TTLSeconds:      &ttl,
					Risk:            "high",
				},
			},
			{
				ID: "rule-2",
				Match: PolicyMatch{
					Env: "prod",
				},
				Effect: PolicyEffect{
					Deny: &deny,
				},
			},
		},
	}

	decision := Evaluate(p, "sha256:policy", Input{Action: "terraform.apply", Env: "prod"})
	if decision.Verdict != "require_approval" {
		t.Fatalf("expected require_approval, got %s", decision.Verdict)
	}
	if decision.TTLSeconds != 600 {
		t.Fatalf("expected ttl 600, got %d", decision.TTLSeconds)
	}
	if decision.Risk != "high" {
		t.Fatalf("expected risk high, got %s", decision.Risk)
	}
	if decision.MatchedRuleID != "rule-1" {
		t.Fatalf("expected matched rule-1, got %s", decision.MatchedRuleID)
	}
}

func TestEvaluatePolicyDenyDefault(t *testing.T) {
	p := Policy{
		PolicyID:      "relia-default",
		PolicyVersion: "2025-12-20",
		Defaults: PolicyDefaults{
			TTLSeconds:      900,
			RequireApproval: false,
			Deny:            true,
		},
	}

	decision := Evaluate(p, "sha256:policy", Input{Action: "terraform.apply", Env: "prod"})
	if decision.Verdict != "deny" {
		t.Fatalf("expected deny, got %s", decision.Verdict)
	}
}

func TestEvaluatePolicyNoMatchUsesDefaults(t *testing.T) {
	p := Policy{
		PolicyID:      "relia-default",
		PolicyVersion: "2025-12-20",
		Defaults: PolicyDefaults{
			TTLSeconds:      120,
			RequireApproval: true,
			Deny:            false,
		},
		Rules: []PolicyRule{
			{
				ID: "resource-only",
				Match: PolicyMatch{
					Resource: "db",
				},
				Effect: PolicyEffect{
					RequireApproval: boolPtr(false),
				},
			},
		},
	}

	decision := Evaluate(p, "sha256:policy", Input{Action: "deploy", Env: "prod", Resource: "queue"})
	if decision.Verdict != "require_approval" {
		t.Fatalf("expected require_approval, got %s", decision.Verdict)
	}
	if decision.TTLSeconds != 120 {
		t.Fatalf("expected ttl 120, got %d", decision.TTLSeconds)
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func TestEvaluatePolicyRuleDeny(t *testing.T) {
	deny := true

	p := Policy{
		PolicyID:      "relia-default",
		PolicyVersion: "2025-12-20",
		Defaults: PolicyDefaults{
			TTLSeconds:      900,
			RequireApproval: false,
			Deny:            false,
		},
		Rules: []PolicyRule{
			{
				ID: "deny-prod",
				Match: PolicyMatch{
					Env: "prod",
				},
				Effect: PolicyEffect{
					Deny:   &deny,
					Reason: "blocked",
				},
			},
		},
	}

	decision := Evaluate(p, "sha256:policy", Input{Action: "deploy", Env: "prod"})
	if decision.Verdict != "deny" {
		t.Fatalf("expected deny, got %s", decision.Verdict)
	}
	if decision.Reason != "blocked" {
		t.Fatalf("expected reason blocked, got %s", decision.Reason)
	}
}

func TestEvaluatePolicyRuleClearsDefaultDeny(t *testing.T) {
	deny := false
	requireApproval := true
	ttl := 300

	p := Policy{
		PolicyID:      "relia-default",
		PolicyVersion: "2025-12-20",
		Defaults: PolicyDefaults{
			TTLSeconds:      900,
			RequireApproval: false,
			Deny:            true,
		},
		Rules: []PolicyRule{
			{
				ID: "override-deny",
				Match: PolicyMatch{
					Action: "deploy",
					Env:    "prod",
				},
				Effect: PolicyEffect{
					Deny:            &deny,
					RequireApproval: &requireApproval,
					TTLSeconds:      &ttl,
					AWSRoleARN:      "arn:aws:iam::123456789012:role/relia-prod",
					Risk:            "high",
				},
			},
		},
	}

	decision := Evaluate(p, "sha256:policy", Input{Action: "deploy", Env: "prod"})
	if decision.Verdict != "require_approval" {
		t.Fatalf("expected require_approval, got %s", decision.Verdict)
	}
	if decision.TTLSeconds != 300 {
		t.Fatalf("expected ttl 300, got %d", decision.TTLSeconds)
	}
	if decision.AWSRoleARN == "" {
		t.Fatalf("expected aws role arn to be set")
	}
	if decision.Risk != "high" {
		t.Fatalf("expected risk high, got %s", decision.Risk)
	}
	if len(decision.ReasonCodes) != 1 || decision.ReasonCodes[0] != "POLICY_MATCH:override-deny" {
		t.Fatalf("unexpected reason codes: %v", decision.ReasonCodes)
	}
}
