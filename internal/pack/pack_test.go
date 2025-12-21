package pack

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/davidahmann/relia/internal/context"
	"github.com/davidahmann/relia/internal/decision"
	"github.com/davidahmann/relia/internal/ledger"
	"github.com/davidahmann/relia/pkg/types"
)

type testSigner struct {
	keyID string
	priv  ed25519.PrivateKey
}

func (s testSigner) KeyID() string {
	return s.keyID
}

func (s testSigner) SignEd25519(message []byte) ([]byte, error) {
	return ed25519.Sign(s.priv, message), nil
}

func TestBuildZipIncludesArtifacts(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)

	createdAt := time.Now().UTC().Format(time.RFC3339)
	source := types.ContextSource{Kind: "github_actions", Repo: "org/repo", Workflow: "wf", RunID: "1", Actor: "dev", Ref: "refs/heads/main", SHA: "abc"}
	inputs := types.ContextInputs{Action: "terraform.apply", Resource: "res", Env: "prod"}
	evidence := types.ContextEvidence{PlanDigest: "sha256:abc"}
	ctx, err := context.BuildContext(source, inputs, evidence, createdAt)
	if err != nil {
		t.Fatalf("context: %v", err)
	}

	policy := types.DecisionPolicy{PolicyID: "relia-default", PolicyVersion: "2025-12-20", PolicyHash: "sha256:policy"}
	dec, err := decision.BuildDecision(ctx.ContextID, policy, "allow", nil, false, "high", createdAt)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}

	receipt, err := ledger.MakeReceipt(ledger.MakeReceiptInput{
		CreatedAt:  createdAt,
		IdemKey:    "idem",
		ContextID:  ctx.ContextID,
		DecisionID: dec.DecisionID,
		Actor:      types.ReceiptActor{Kind: "workload", Subject: "dev"},
		Request:    types.ReceiptRequest{RequestID: "req", Action: "terraform.apply", Resource: "res", Env: "prod"},
		Policy:     types.ReceiptPolicy{PolicyHash: "sha256:policy"},
		Outcome:    types.ReceiptOutcome{Status: types.OutcomeDenied},
	}, testSigner{keyID: "test", priv: priv})
	if err != nil {
		t.Fatalf("receipt: %v", err)
	}

	zipBytes, err := BuildZip(Input{
		Receipt:   receipt,
		Context:   ctx,
		Decision:  dec,
		Policy:    []byte("policy_id: relia-default\n"),
		Approvals: []ApprovalRecord{{ApprovalID: "approval-1", Status: "approved", ReceiptID: receipt.ReceiptID}},
	}, "http://localhost:8080")
	if err != nil {
		t.Fatalf("build zip: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("zip reader: %v", err)
	}

	expected := map[string]bool{
		"receipt.json":   false,
		"context.json":   false,
		"decision.json":  false,
		"policy.yaml":    false,
		"approvals.json": false,
		"manifest.json":  false,
		"sha256sums.txt": false,
	}

	for _, file := range reader.File {
		if _, ok := expected[file.Name]; ok {
			expected[file.Name] = true
		}
	}

	for name, seen := range expected {
		if !seen {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestBuildFilesRequiresPolicy(t *testing.T) {
	_, err := BuildFiles(Input{}, "")
	if err == nil {
		t.Fatalf("expected error for missing policy")
	}
}

func TestWriteZip(t *testing.T) {
	files := map[string][]byte{
		"a.txt": []byte("alpha"),
		"b.txt": []byte("bravo"),
	}
	buf := bytes.NewBuffer(nil)
	if err := WriteZip(buf, files); err != nil {
		t.Fatalf("write zip: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("zip reader: %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("expected 2 files, got %d", len(reader.File))
	}
}
