package crypto

import (
	"bytes"
	"testing"
)

func TestDigestAndSignVerify(t *testing.T) {
	data := []byte("test payload")
	digest := DigestBytes(data)

	seed := bytes.Repeat([]byte{0x01}, 32)
	priv, pub, err := KeyPairFromSeed(seed)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}

	sig, err := SignEd25519(priv, digest)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	ok, err := VerifyEd25519(pub, digest, sig)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatalf("expected signature to verify")
	}

	badDigest := DigestBytes([]byte("other"))
	ok, err = VerifyEd25519(pub, badDigest, sig)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatalf("expected signature to fail for different digest")
	}
}

func TestKeyPairFromSeedInvalidSize(t *testing.T) {
	_, _, err := KeyPairFromSeed([]byte{0x01})
	if err != ErrInvalidSeedSize {
		t.Fatalf("expected ErrInvalidSeedSize, got %v", err)
	}
}

func TestSignVerifyInvalidDigestLen(t *testing.T) {
	seed := bytes.Repeat([]byte{0x01}, 32)
	priv, pub, err := KeyPairFromSeed(seed)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}

	_, err = SignEd25519(priv, []byte{0x01})
	if err != ErrInvalidDigestLen {
		t.Fatalf("expected ErrInvalidDigestLen, got %v", err)
	}

	_, err = VerifyEd25519(pub, []byte{0x01}, []byte{0x02})
	if err != ErrInvalidDigestLen {
		t.Fatalf("expected ErrInvalidDigestLen, got %v", err)
	}
}
