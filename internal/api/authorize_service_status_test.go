package api

import (
	"testing"
	"time"

	"github.com/davidahmann/relia/internal/ledger"
)

func TestAuthorizeExistingIssuingAndErroredStates(t *testing.T) {
	service := newTestService(t, "../../policies/relia.yaml")

	claims := ActorContext{
		Subject:  "repo:org/repo:ref:refs/heads/main",
		Issuer:   "relia-dev",
		Repo:     "org/repo",
		Workflow: "terraform-prod",
		RunID:    "123456",
		SHA:      "abcdef123",
	}
	req := AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "prod"}
	idemKey, err := ComputeIdemKey(claims, req)
	if err != nil {
		t.Fatalf("idem key: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if err := service.Ledger.PutIdempotencyKey(ledger.IdempotencyKey{IdemKey: idemKey, Status: string(IdemIssuing), CreatedAt: now}); err != nil {
		t.Fatalf("put idem: %v", err)
	}
	resp, err := service.Authorize(claims, req, now)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.Error != "issuing in progress" {
		t.Fatalf("expected issuing error, got %q", resp.Error)
	}

	if err := service.Ledger.PutIdempotencyKey(ledger.IdempotencyKey{IdemKey: idemKey, Status: string(IdemErrored), CreatedAt: now}); err != nil {
		t.Fatalf("put idem: %v", err)
	}
	resp, err = service.Authorize(claims, req, now)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.Error != "previous error" {
		t.Fatalf("expected previous error, got %q", resp.Error)
	}

	if err := service.Ledger.PutIdempotencyKey(ledger.IdempotencyKey{IdemKey: idemKey, Status: "weird", CreatedAt: now}); err != nil {
		t.Fatalf("put idem: %v", err)
	}
	resp, err = service.Authorize(claims, req, now)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.Error != "unsupported state" {
		t.Fatalf("expected unsupported state, got %q", resp.Error)
	}
}
