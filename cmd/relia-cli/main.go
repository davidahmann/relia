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
	exitFn(run(os.Args, os.Stdout, os.Stderr))
}

var exitFn = os.Exit

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		usage(stderr)
		return 2
	}

	switch args[1] {
	case "verify":
		return handleVerify(args[2:], stdout, stderr)
	case "pack":
		return handlePack(args[2:], stdout, stderr)
	case "policy":
		return handlePolicy(args[2:], stdout, stderr)
	default:
		usage(stderr)
		return 2
	}
}

func handleVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", envOrDefault("RELIA_ADDR", defaultAddr), "Relia API address")
	jsonOut := fs.Bool("json", false, "print raw JSON response")
	token := fs.String("token", envOrDefault("RELIA_TOKEN", os.Getenv("RELIA_DEV_TOKEN")), "bearer token")
	if err := fs.Parse(args); err != nil {
		fs.Usage()
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "verify requires <receipt_id>")
		fs.Usage()
		return 2
	}
	receiptID := fs.Arg(0)

	respBody, status, err := httpGet(http.DefaultClient, *addr+"/v1/verify/"+receiptID, *token)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}

	if *jsonOut {
		_, _ = stdout.Write(respBody)
		return 0
	}

	var payload struct {
		ReceiptID string `json:"receipt_id"`
		Valid     bool   `json:"valid"`
		Error     string `json:"error,omitempty"`
		Grade     string `json:"grade,omitempty"`
	}
	if err := json.Unmarshal(respBody, &payload); err != nil {
		fmt.Fprintln(stderr, "invalid response:", err)
		return 1
	}

	if status != http.StatusOK {
		fmt.Fprintf(stderr, "verify failed: %s\n", strings.TrimSpace(string(respBody)))
		return 1
	}

	if payload.Valid {
		if payload.Grade != "" {
			fmt.Fprintf(stdout, "valid=true receipt_id=%s grade=%s\n", payload.ReceiptID, payload.Grade)
		} else {
			fmt.Fprintf(stdout, "valid=true receipt_id=%s\n", payload.ReceiptID)
		}
		return 0
	}
	fmt.Fprintf(stdout, "valid=false receipt_id=%s error=%s\n", payload.ReceiptID, payload.Error)
	return 1
}

func handlePack(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", envOrDefault("RELIA_ADDR", defaultAddr), "Relia API address")
	outPath := fs.String("out", "relia-pack.zip", "output zip path")
	token := fs.String("token", envOrDefault("RELIA_TOKEN", os.Getenv("RELIA_DEV_TOKEN")), "bearer token")
	if err := fs.Parse(args); err != nil {
		fs.Usage()
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "pack requires <receipt_id>")
		fs.Usage()
		return 2
	}
	receiptID := fs.Arg(0)

	respBody, status, err := httpGet(http.DefaultClient, *addr+"/v1/pack/"+receiptID, *token)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if status != http.StatusOK {
		fmt.Fprintf(stderr, "pack failed: %s\n", strings.TrimSpace(string(respBody)))
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o750); err != nil && filepath.Dir(*outPath) != "." {
		fmt.Fprintln(stderr, "output dir:", err)
		return 1
	}
	if err := os.WriteFile(*outPath, respBody, 0o600); err != nil {
		fmt.Fprintln(stderr, "write output:", err)
		return 1
	}
	fmt.Fprintf(stdout, "wrote %s\n", *outPath)
	return 0
}

func handlePolicy(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "lint":
		fs := flag.NewFlagSet("policy lint", flag.ContinueOnError)
		fs.SetOutput(stderr)
		if err := fs.Parse(args[1:]); err != nil {
			fs.Usage()
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(stderr, "policy lint requires <policy_path>")
			fs.Usage()
			return 2
		}
		path := fs.Arg(0)
		loaded, err := policy.LoadPolicy(path)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		fmt.Fprintf(stdout, "ok policy_id=%s policy_hash=%s\n", loaded.Policy.PolicyID, loaded.Hash)
		return 0
	case "test":
		fs := flag.NewFlagSet("policy test", flag.ContinueOnError)
		fs.SetOutput(stderr)
		policyPath := fs.String("policy", "", "path to policy yaml (required)")
		action := fs.String("action", "", "action to test (required)")
		resource := fs.String("resource", "", "resource to test (required)")
		envName := fs.String("env", "", "environment to test (required)")
		jsonOut := fs.Bool("json", false, "print raw JSON output")
		if err := fs.Parse(args[1:]); err != nil {
			fs.Usage()
			return 2
		}
		if *policyPath == "" || *action == "" || *resource == "" || *envName == "" {
			fmt.Fprintln(stderr, "policy test requires --policy --action --resource --env")
			fs.Usage()
			return 2
		}
		loaded, err := policy.LoadPolicy(*policyPath)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}

		decision := policy.Evaluate(loaded.Policy, loaded.Hash, policy.Input{
			Action:   *action,
			Resource: *resource,
			Env:      *envName,
		})
		if *jsonOut {
			out, _ := json.MarshalIndent(decision, "", "  ")
			_, _ = stdout.Write(append(out, '\n'))
			return 0
		}

		fmt.Fprintf(stdout, "verdict=%s requires_approval=%t\n", decision.Verdict, decision.RequireApproval)
		if decision.MatchedRuleID != "" {
			fmt.Fprintf(stdout, "matched_rule=%s\n", decision.MatchedRuleID)
		} else {
			fmt.Fprintf(stdout, "matched_rule=<defaults>\n")
		}
		fmt.Fprintf(stdout, "policy_id=%s policy_version=%s policy_hash=%s\n", decision.PolicyID, decision.PolicyVersion, decision.PolicyHash)
		if decision.AWSRoleARN != "" {
			fmt.Fprintf(stdout, "aws_role_arn=%s\n", decision.AWSRoleARN)
		}
		if decision.TTLSeconds != 0 {
			fmt.Fprintf(stdout, "ttl_seconds=%d\n", decision.TTLSeconds)
		}
		if decision.Risk != "" {
			fmt.Fprintf(stdout, "risk=%s\n", decision.Risk)
		}
		if decision.Reason != "" {
			fmt.Fprintf(stdout, "reason=%s\n", decision.Reason)
		}
		if len(decision.ReasonCodes) > 0 {
			fmt.Fprintf(stdout, "reason_codes=%s\n", strings.Join(decision.ReasonCodes, ","))
		}
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func httpGet(client *http.Client, url string, token string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
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

func usage(w io.Writer) {
	fmt.Fprint(w, `Relia CLI

Usage:
  relia verify <receipt_id> [--addr URL] [--json] [--token TOKEN]
  relia pack <receipt_id> --out relia-pack.zip [--addr URL] [--token TOKEN]
  relia policy lint <policy_path>
  relia policy test --policy PATH --action ACTION --resource RESOURCE --env ENV [--json]
`)
}
