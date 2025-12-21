package context

import (
	"github.com/davidahmann/relia_oss/internal/crypto"
	"github.com/davidahmann/relia_oss/pkg/types"
)

const ContextSchema = "fabra.context.v0.1"

// BuildContext builds a context record and computes its context_id.
func BuildContext(source types.ContextSource, inputs types.ContextInputs, evidence types.ContextEvidence, createdAt string) (types.ContextRecord, error) {
	record := types.ContextRecord{
		Schema:    ContextSchema,
		CreatedAt: createdAt,
		Source:    source,
		Inputs:    inputs,
		Evidence:  evidence,
	}

	signingView := map[string]any{
		"schema":     record.Schema,
		"created_at": record.CreatedAt,
		"source": map[string]any{
			"kind":     record.Source.Kind,
			"repo":     record.Source.Repo,
			"workflow": record.Source.Workflow,
			"run_id":   record.Source.RunID,
			"actor":    record.Source.Actor,
			"ref":      record.Source.Ref,
			"sha":      record.Source.SHA,
		},
		"inputs": map[string]any{
			"action":   record.Inputs.Action,
			"resource": record.Inputs.Resource,
			"env":      record.Inputs.Env,
			"intent":   record.Inputs.Intent,
		},
		"evidence": map[string]any{
			"plan_digest": record.Evidence.PlanDigest,
			"diff_url":    record.Evidence.DiffURL,
		},
	}

	canonical, err := crypto.Canonicalize(signingView)
	if err != nil {
		return types.ContextRecord{}, err
	}

	record.ContextID = crypto.DigestWithPrefix(canonical)
	return record, nil
}
