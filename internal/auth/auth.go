package auth

import (
	"errors"
	"net/http"
	"os"
	"strings"
)

var (
	ErrMissingBearer = errors.New("missing bearer token")
	ErrInvalidToken  = errors.New("invalid token")
)

type Claims struct {
	Subject  string
	Issuer   string
	Repo     string
	Workflow string
	RunID    string
	SHA      string
	Token    string
}

type Authenticator interface {
	Authenticate(r *http.Request) (Claims, error)
}

type MultiAuthenticator struct {
	DevToken string
	OIDC     *GitHubOIDCAuthenticator
}

func NewAuthenticatorFromEnv() *MultiAuthenticator {
	audience := os.Getenv("RELIA_GITHUB_OIDC_AUDIENCE")
	if audience == "" {
		audience = "relia"
	}
	return &MultiAuthenticator{
		DevToken: os.Getenv("RELIA_DEV_TOKEN"),
		OIDC:     NewGitHubOIDCAuthenticator(audience),
	}
}

func (a *MultiAuthenticator) Authenticate(r *http.Request) (Claims, error) {
	bearer, err := extractBearer(r)
	if err != nil {
		return Claims{}, err
	}

	if a.DevToken != "" {
		if bearer == a.DevToken {
			return Claims{Subject: "dev", Issuer: "relia-dev", Repo: "dev/repo", Workflow: "dev", RunID: "dev", SHA: "dev", Token: bearer}, nil
		}
	}

	if a.OIDC != nil {
		claims, err := a.OIDC.AuthenticateBearer(bearer)
		if err == nil {
			claims.Token = bearer
			return claims, nil
		}
	}

	return Claims{}, ErrInvalidToken
}

func extractBearer(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrMissingBearer
	}
	if !strings.HasPrefix(auth, "Bearer ") {
		return "", ErrInvalidToken
	}
	token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	if token == "" {
		return "", ErrInvalidToken
	}
	return token, nil
}
