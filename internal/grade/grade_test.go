package grade

import (
	"testing"

	"github.com/davidahmann/relia/internal/ledger"
	"github.com/davidahmann/relia/pkg/types"
)

func TestEvaluateInvalidSignatureIsF(t *testing.T) {
	got := Evaluate(Input{Valid: false})
	if got.Grade != "F" {
		t.Fatalf("expected F, got %s", got.Grade)
	}
}

func TestEvaluateHeuristics(t *testing.T) {
	receipt := ledger.StoredReceipt{
		PolicyHash: "sha256:x",
		BodyJSON: []byte(`{
  "policy":{"policy_hash":"sha256:x"},
  "credential_grant":{"role_arn":"arn:aws:iam::123:role/test","ttl_seconds":900},
  "approval":{"required":true,"status":"approved"}
}`),
	}
	ctx := &types.ContextRecord{Evidence: types.ContextEvidence{PlanDigest: "sha256:p", DiffURL: "https://example.com"}}
	dec := &types.DecisionRecord{RequiresApproval: true, Policy: types.DecisionPolicy{PolicyHash: "sha256:x"}}

	got := Evaluate(Input{Valid: true, Receipt: receipt, Context: ctx, Decision: dec})
	if got.Grade != "A" {
		t.Fatalf("expected A, got %s reasons=%v", got.Grade, got.Reasons)
	}

	ctx2 := &types.ContextRecord{Evidence: types.ContextEvidence{}}
	got = Evaluate(Input{Valid: true, Receipt: receipt, Context: ctx2, Decision: dec})
	if got.Grade != "C" {
		t.Fatalf("expected C, got %s reasons=%v", got.Grade, got.Reasons)
	}

	receipt2 := receipt
	receipt2.BodyJSON = []byte(`{"policy":{"policy_hash":"sha256:x"},"credential_grant":{"role_arn":"arn:aws:iam::123:role/test","ttl_seconds":900},"approval":{"required":true,"status":"pending"}}`)
	got = Evaluate(Input{Valid: true, Receipt: receipt2, Context: ctx, Decision: dec})
	if got.Grade != "D" {
		t.Fatalf("expected D, got %s reasons=%v", got.Grade, got.Reasons)
	}
}
