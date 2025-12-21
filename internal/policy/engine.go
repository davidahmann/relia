package policy

type Input struct {
	Action   string
	Resource string
	Env      string
}

type Decision struct {
	Verdict         string
	RequireApproval bool
	TTLSeconds      int
	AWSRoleARN      string
	Risk            string
	Reason          string
	MatchedRuleID   string
	ReasonCodes     []string
	PolicyID        string
	PolicyVersion   string
	PolicyHash      string
}

// Evaluate applies the first matching rule to input, otherwise defaults.
func Evaluate(p Policy, policyHash string, input Input) Decision {
	decision := Decision{
		Verdict:         "allow",
		RequireApproval: p.Defaults.RequireApproval,
		TTLSeconds:      p.Defaults.TTLSeconds,
		PolicyID:        p.PolicyID,
		PolicyVersion:   p.PolicyVersion,
		PolicyHash:      policyHash,
	}

	if p.Defaults.Deny {
		decision.Verdict = "deny"
	}

	for _, rule := range p.Rules {
		if !matchRule(rule.Match, input) {
			continue
		}

		decision.MatchedRuleID = rule.ID
		decision.ReasonCodes = append(decision.ReasonCodes, "POLICY_MATCH:"+rule.ID)

		if rule.Effect.RequireApproval != nil {
			decision.RequireApproval = *rule.Effect.RequireApproval
		}
		if rule.Effect.Deny != nil {
			if *rule.Effect.Deny {
				decision.Verdict = "deny"
			} else if decision.Verdict == "deny" {
				decision.Verdict = "allow"
			}
		}
		if rule.Effect.TTLSeconds != nil {
			decision.TTLSeconds = *rule.Effect.TTLSeconds
		}
		if rule.Effect.AWSRoleARN != "" {
			decision.AWSRoleARN = rule.Effect.AWSRoleARN
		}
		if rule.Effect.Risk != "" {
			decision.Risk = rule.Effect.Risk
		}
		if rule.Effect.Reason != "" {
			decision.Reason = rule.Effect.Reason
		}

		if decision.Verdict != "deny" {
			if decision.RequireApproval {
				decision.Verdict = "require_approval"
			} else {
				decision.Verdict = "allow"
			}
		}
		return decision
	}

	if decision.Verdict != "deny" {
		if decision.RequireApproval {
			decision.Verdict = "require_approval"
		} else {
			decision.Verdict = "allow"
		}
	}

	return decision
}

func matchRule(match PolicyMatch, input Input) bool {
	if match.Action != "" && match.Action != input.Action {
		return false
	}
	if match.Resource != "" && match.Resource != input.Resource {
		return false
	}
	if match.Env != "" && match.Env != input.Env {
		return false
	}
	return true
}
