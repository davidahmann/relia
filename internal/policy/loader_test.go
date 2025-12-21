package policy

import (
	"os"
	"testing"

	"github.com/davidahmann/relia_oss/internal/crypto"
)

func TestLoadPolicy(t *testing.T) {
	loaded, err := LoadPolicy("../../policies/relia.yaml")
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}

	if loaded.Policy.PolicyID == "" {
		t.Fatalf("policy id missing")
	}

	data, err := os.ReadFile("../../policies/relia.yaml")
	if err != nil {
		t.Fatalf("read policy: %v", err)
	}

	expected := crypto.DigestWithPrefix(data)
	if loaded.Hash != expected {
		t.Fatalf("policy hash mismatch: got %s want %s", loaded.Hash, expected)
	}
}
