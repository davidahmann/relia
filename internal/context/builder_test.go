package context

import (
	"testing"

	"github.com/davidahmann/relia_oss/pkg/types"
)

func TestBuildContextDeterministicID(t *testing.T) {
	source := types.ContextSource{
		Kind:     "github_actions",
		Repo:     "org/repo",
		Workflow: "wf",
		RunID:    "run",
		Actor:    "actor",
		Ref:      "refs/heads/main",
		SHA:      "sha",
	}
	inputs := types.ContextInputs{
		Action:   "terraform.apply",
		Resource: "resource",
		Env:      "prod",
		Intent: map[string]any{
			"change_id": "CHG-1",
		},
	}
	evidence := types.ContextEvidence{
		PlanDigest: "sha256:plan",
		DiffURL:    "https://example.com",
	}

	ctxA, err := BuildContext(source, inputs, evidence, "2025-12-20T16:34:12Z")
	if err != nil {
		t.Fatalf("build context: %v", err)
	}
	ctxB, err := BuildContext(source, inputs, evidence, "2025-12-20T16:34:12Z")
	if err != nil {
		t.Fatalf("build context: %v", err)
	}

	if ctxA.ContextID == "" {
		t.Fatalf("context id missing")
	}
	if ctxA.ContextID != ctxB.ContextID {
		t.Fatalf("context id not deterministic")
	}

	inputs.Intent["change_id"] = "CHG-2"
	ctxC, err := BuildContext(source, inputs, evidence, "2025-12-20T16:34:12Z")
	if err != nil {
		t.Fatalf("build context: %v", err)
	}
	if ctxA.ContextID == ctxC.ContextID {
		t.Fatalf("context id should change when intent changes")
	}
}
