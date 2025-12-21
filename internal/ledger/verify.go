package ledger

import (
	"crypto/ed25519"
	"errors"

	"github.com/davidahmann/relia/internal/crypto"
)

var (
	ErrReceiptDigestMismatch = errors.New("receipt digest mismatch")
	ErrReceiptSignature      = errors.New("receipt signature invalid")
)

// VerifyReceipt validates digest consistency and signature.
func VerifyReceipt(receipt StoredReceipt, publicKey ed25519.PublicKey) error {
	digestBytes := crypto.DigestBytes(receipt.BodyJSON)
	digest := crypto.DigestWithPrefix(receipt.BodyJSON)
	if receipt.BodyDigest != digest || receipt.ReceiptID != digest {
		return ErrReceiptDigestMismatch
	}

	ok, err := crypto.VerifyEd25519(publicKey, digestBytes, receipt.Sig)
	if err != nil {
		return err
	}
	if !ok {
		return ErrReceiptSignature
	}
	return nil
}
