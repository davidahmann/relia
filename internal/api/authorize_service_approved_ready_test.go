package api

import (
	"testing"

	"github.com/davidahmann/relia/internal/ledger"
)

func TestAuthorizeApprovedReadyMissingLatestReceipt(t *testing.T) {
	svc := newTestService(t, "../../policies/relia.yaml")

	claims := ActorContext{Subject: "sub", Issuer: "iss", Repo: "repo", Workflow: "wf", RunID: "run", SHA: "sha"}
	req := AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "dev"}
	idemKey, err := ComputeIdemKey(claims, req)
	if err != nil {
		t.Fatalf("idem: %v", err)
	}

	if err := svc.Ledger.PutIdempotencyKey(ledger.IdempotencyKey{IdemKey: idemKey, Status: string(IdemApprovedReady), CreatedAt: "now", UpdatedAt: "now"}); err != nil {
		t.Fatalf("put idem: %v", err)
	}

	if _, err := svc.Authorize(claims, req, "2025-12-20T16:34:14Z"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAuthorizeApprovedReadyMissingPolicyVersion(t *testing.T) {
	svc := newTestService(t, "../../policies/relia.yaml")

	claims := ActorContext{Subject: "sub", Issuer: "iss", Repo: "repo", Workflow: "wf", RunID: "run", SHA: "sha"}
	req := AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "dev"}
	idemKey, err := ComputeIdemKey(claims, req)
	if err != nil {
		t.Fatalf("idem: %v", err)
	}

	latest := "receipt-1"
	if err := svc.Ledger.PutReceipt(ledger.ReceiptRecord{ReceiptID: latest, PolicyHash: "missing", ContextID: "c", DecisionID: "d"}); err != nil {
		t.Fatalf("put receipt: %v", err)
	}
	if err := svc.Ledger.PutIdempotencyKey(ledger.IdempotencyKey{
		IdemKey:         idemKey,
		Status:          string(IdemApprovedReady),
		LatestReceiptID: &latest,
		CreatedAt:       "now",
		UpdatedAt:       "now",
	}); err != nil {
		t.Fatalf("put idem: %v", err)
	}

	if _, err := svc.Authorize(claims, req, "2025-12-20T16:34:14Z"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAuthorizeApprovedReadyMissingRoleArn(t *testing.T) {
	svc := newTestService(t, "../../policies/relia.yaml")

	claims := ActorContext{Subject: "sub", Issuer: "iss", Repo: "repo", Workflow: "wf", RunID: "run", SHA: "sha", Token: "jwt"}
	req := AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "dev"}
	idemKey, err := ComputeIdemKey(claims, req)
	if err != nil {
		t.Fatalf("idem: %v", err)
	}

	policyHash := "ph1"
	policy := `
policy_id: test
policy_version: "1"
defaults:
  ttl_seconds: 900
  require_approval: false
  deny: false
rules:
  - id: allow-without-role
    match:
      action: "terraform.apply"
      env: "dev"
    effect:
      ttl_seconds: 900
`
	if err := svc.Ledger.PutPolicyVersion(ledger.PolicyVersionRecord{PolicyHash: policyHash, PolicyID: "test", PolicyVersion: "1", PolicyYAML: policy, CreatedAt: "now"}); err != nil {
		t.Fatalf("put policy: %v", err)
	}

	latest := "receipt-2"
	if err := svc.Ledger.PutReceipt(ledger.ReceiptRecord{ReceiptID: latest, PolicyHash: policyHash, ContextID: "c", DecisionID: "d"}); err != nil {
		t.Fatalf("put receipt: %v", err)
	}
	if err := svc.Ledger.PutIdempotencyKey(ledger.IdempotencyKey{
		IdemKey:         idemKey,
		Status:          string(IdemApprovedReady),
		LatestReceiptID: &latest,
		CreatedAt:       "now",
		UpdatedAt:       "now",
	}); err != nil {
		t.Fatalf("put idem: %v", err)
	}

	if _, err := svc.Authorize(claims, req, "2025-12-20T16:34:14Z"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestAuthorizeApprovedReadyUnexpectedVerdict(t *testing.T) {
	svc := newTestService(t, "../../policies/relia.yaml")

	claims := ActorContext{Subject: "sub", Issuer: "iss", Repo: "repo", Workflow: "wf", RunID: "run", SHA: "sha"}
	req := AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "dev"}
	idemKey, err := ComputeIdemKey(claims, req)
	if err != nil {
		t.Fatalf("idem: %v", err)
	}

	policyHash := "ph2"
	policy := `
policy_id: test
policy_version: "1"
defaults:
  ttl_seconds: 900
  require_approval: false
  deny: true
rules: []
`
	if err := svc.Ledger.PutPolicyVersion(ledger.PolicyVersionRecord{PolicyHash: policyHash, PolicyID: "test", PolicyVersion: "1", PolicyYAML: policy, CreatedAt: "now"}); err != nil {
		t.Fatalf("put policy: %v", err)
	}

	latest := "receipt-3"
	if err := svc.Ledger.PutReceipt(ledger.ReceiptRecord{ReceiptID: latest, PolicyHash: policyHash, ContextID: "c", DecisionID: "d"}); err != nil {
		t.Fatalf("put receipt: %v", err)
	}
	if err := svc.Ledger.PutIdempotencyKey(ledger.IdempotencyKey{
		IdemKey:         idemKey,
		Status:          string(IdemApprovedReady),
		LatestReceiptID: &latest,
		CreatedAt:       "now",
		UpdatedAt:       "now",
	}); err != nil {
		t.Fatalf("put idem: %v", err)
	}

	if _, err := svc.Authorize(claims, req, "2025-12-20T16:34:14Z"); err == nil {
		t.Fatalf("expected error")
	}
}
