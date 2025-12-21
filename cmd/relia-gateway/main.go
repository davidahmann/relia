package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davidahmann/relia/internal/api"
	"github.com/davidahmann/relia/internal/auth"
	"github.com/davidahmann/relia/internal/slack"
)

func main() {
	if err := runFn(os.Getenv, listenAndServe, newServer); err != nil {
		fatalf("server error: %v", err)
	}
}

var runFn = run
var fatalf = log.Fatalf

func newServer(addr string, policyPath string, signingSecret string) *http.Server {
	authorizeService, err := api.NewAuthorizeService(policyPath)
	if err != nil {
		log.Fatalf("authorize service error: %v", err)
	}

	slackHandler := &slack.InteractionHandler{
		SigningSecret: signingSecret,
		Approver:      authorizeService,
	}

	h := &api.Handler{
		Auth:             auth.NewAuthenticatorFromEnv(),
		AuthorizeService: authorizeService,
		SlackHandler:     slackHandler,
	}
	return &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(h),
		ReadHeaderTimeout: 5 * time.Second,
	}
}

type envFn func(string) string
type listenFn func(*http.Server) error
type serverFactory func(addr string, policyPath string, signingSecret string) *http.Server

func run(getenv envFn, listen listenFn, factory serverFactory) error {
	addr := getenv("RELIA_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	policyPath := getenv("RELIA_POLICY_PATH")
	if policyPath == "" {
		policyPath = "policies/relia.yaml"
	}

	signingSecret := getenv("RELIA_SLACK_SIGNING_SECRET")

	server := factory(addr, policyPath, signingSecret)

	log.Printf("relia-gateway listening on %s", addr)
	if err := listen(server); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func listenAndServe(server *http.Server) error {
	return server.ListenAndServe()
}
