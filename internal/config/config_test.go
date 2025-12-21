package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "relia.yaml")

	os.Setenv("SLACK_SIGNING_SECRET", "secret")
	defer os.Unsetenv("SLACK_SIGNING_SECRET")

	data := `
listen_addr: ":8080"
policy_path: "./policies/relia.yaml"
slack:
  enabled: true
  signing_secret: "${SLACK_SIGNING_SECRET}"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Slack.SigningSecret != "secret" {
		t.Fatalf("expected expanded signing secret")
	}
}

func TestValidateMissingFields(t *testing.T) {
	if err := (Config{}).Validate(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateSlackRequiresSecret(t *testing.T) {
	cfg := Config{ListenAddr: ":8080", PolicyPath: "policies/relia.yaml", Slack: SlackConfig{Enabled: true}}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestValidateDBRequiresDSN(t *testing.T) {
	cfg := Config{ListenAddr: ":8080", PolicyPath: "policies/relia.yaml", DB: DBConfig{Driver: "sqlite"}}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load("does-not-exist.yaml"); err == nil {
		t.Fatalf("expected error")
	}
}
