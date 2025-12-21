package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/davidahmann/relia/internal/policy"
)

const defaultAddr = "http://localhost:8080"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "verify":
		handleVerify(os.Args[2:])
	case "pack":
		handlePack(os.Args[2:])
	case "policy":
		handlePolicy(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func handleVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	addr := fs.String("addr", envOrDefault("RELIA_ADDR", defaultAddr), "Relia API address")
	jsonOut := fs.Bool("json", false, "print raw JSON response")
	token := fs.String("token", envOrDefault("RELIA_TOKEN", os.Getenv("RELIA_DEV_TOKEN")), "bearer token")
	if err := fs.Parse(args); err != nil {
		fs.Usage()
		os.Exit(2)
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "verify requires <receipt_id>")
		fs.Usage()
		os.Exit(2)
	}
	receiptID := fs.Arg(0)

	respBody, status, err := httpGet(*addr+"/v1/verify/"+receiptID, *token)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if *jsonOut {
		_, _ = os.Stdout.Write(respBody)
		return
	}

	var payload struct {
		ReceiptID string `json:"receipt_id"`
		Valid     bool   `json:"valid"`
		Error     string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		fmt.Fprintln(os.Stderr, "invalid response:", err)
		os.Exit(1)
	}

	if status != http.StatusOK {
		fmt.Fprintf(os.Stderr, "verify failed: %s\n", strings.TrimSpace(string(respBody)))
		os.Exit(1)
	}

	if payload.Valid {
		fmt.Printf("valid=true receipt_id=%s\n", payload.ReceiptID)
		return
	}
	fmt.Printf("valid=false receipt_id=%s error=%s\n", payload.ReceiptID, payload.Error)
	os.Exit(1)
}

func handlePack(args []string) {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	addr := fs.String("addr", envOrDefault("RELIA_ADDR", defaultAddr), "Relia API address")
	outPath := fs.String("out", "relia-pack.zip", "output zip path")
	token := fs.String("token", envOrDefault("RELIA_TOKEN", os.Getenv("RELIA_DEV_TOKEN")), "bearer token")
	if err := fs.Parse(args); err != nil {
		fs.Usage()
		os.Exit(2)
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "pack requires <receipt_id>")
		fs.Usage()
		os.Exit(2)
	}
	receiptID := fs.Arg(0)

	respBody, status, err := httpGet(*addr+"/v1/pack/"+receiptID, *token)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if status != http.StatusOK {
		fmt.Fprintf(os.Stderr, "pack failed: %s\n", strings.TrimSpace(string(respBody)))
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o750); err != nil && filepath.Dir(*outPath) != "." {
		fmt.Fprintln(os.Stderr, "output dir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outPath, respBody, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "write output:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", *outPath)
}

func handlePolicy(args []string) {
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	switch args[0] {
	case "lint":
		fs := flag.NewFlagSet("policy lint", flag.ContinueOnError)
		if err := fs.Parse(args[1:]); err != nil {
			fs.Usage()
			os.Exit(2)
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "policy lint requires <policy_path>")
			fs.Usage()
			os.Exit(2)
		}
		path := fs.Arg(0)
		loaded, err := policy.LoadPolicy(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Printf("ok policy_id=%s policy_hash=%s\n", loaded.Policy.PolicyID, loaded.Hash)
	default:
		usage()
		os.Exit(2)
	}
}

func httpGet(url string, token string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Print(`Relia CLI

Usage:
  relia verify <receipt_id> [--addr URL] [--json] [--token TOKEN]
  relia pack <receipt_id> --out relia-pack.zip [--addr URL] [--token TOKEN]
  relia policy lint <policy_path>
`)
}
