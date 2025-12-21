package ledger

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"github.com/davidahmann/relia_oss/internal/crypto"
	"github.com/davidahmann/relia_oss/pkg/types"
)

type testSigner struct {
	keyID string
	priv  ed25519.PrivateKey
}

func (s testSigner) KeyID() string {
	return s.keyID
}

func (s testSigner) SignEd25519(message []byte) ([]byte, error) {
	return crypto.SignEd25519(s.priv, message)
}

func TestMakeReceiptAndVerify(t *testing.T) {
	seed := bytes.Repeat([]byte{0x01}, 32)
	priv, pub, err := crypto.KeyPairFromSeed(seed)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}

	signer := testSigner{keyID: "test-key", priv: priv}

	input := MakeReceiptInput{
		Schema:     ReceiptSchema,
		CreatedAt:  "2025-12-20T16:34:14Z",
		IdemKey:    "idem:v1:sha256:abc",
		ContextID:  "sha256:ctx",
		DecisionID: "sha256:dec",
		Actor: types.ReceiptActor{
			Kind:     "workload",
			Subject:  "repo:org/repo:ref:refs/heads/main",
			Issuer:   "https://token.actions.githubusercontent.com",
			Repo:     "org/repo",
			Workflow: "wf",
			RunID:    "123",
			SHA:      "abc",
		},
		Request: types.ReceiptRequest{
			RequestID: "01JTEST",
			Action:    "deploy",
			Resource:  "resource",
			Env:       "prod",
			Intent: map[string]any{
				"change_id": "CHG-1",
			},
		},
		Policy: types.ReceiptPolicy{
			PolicyID:      "relia-default",
			PolicyVersion: "2025-12-20",
			PolicyHash:    "sha256:policy",
		},
		Outcome: types.ReceiptOutcome{
			Status: types.OutcomeDenied,
		},
	}

	receipt, err := MakeReceipt(input, signer)
	if err != nil {
		t.Fatalf("make receipt: %v", err)
	}

	if receipt.ReceiptID == "" || receipt.BodyDigest == "" {
		t.Fatalf("missing digest")
	}
	if receipt.ReceiptID != receipt.BodyDigest {
		t.Fatalf("receipt id should equal body digest")
	}

	if err := VerifyReceipt(receipt, pub); err != nil {
		t.Fatalf("verify receipt: %v", err)
	}
}

func TestMakeReceiptRejectsOutcome(t *testing.T) {
	seed := bytes.Repeat([]byte{0x01}, 32)
	priv, _, err := crypto.KeyPairFromSeed(seed)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}

	signer := testSigner{keyID: "test-key", priv: priv}

	input := MakeReceiptInput{
		Schema:     ReceiptSchema,
		CreatedAt:  "2025-12-20T16:34:14Z",
		IdemKey:    "idem:v1:sha256:abc",
		ContextID:  "sha256:ctx",
		DecisionID: "sha256:dec",
		Actor:      types.ReceiptActor{Kind: "workload"},
		Request:    types.ReceiptRequest{Action: "deploy", Resource: "res", Env: "prod"},
		Policy:     types.ReceiptPolicy{PolicyHash: "sha256:policy"},
		Outcome:    types.ReceiptOutcome{Status: types.OutcomeStatus("invalid")},
	}

	_, err = MakeReceipt(input, signer)
	if err == nil {
		t.Fatalf("expected error for invalid outcome")
	}
}
