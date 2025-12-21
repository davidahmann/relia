package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEd25519PrivateKey_SeedHex(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, []byte("hex:"+hex.EncodeToString(seed)), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	priv, pub, err := LoadEd25519PrivateKey(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(priv) != ed25519.PrivateKeySize || len(pub) != ed25519.PublicKeySize {
		t.Fatalf("unexpected key sizes: priv=%d pub=%d", len(priv), len(pub))
	}
}

func TestLoadEd25519PrivateKey_PrivateKeyBase64(t *testing.T) {
	seed := make([]byte, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)

	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, []byte("base64:"+base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	priv2, pub2, err := LoadEd25519PrivateKey(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(priv2) != string(priv) {
		t.Fatalf("private key mismatch")
	}
	if len(pub2) != ed25519.PublicKeySize {
		t.Fatalf("unexpected pub size: %d", len(pub2))
	}
}

func TestDecodeBytesErrors(t *testing.T) {
	if _, err := decodeBytes([]byte("")); err == nil {
		t.Fatalf("expected error for empty")
	}
	if _, err := decodeBytes([]byte("not-a-key")); err == nil {
		t.Fatalf("expected error for unrecognized encoding")
	}
}
