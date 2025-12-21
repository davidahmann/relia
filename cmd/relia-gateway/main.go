package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davidahmann/relia_oss/internal/api"
	"github.com/davidahmann/relia_oss/internal/auth"
)

func main() {
	addr := os.Getenv("RELIA_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	policyPath := os.Getenv("RELIA_POLICY_PATH")
	if policyPath == "" {
		policyPath = "policies/relia.yaml"
	}

	server := newServer(addr, policyPath)

	log.Printf("relia-gateway listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func newServer(addr string, policyPath string) *http.Server {
	authorizeService, err := api.NewAuthorizeService(policyPath)
	if err != nil {
		log.Fatalf("authorize service error: %v", err)
	}

	h := &api.Handler{
		Auth:             auth.NewAuthenticatorFromEnv(),
		AuthorizeService: authorizeService,
	}
	return &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(h),
		ReadHeaderTimeout: 5 * time.Second,
	}
}
