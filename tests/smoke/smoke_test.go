package smoke

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

func TestSmoke(t *testing.T) {
	os.Setenv("RELIA_DEV_TOKEN", "test-token")
	defer os.Unsetenv("RELIA_DEV_TOKEN")

	service, err := api.NewAuthorizeService(api.NewAuthorizeServiceInput{PolicyPath: "../../policies/relia.yaml"})
	if err != nil {
		t.Fatalf("authorize service: %v", err)
	}

	router := api.NewRouter(&api.Handler{
		Auth:             auth.NewAuthenticatorFromEnv(),
		AuthorizeService: service,
		SlackHandler:     nil,
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	// auth gate sanity check
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/v1/verify/anything", nil)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	receiptID := authorize(t, srv.URL)
	verify(t, srv.URL, receiptID)
	pack(t, srv.URL, receiptID)
}

func authorize(t *testing.T, baseURL string) string {
	t.Helper()

	body := bytes.NewBufferString(`{"action":"terraform.apply","resource":"res","env":"dev"}`)
	req, err := http.NewRequest(http.MethodPost, baseURL+"/v1/authorize", body)
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
		Verdict   string `json:"verdict"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.ReceiptID == "" {
		t.Fatalf("missing receipt_id")
	}
	if payload.Verdict == "" {
		t.Fatalf("missing verdict")
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

	found := false
	for _, f := range reader.File {
		if f.Name == "manifest.json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected manifest.json in pack")
	}
}
