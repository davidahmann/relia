package decision

import (
	"testing"

	"github.com/davidahmann/relia/pkg/types"
)

func TestBuildDecisionDeterministicID(t *testing.T) {
	policy := types.DecisionPolicy{
		PolicyID:      "relia-default",
		PolicyVersion: "2025-12-20",
		PolicyHash:    "sha256:policy",
	}

	recA, err := BuildDecision("sha256:ctx", policy, "allow", []string{"POLICY_MATCH:rule"}, false, "low", "2025-12-20T16:34:13Z")
	if err != nil {
		t.Fatalf("build decision: %v", err)
	}

	recB, err := BuildDecision("sha256:ctx", policy, "allow", []string{"POLICY_MATCH:rule"}, false, "low", "2025-12-20T16:34:13Z")
	if err != nil {
		t.Fatalf("build decision: %v", err)
	}

	if recA.DecisionID == "" {
		t.Fatalf("decision id missing")
	}
	if recA.DecisionID != recB.DecisionID {
		t.Fatalf("decision id not deterministic")
	}

	recC, err := BuildDecision("sha256:ctx", policy, "allow", []string{"POLICY_MATCH:rule"}, false, "high", "2025-12-20T16:34:13Z")
	if err != nil {
		t.Fatalf("build decision: %v", err)
	}
	if recA.DecisionID == recC.DecisionID {
		t.Fatalf("decision id should change when risk changes")
	}
}
