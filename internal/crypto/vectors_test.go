package crypto

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestReceiptBodyVector(t *testing.T) {
	bodyRaw, err := os.ReadFile("../../spec/v0.1/vectors/receipt_body.json")
	if err != nil {
		t.Fatalf("read receipt body: %v", err)
	}

	dec := json.NewDecoder(bytes.NewReader(bodyRaw))
	dec.UseNumber()
	var body any
	if err := dec.Decode(&body); err != nil {
		t.Fatalf("decode receipt body: %v", err)
	}

	canonical, err := Canonicalize(body)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	expectedDigest := readTrimmed(t, "../../spec/v0.1/vectors/expected_body_digest.txt")
	digest := DigestWithPrefix(canonical)
	if digest != expectedDigest {
		t.Fatalf("digest mismatch: got %s want %s", digest, expectedDigest)
	}

	receiptID := readTrimmed(t, "../../spec/v0.1/vectors/expected_receipt_id.txt")
	if digest != receiptID {
		t.Fatalf("receipt id mismatch: got %s want %s", digest, receiptID)
	}

	seed := bytes.Repeat([]byte{0x01}, 32)
	priv, pub, err := KeyPairFromSeed(seed)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}

	sig, err := SignEd25519(priv, DigestBytes(canonical))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	expectedSig := readTrimmed(t, "../../spec/v0.1/vectors/expected_sig.txt")
	if base64.StdEncoding.EncodeToString(sig) != expectedSig {
		t.Fatalf("signature mismatch")
	}

	ok, err := VerifyEd25519(pub, DigestBytes(canonical), sig)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatalf("expected signature to verify")
	}
}

func readTrimmed(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.TrimSpace(string(b))
}
