package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/davidahmann/relia/internal/auth"
	"github.com/davidahmann/relia/internal/ledger"
)

func TestNewAuthorizeServiceDefaults(t *testing.T) {
	svc, err := NewAuthorizeService(NewAuthorizeServiceInput{PolicyPath: "../../policies/relia.yaml"})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	if svc.Ledger == nil || svc.Signer == nil || svc.PublicKey == nil || svc.Broker == nil {
		t.Fatalf("expected defaults to be set")
	}
}

func TestNewAuthorizeServiceMissingPolicyPath(t *testing.T) {
	if _, err := NewAuthorizeService(NewAuthorizeServiceInput{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestPtrOrNil(t *testing.T) {
	if ptrOrNil("") != nil {
		t.Fatalf("expected nil")
	}
	if v := ptrOrNil("x"); v == nil || *v != "x" {
		t.Fatalf("unexpected value")
	}
}

func TestPackInvalidStoredArtifacts(t *testing.T) {
	os.Setenv("RELIA_DEV_TOKEN", "test-token")
	defer os.Unsetenv("RELIA_DEV_TOKEN")

	svc := newTestService(t, "../../policies/relia.yaml")

	claims := ActorContext{
		Subject:  "repo:org/repo:ref:refs/heads/main",
		Issuer:   "relia-dev",
		Repo:     "org/repo",
		Workflow: "terraform-dev",
		RunID:    "123456",
		SHA:      "abcdef123",
		Token:    "jwt",
	}

	resp, err := svc.Authorize(claims, AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "dev"}, "2025-12-20T16:34:14Z")
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}

	receipt, ok := svc.Ledger.GetReceipt(resp.ReceiptID)
	if !ok {
		t.Fatalf("missing receipt record")
	}

	router := NewRouter(&Handler{Auth: auth.NewAuthenticatorFromEnv(), AuthorizeService: svc})

	// Invalid context JSON => 500
	if err := svc.Ledger.PutContext(ledger.ContextRecord{ContextID: receipt.ContextID, BodyJSON: []byte("bad"), CreatedAt: "now"}); err != nil {
		t.Fatalf("put ctx: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/pack/"+resp.ReceiptID, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}

	// Fix context, break decision JSON => 500
	if err := svc.Ledger.PutContext(ledger.ContextRecord{ContextID: receipt.ContextID, BodyJSON: []byte(`{"context_id":"x"}`), CreatedAt: "now"}); err != nil {
		t.Fatalf("put ctx: %v", err)
	}
	if err := svc.Ledger.PutDecision(ledger.DecisionRecord{DecisionID: receipt.DecisionID, ContextID: receipt.ContextID, PolicyHash: receipt.PolicyHash, Verdict: "allow", BodyJSON: []byte("bad"), CreatedAt: "now"}); err != nil {
		t.Fatalf("put dec: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/pack/"+resp.ReceiptID, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", res.Code)
	}

	// Ensure router still requires receipt id.
	req = httptest.NewRequest(http.MethodGet, "/v1/pack/", bytes.NewBuffer(nil))
	req.Header.Set("Authorization", "Bearer test-token")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}
