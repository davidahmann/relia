package grade

import (
	"encoding/json"
	"strings"

	"github.com/davidahmann/relia/internal/ledger"
	"github.com/davidahmann/relia/pkg/types"
)

type Result struct {
	Grade   string
	Reasons []string
}

type Input struct {
	Valid    bool
	Receipt  ledger.StoredReceipt
	Context  *types.ContextRecord
	Decision *types.DecisionRecord
}

type receiptBody struct {
	Policy          types.ReceiptPolicy           `json:"policy"`
	Approval        *types.ReceiptApproval        `json:"approval,omitempty"`
	CredentialGrant *types.ReceiptCredentialGrant `json:"credential_grant,omitempty"`
	Outcome         types.ReceiptOutcome          `json:"outcome"`
}

func Evaluate(in Input) Result {
	if !in.Valid {
		return Result{Grade: "F", Reasons: []string{"invalid_signature"}}
	}

	var body receiptBody
	_ = json.Unmarshal(in.Receipt.BodyJSON, &body)

	missing := map[string]bool{}

	if body.Policy.PolicyHash == "" && in.Receipt.PolicyHash == "" {
		missing["policy_hash"] = true
	}

	if in.Decision != nil {
		if in.Decision.Policy.PolicyHash == "" {
			missing["decision_policy_hash"] = true
		}
	}

	if in.Context != nil {
		if strings.TrimSpace(in.Context.Evidence.PlanDigest) == "" {
			missing["plan_digest"] = true
		}
		if strings.TrimSpace(in.Context.Evidence.DiffURL) == "" {
			missing["diff_url"] = true
		}
	}

	if body.CredentialGrant == nil || strings.TrimSpace(body.CredentialGrant.RoleARN) == "" {
		missing["role_arn"] = true
	}

	if body.CredentialGrant == nil || body.CredentialGrant.TTLSeconds <= 0 {
		missing["ttl"] = true
	}

	approvalRequired := false
	approvalApproved := false
	if in.Decision != nil {
		approvalRequired = in.Decision.RequiresApproval
	}
	if body.Approval != nil && body.Approval.Required {
		approvalRequired = true
	}
	if body.Approval != nil && strings.EqualFold(body.Approval.Status, "approved") {
		approvalApproved = true
	}

	if approvalRequired && !approvalApproved {
		missing["approval"] = true
	}

	// Heuristic grading.
	grade := "A"
	switch {
	case missing["policy_hash"] || missing["decision_policy_hash"]:
		grade = "F"
	case missing["approval"]:
		grade = "D"
	case missing["plan_digest"] && missing["diff_url"]:
		grade = "C"
	case missing["plan_digest"] || missing["diff_url"] || missing["role_arn"] || missing["ttl"]:
		grade = "B"
	}

	reasons := []string{}
	for k, v := range missing {
		if v {
			reasons = append(reasons, "missing_"+k)
		}
	}

	return Result{Grade: grade, Reasons: reasons}
}
