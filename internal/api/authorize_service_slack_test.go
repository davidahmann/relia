package api

import (
	"crypto/ed25519"
	"testing"

	"github.com/davidahmann/relia/internal/aws"
	"github.com/davidahmann/relia/internal/ledger"
	"github.com/davidahmann/relia/internal/slack"
)

type fakeSlackNotifier struct {
	called  int
	channel string
	input   slack.ApprovalMessageInput
}

func (f *fakeSlackNotifier) PostApproval(channel string, message slack.ApprovalMessageInput) (string, error) {
	f.called++
	f.channel = channel
	f.input = message
	return "1700000000.1234", nil
}

func TestAuthorizePostsSlackForPendingApproval(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	notifier := &fakeSlackNotifier{}
	service, err := NewAuthorizeService(NewAuthorizeServiceInput{
		PolicyPath: "../../policies/relia.yaml",
		Ledger:     ledger.NewInMemoryStore(),
		Signer:     devSigner{keyID: "test", priv: priv},
		PublicKey:  pub,
		Broker:     aws.DevBroker{},
		Slack:      notifier,
		SlackChan:  "C123",
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}

	claims := ActorContext{
		Subject:  "repo:org/repo:ref:refs/heads/main",
		Issuer:   "relia-dev",
		Repo:     "org/repo",
		Workflow: "terraform-prod",
		RunID:    "123456",
		SHA:      "abcdef123",
		Token:    "jwt",
	}
	req := AuthorizeRequest{
		Action:   "terraform.apply",
		Resource: "res",
		Env:      "prod",
		Evidence: AuthorizeEvidence{DiffURL: "https://example.test/diff"},
	}

	resp, err := service.Authorize(claims, req, "2025-12-20T16:34:14Z")
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	if resp.Verdict != string(VerdictRequireApproval) || resp.Approval == nil {
		t.Fatalf("expected require_approval, got %+v", resp)
	}

	if notifier.called != 1 {
		t.Fatalf("expected slack notifier to be called once, got %d", notifier.called)
	}
	if notifier.channel != "C123" || notifier.input.ApprovalID != resp.Approval.ApprovalID || notifier.input.ReceiptID != resp.ReceiptID {
		t.Fatalf("unexpected notifier input: channel=%s input=%+v", notifier.channel, notifier.input)
	}

	approval, ok := service.Ledger.GetApproval(resp.Approval.ApprovalID)
	if !ok {
		t.Fatalf("expected approval record")
	}
	if approval.SlackChannel == nil || *approval.SlackChannel != "C123" {
		t.Fatalf("expected slack channel to be stored")
	}
	if approval.SlackMsgTS == nil || *approval.SlackMsgTS != "1700000000.1234" {
		t.Fatalf("expected slack msg ts to be stored")
	}
}
