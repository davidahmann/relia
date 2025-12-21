package api

import (
	"testing"
	"time"
)

func TestApproveIdempotentWhenAlreadyFinalized(t *testing.T) {
	service := newTestService(t, "../../policies/relia.yaml")

	claims := ActorContext{
		Subject:  "repo:org/repo:ref:refs/heads/main",
		Issuer:   "relia-dev",
		Repo:     "org/repo",
		Workflow: "terraform-prod",
		RunID:    "123456",
		SHA:      "abcdef123",
	}

	now := time.Now().UTC().Format(time.RFC3339)
	resp, err := service.Authorize(claims, AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "prod"}, now)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.Approval == nil || resp.Approval.ApprovalID == "" {
		t.Fatalf("expected approval id")
	}

	receipt1, err := service.Approve(resp.Approval.ApprovalID, "approved", now)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if receipt1 == "" {
		t.Fatalf("expected receipt id")
	}

	receipt2, err := service.Approve(resp.Approval.ApprovalID, "approved", now)
	if err != nil {
		t.Fatalf("approve (idempotent): %v", err)
	}
	if receipt2 != receipt1 {
		t.Fatalf("expected same receipt id, got %s vs %s", receipt2, receipt1)
	}
}
