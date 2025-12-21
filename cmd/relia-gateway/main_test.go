package main

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestNewServer(t *testing.T) {
	addr := "127.0.0.1:9999"
	srv := newServer(addr, "policies/relia.yaml", "")
	if srv.Addr != addr {
		t.Fatalf("expected addr %s, got %s", addr, srv.Addr)
	}
	if srv.Handler == nil {
		t.Fatalf("expected handler to be set")
	}
}

func TestRunDefaults(t *testing.T) {
	factory := func(addr string, policyPath string, signingSecret string) *http.Server {
		if addr != ":8080" {
			t.Fatalf("expected default addr, got %s", addr)
		}
		if policyPath != "policies/relia.yaml" {
			t.Fatalf("expected default policy path, got %s", policyPath)
		}
		if signingSecret != "" {
			t.Fatalf("expected empty signing secret")
		}
		return &http.Server{Addr: addr}
	}

	listen := func(_ *http.Server) error {
		return http.ErrServerClosed
	}

	getenv := func(string) string { return "" }
	if err := run(nil, getenv, listen, factory); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunError(t *testing.T) {
	listenErr := errors.New("listen failed")
	listen := func(_ *http.Server) error {
		return listenErr
	}

	factory := func(addr string, policyPath string, signingSecret string) *http.Server {
		return &http.Server{Addr: addr}
	}

	getenv := func(key string) string {
		if key == "RELIA_LISTEN_ADDR" {
			return "127.0.0.1:1234"
		}
		return ""
	}

	if err := run(nil, getenv, listen, factory); err == nil {
		t.Fatalf("expected error")
	}
}

func TestRunLoadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "relia.yaml")
	if err := os.WriteFile(path, []byte("listen_addr: \":9999\"\npolicy_path: \"./policies/relia.yaml\"\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	factory := func(addr string, policyPath string, signingSecret string) *http.Server {
		if addr != ":9999" {
			t.Fatalf("expected addr from config, got %s", addr)
		}
		if policyPath != "./policies/relia.yaml" {
			t.Fatalf("expected policy path from config, got %s", policyPath)
		}
		return &http.Server{Addr: addr}
	}

	listen := func(_ *http.Server) error { return http.ErrServerClosed }
	getenv := func(key string) string {
		if key == "RELIA_CONFIG_PATH" {
			return path
		}
		return ""
	}

	if err := run(nil, getenv, listen, factory); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "a", "b"); got != "a" {
		t.Fatalf("expected a, got %s", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestListenAndServeInvalidAddr(t *testing.T) {
	err := listenAndServe(&http.Server{Addr: "127.0.0.1"})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestMainNoError(t *testing.T) {
	oldRun := runFn
	oldFatal := fatalf
	defer func() {
		runFn = oldRun
		fatalf = oldFatal
	}()

	runFn = func(args []string, envFn envFn, listenFn listenFn, serverFactory serverFactory) error {
		return nil
	}

	called := false
	fatalf = func(string, ...any) {
		called = true
	}

	main()
	if called {
		t.Fatalf("unexpected fatal call")
	}
}

func TestMainError(t *testing.T) {
	oldRun := runFn
	oldFatal := fatalf
	defer func() {
		runFn = oldRun
		fatalf = oldFatal
	}()

	runFn = func(args []string, envFn envFn, listenFn listenFn, serverFactory serverFactory) error {
		return errors.New("boom")
	}

	called := false
	fatalf = func(string, ...any) {
		called = true
	}

	main()
	if !called {
		t.Fatalf("expected fatal call")
	}
}
