package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Relia CLI") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestVerifySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"receipt_id":"r1","valid":true}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "verify", "--addr", server.URL, "--token", "test-token", "r1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid=true") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestVerifyInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{invalid"))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "verify", "--addr", server.URL, "r1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "invalid response") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestVerifyJSONOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"receipt_id":"r1","valid":true}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "verify", "--addr", server.URL, "--json", "r1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"receipt_id":"r1"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestVerifyNonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "verify", "--addr", server.URL, "r1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "verify failed") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestVerifyValidFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"receipt_id":"r1","valid":false,"error":"bad"}`))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "verify", "--addr", server.URL, "r1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(stdout.String(), "valid=false") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestPackSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("zip-content"))
	}))
	defer server.Close()

	dir := t.TempDir()
	outPath := filepath.Join(dir, "out", "relia-pack.zip")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "pack", "--addr", server.URL, "--out", outPath, "r1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d: %s", code, stderr.String())
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(stdout.String(), "wrote") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestPackFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("nope"))
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "pack", "--addr", server.URL, "r1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "pack failed") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestPolicyLint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(path, []byte("policy_id: relia-default\npolicy_version: \"2025-12-20\"\n"), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "policy", "lint", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected code 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ok policy_id=") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestPolicyLintMissingArg(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "policy", "lint"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected code 2, got %d", code)
	}
}

func TestPolicyUnknownSubcommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "policy", "unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected code 2, got %d", code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"relia", "unknown"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected code 2, got %d", code)
	}
}

func TestEnvOrDefault(t *testing.T) {
	os.Setenv("RELIA_TEST_ENV", "value")
	defer os.Unsetenv("RELIA_TEST_ENV")

	if got := envOrDefault("RELIA_TEST_ENV", "fallback"); got != "value" {
		t.Fatalf("expected env value, got %s", got)
	}
	if got := envOrDefault("RELIA_TEST_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %s", got)
	}
}

func TestMainExitCode(t *testing.T) {
	oldExit := exitFn
	oldArgs := os.Args
	defer func() {
		exitFn = oldExit
		os.Args = oldArgs
	}()

	var code int
	exitFn = func(c int) {
		code = c
	}
	os.Args = []string{"relia"}

	main()

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
