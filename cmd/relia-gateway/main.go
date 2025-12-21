package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/davidahmann/relia/internal/api"
	"github.com/davidahmann/relia/internal/auth"
	"github.com/davidahmann/relia/internal/config"
	"github.com/davidahmann/relia/internal/slack"
)

func main() {
	if err := runFn(os.Args[1:], os.Getenv, listenAndServe, newServer); err != nil {
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

func run(args []string, getenv envFn, listen listenFn, factory serverFactory) error {
	fs := flag.NewFlagSet("relia-gateway", flag.ContinueOnError)
	configPath := fs.String("config", "", "path to relia config file")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfgFile := *configPath
	if cfgFile == "" {
		cfgFile = getenv("RELIA_CONFIG_PATH")
	}

	var cfg config.Config
	if cfgFile != "" {
		loaded, err := config.Load(cfgFile)
		if err != nil {
			return err
		}
		cfg = loaded
	}

	addr := firstNonEmpty(getenv("RELIA_LISTEN_ADDR"), cfg.ListenAddr, ":8080")

	policyPath := firstNonEmpty(getenv("RELIA_POLICY_PATH"), cfg.PolicyPath, "policies/relia.yaml")

	signingSecret := firstNonEmpty(getenv("RELIA_SLACK_SIGNING_SECRET"), cfg.Slack.SigningSecret, "")

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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
