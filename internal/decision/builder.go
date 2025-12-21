package decision

import (
	"github.com/davidahmann/relia_oss/internal/crypto"
	"github.com/davidahmann/relia_oss/pkg/types"
)

const DecisionSchema = "lumyn.decision.v0.1"

// BuildDecision builds a decision record and computes its decision_id.
func BuildDecision(contextID string, policy types.DecisionPolicy, verdict string, reasonCodes []string, requiresApproval bool, risk string, createdAt string) (types.DecisionRecord, error) {
	record := types.DecisionRecord{
		Schema:           DecisionSchema,
		CreatedAt:        createdAt,
		ContextID:        contextID,
		Policy:           policy,
		Verdict:          verdict,
		ReasonCodes:      reasonCodes,
		RequiresApproval: requiresApproval,
		Risk:             risk,
	}

	signingView := map[string]any{
		"schema":     record.Schema,
		"created_at": record.CreatedAt,
		"context_id": record.ContextID,
		"policy": map[string]any{
			"policy_id":      record.Policy.PolicyID,
			"policy_version": record.Policy.PolicyVersion,
			"policy_hash":    record.Policy.PolicyHash,
		},
		"verdict":           record.Verdict,
		"reason_codes":      record.ReasonCodes,
		"requires_approval": record.RequiresApproval,
		"risk":              record.Risk,
	}

	canonical, err := crypto.Canonicalize(signingView)
	if err != nil {
		return types.DecisionRecord{}, err
	}

	record.DecisionID = crypto.DigestWithPrefix(canonical)
	return record, nil
}
