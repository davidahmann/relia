//go:build e2e

package e2e

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/davidahmann/relia/internal/api"
	"github.com/davidahmann/relia/internal/auth"
)

func TestE2EAuthorizeVerifyPack(t *testing.T) {
	os.Setenv("RELIA_DEV_TOKEN", "test-token")
	defer os.Unsetenv("RELIA_DEV_TOKEN")

	service, err := api.NewAuthorizeService("../../policies/relia.yaml")
	if err != nil {
		t.Fatalf("authorize service: %v", err)
	}

	router := api.NewRouter(&api.Handler{
		Auth:             auth.NewAuthenticatorFromEnv(),
		AuthorizeService: service,
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	receiptID := authorize(t, srv.URL, `{"action":"terraform.apply","resource":"res","env":"dev","request_id":"req-1"}`)
	receiptID2 := authorize(t, srv.URL, `{"action":"terraform.apply","resource":"res","env":"dev","request_id":"req-1"}`)
	if receiptID != receiptID2 {
		t.Fatalf("expected idempotent receipt_id, got %s vs %s", receiptID, receiptID2)
	}

	verify(t, srv.URL, receiptID)
	pack(t, srv.URL, receiptID)
}

func authorize(t *testing.T, baseURL, body string) string {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/authorize", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("authorize status: %d", res.StatusCode)
	}

	var payload struct {
		ReceiptID string `json:"receipt_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.ReceiptID == "" {
		t.Fatalf("missing receipt_id")
	}
	return payload.ReceiptID
}

func verify(t *testing.T, baseURL, receiptID string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/verify/"+receiptID, nil)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("verify status: %d", res.StatusCode)
	}

	var payload struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !payload.Valid {
		t.Fatalf("expected valid receipt")
	}
}

func pack(t *testing.T, baseURL, receiptID string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/v1/pack/"+receiptID, nil)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("pack status: %d", res.StatusCode)
	}

	zipBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("zip: %v", err)
	}

	want := map[string]bool{
		"receipt.json":   false,
		"context.json":   false,
		"decision.json":  false,
		"policy.yaml":    false,
		"manifest.json":  false,
		"sha256sums.txt": false,
	}
	for _, f := range reader.File {
		if _, ok := want[f.Name]; ok {
			want[f.Name] = true
		}
	}
	for name, ok := range want {
		if !ok {
			t.Fatalf("missing %s", name)
		}
	}
}
