package api

import (
	"crypto/ed25519"
	"testing"

	"github.com/davidahmann/relia/internal/aws"
	"github.com/davidahmann/relia/internal/ledger"
)

func newTestService(t *testing.T, policyPath string) *AuthorizeService {
	t.Helper()

	seed := make([]byte, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	service, err := NewAuthorizeService(NewAuthorizeServiceInput{
		PolicyPath: policyPath,
		Ledger:     ledger.NewInMemoryStore(),
		Signer:     devSigner{keyID: "test", priv: priv},
		PublicKey:  pub,
		Broker:     aws.DevBroker{},
	})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	return service
}
