package pack

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/davidahmann/relia/internal/context"
	"github.com/davidahmann/relia/internal/decision"
	"github.com/davidahmann/relia/internal/ledger"
	"github.com/davidahmann/relia/pkg/types"
)

func TestBuildSummaryIncludesLinksAndHTML(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)

	createdAt := time.Now().UTC().Format(time.RFC3339)
	ctx, err := context.BuildContext(
		types.ContextSource{Kind: "github_actions", Repo: "org/repo", Workflow: "wf", RunID: "1", Actor: "dev", SHA: "abc"},
		types.ContextInputs{Action: "terraform.apply", Resource: "stack/prod", Env: "prod"},
		types.ContextEvidence{PlanDigest: "sha256:plan", DiffURL: "https://example.com/diff"},
		createdAt,
	)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	policyMeta := types.DecisionPolicy{PolicyID: "p", PolicyVersion: "v", PolicyHash: "sha256:ph"}
	dec, err := decision.BuildDecision(ctx.ContextID, policyMeta, "allow", nil, false, "low", createdAt)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	receipt, err := ledger.MakeReceipt(ledger.MakeReceiptInput{
		CreatedAt:  createdAt,
		IdemKey:    "idem",
		ContextID:  ctx.ContextID,
		DecisionID: dec.DecisionID,
		Actor:      types.ReceiptActor{Kind: "workload", Subject: "dev"},
		Request:    types.ReceiptRequest{Action: "terraform.apply", Resource: "stack/prod", Env: "prod"},
		Policy:     types.ReceiptPolicy{PolicyID: "p", PolicyVersion: "v", PolicyHash: "sha256:ph"},
		CredentialGrant: &types.ReceiptCredentialGrant{
			Provider:   "aws_sts",
			Method:     "AssumeRoleWithWebIdentity",
			RoleARN:    "arn:aws:iam::123:role/test",
			TTLSeconds: 900,
		},
		Outcome: types.ReceiptOutcome{Status: types.OutcomeIssuedCredentials, ExpiresAt: createdAt},
	}, testSigner{keyID: "k1", priv: priv})
	if err != nil {
		t.Fatalf("receipt: %v", err)
	}

	summary, htmlBytes, err := BuildSummary(Input{
		Receipt:   receipt,
		Context:   ctx,
		Decision:  dec,
		Policy:    []byte("policy_id: p\n"),
		CreatedAt: createdAt,
	}, "https://relia.example.com")
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.VerifyURL == "" || summary.PackURL == "" {
		t.Fatalf("expected links to be set")
	}
	if summary.Grade == "" {
		t.Fatalf("expected grade")
	}
	if len(htmlBytes) == 0 {
		t.Fatalf("expected html bytes")
	}
}

func TestBuildSummaryNoLinksWhenNoBaseURL(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)

	createdAt := time.Now().UTC().Format(time.RFC3339)
	ctx, err := context.BuildContext(
		types.ContextSource{Kind: "github_actions", Repo: "org/repo", Workflow: "wf", RunID: "1", Actor: "dev", SHA: "abc"},
		types.ContextInputs{Action: "deploy.prod", Resource: "refs/heads/main", Env: "prod"},
		types.ContextEvidence{},
		createdAt,
	)
	if err != nil {
		t.Fatalf("context: %v", err)
	}
	policyMeta := types.DecisionPolicy{PolicyID: "p", PolicyVersion: "v", PolicyHash: "sha256:ph"}
	dec, err := decision.BuildDecision(ctx.ContextID, policyMeta, "deny", nil, true, "high", createdAt)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	receipt, err := ledger.MakeReceipt(ledger.MakeReceiptInput{
		CreatedAt:  createdAt,
		IdemKey:    "idem",
		ContextID:  ctx.ContextID,
		DecisionID: dec.DecisionID,
		Actor:      types.ReceiptActor{Kind: "workload", Subject: "dev"},
		Request:    types.ReceiptRequest{Action: "deploy.prod", Resource: "refs/heads/main", Env: "prod"},
		Policy:     types.ReceiptPolicy{PolicyID: "p", PolicyVersion: "v", PolicyHash: "sha256:ph"},
		Approval:   &types.ReceiptApproval{Required: true, Status: "pending"},
		Outcome:    types.ReceiptOutcome{Status: types.OutcomeApprovalPending},
	}, testSigner{keyID: "k1", priv: priv})
	if err != nil {
		t.Fatalf("receipt: %v", err)
	}

	summary, htmlBytes, err := BuildSummary(Input{
		Receipt:   receipt,
		Context:   ctx,
		Decision:  dec,
		Policy:    []byte("policy_id: p\n"),
		CreatedAt: createdAt,
	}, "")
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.VerifyURL != "" || summary.PackURL != "" {
		t.Fatalf("expected no links")
	}
	if len(htmlBytes) == 0 {
		t.Fatalf("expected html bytes")
	}
}
