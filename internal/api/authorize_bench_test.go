package api

import (
	"fmt"
	"testing"
)

func BenchmarkAuthorize(b *testing.B) {
	service, err := NewAuthorizeService("../../policies/relia.yaml")
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
			service.Store = NewInMemoryIdemStore()
			service.Artifacts = NewArtifactStore()
		}
		req.RequestID = fmt.Sprintf("req-%d", i)
		if _, err := service.Authorize(claims, req, "2025-12-20T16:34:14Z"); err != nil {
			b.Fatalf("authorize: %v", err)
		}
	}
}
