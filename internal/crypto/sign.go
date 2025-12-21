package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
)

// DigestBytes returns the raw SHA-256 digest bytes.
func DigestBytes(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}

// DigestHex returns the SHA-256 digest as lowercase hex.
func DigestHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// DigestWithPrefix returns the SHA-256 digest with the "sha256:" prefix.
func DigestWithPrefix(data []byte) string {
	return "sha256:" + DigestHex(data)
}

// SignEd25519 signs a digest using Ed25519.
func SignEd25519(privateKey ed25519.PrivateKey, digest []byte) ([]byte, error) {
	if len(digest) != sha256.Size {
		return nil, ErrInvalidDigestLen
	}
	return ed25519.Sign(privateKey, digest), nil
}

// VerifyEd25519 verifies a digest signature using Ed25519.
func VerifyEd25519(publicKey ed25519.PublicKey, digest, sig []byte) (bool, error) {
	if len(digest) != sha256.Size {
		return false, ErrInvalidDigestLen
	}
	return ed25519.Verify(publicKey, digest, sig), nil
}
