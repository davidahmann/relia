package api

import (
	"crypto/ed25519"
	"fmt"
	"testing"

	"github.com/davidahmann/relia/internal/ledger"
)

func BenchmarkAuthorize(b *testing.B) {
	seed := make([]byte, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	pub := priv.Public().(ed25519.PublicKey)

	service, err := NewAuthorizeService(NewAuthorizeServiceInput{
		PolicyPath: "../../policies/relia.yaml",
		Ledger:     ledger.NewInMemoryStore(),
		Signer:     devSigner{keyID: "bench", priv: priv},
		PublicKey:  pub,
	})
	if err != nil {
		b.Fatalf("service: %v", err)
	}

	claims := ActorContext{
		Subject:  "repo:org/repo:ref:refs/heads/main",
		Issuer:   "relia-dev",
		Repo:     "org/repo",
		Workflow: "terraform-prod",
		RunID:    "123456",
		SHA:      "abcdef123",
	}
	req := AuthorizeRequest{Action: "terraform.apply", Resource: "res", Env: "dev"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if i%200 == 0 {
			service.Ledger = ledger.NewInMemoryStore()
		}
		req.RequestID = fmt.Sprintf("req-%d", i)
		if _, err := service.Authorize(claims, req, "2025-12-20T16:34:14Z"); err != nil {
			b.Fatalf("authorize: %v", err)
		}
	}
}
