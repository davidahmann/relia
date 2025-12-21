package crypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// LoadEd25519PrivateKey loads an Ed25519 private key from a file.
// Supported formats:
// - raw 64-byte private key
// - raw 32-byte seed
// - hex or base64 encoding of either form
func LoadEd25519PrivateKey(path string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	// #nosec G304 -- path is operator-configured.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	data, err := decodeBytes(raw)
	if err != nil {
		return nil, nil, err
	}

	switch len(data) {
	case ed25519.PrivateKeySize:
		priv := ed25519.PrivateKey(data)
		pub := priv.Public().(ed25519.PublicKey)
		return priv, pub, nil
	case ed25519.SeedSize:
		priv := ed25519.NewKeyFromSeed(data)
		pub := priv.Public().(ed25519.PublicKey)
		return priv, pub, nil
	default:
		return nil, nil, fmt.Errorf("unsupported private key length: %d", len(data))
	}
}

func decodeBytes(raw []byte) ([]byte, error) {
	trim := strings.TrimSpace(string(raw))
	if trim == "" {
		return nil, fmt.Errorf("empty key file")
	}
	if strings.HasPrefix(trim, "base64:") {
		return base64.StdEncoding.DecodeString(strings.TrimPrefix(trim, "base64:"))
	}
	if strings.HasPrefix(trim, "hex:") {
		return hex.DecodeString(strings.TrimPrefix(trim, "hex:"))
	}

	// try raw bytes first (common when file is binary)
	if len(raw) == ed25519.PrivateKeySize || len(raw) == ed25519.SeedSize {
		return raw, nil
	}

	// try hex
	if out, err := hex.DecodeString(trim); err == nil {
		return out, nil
	}
	// try base64
	if out, err := base64.StdEncoding.DecodeString(trim); err == nil {
		return out, nil
	}

	// try rawurl base64
	if out, err := base64.RawURLEncoding.DecodeString(trim); err == nil {
		return out, nil
	}
	return nil, fmt.Errorf("unrecognized key encoding")
}
