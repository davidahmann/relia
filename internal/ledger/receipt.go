package ledger

import (
	"fmt"

	"github.com/davidahmann/relia_oss/internal/crypto"
	"github.com/davidahmann/relia_oss/pkg/types"
)

const ReceiptSchema = "relia.receipt.v0.1"

type Signer interface {
	KeyID() string
	SignEd25519(message []byte) ([]byte, error)
}

type MakeReceiptInput struct {
	Schema    string
	CreatedAt string

	IdemKey             string
	SupersedesReceiptID *string

	ContextID  string
	DecisionID string

	Actor   types.ReceiptActor
	Request types.ReceiptRequest
	Policy  types.ReceiptPolicy

	Approval        *types.ReceiptApproval
	CredentialGrant *types.ReceiptCredentialGrant
	Outcome         types.ReceiptOutcome
}

type StoredReceipt struct {
	ReceiptID  string
	BodyDigest string
	BodyJSON   []byte
	KeyID      string
	Sig        []byte

	IdemKey             string
	CreatedAt           string
	SupersedesReceiptID *string
	ContextID           string
	DecisionID          string
	OutcomeStatus       types.OutcomeStatus
	ApprovalID          *string
	PolicyHash          string
	Final               bool
	ExpiresAt           *string
}

// MakeReceipt canonicalizes + hashes + signs a receipt body.
func MakeReceipt(in MakeReceiptInput, signer Signer) (StoredReceipt, error) {
	if in.Schema == "" {
		in.Schema = ReceiptSchema
	}
	if in.Schema != ReceiptSchema {
		return StoredReceipt{}, fmt.Errorf("invalid schema: %s", in.Schema)
	}
	if in.IdemKey == "" || in.ContextID == "" || in.DecisionID == "" || in.Policy.PolicyHash == "" {
		return StoredReceipt{}, fmt.Errorf("missing required receipt fields")
	}
	if !validOutcome(in.Outcome.Status) {
		return StoredReceipt{}, fmt.Errorf("invalid outcome status: %s", in.Outcome.Status)
	}

	body := map[string]any{
		"schema":      in.Schema,
		"created_at":  in.CreatedAt,
		"context_id":  in.ContextID,
		"decision_id": in.DecisionID,
		"actor": map[string]any{
			"kind":     in.Actor.Kind,
			"subject":  in.Actor.Subject,
			"issuer":   in.Actor.Issuer,
			"repo":     in.Actor.Repo,
			"workflow": in.Actor.Workflow,
			"run_id":   in.Actor.RunID,
			"sha":      in.Actor.SHA,
		},
		"request": map[string]any{
			"request_id": in.Request.RequestID,
			"action":     in.Request.Action,
			"resource":   in.Request.Resource,
			"env":        in.Request.Env,
			"intent":     in.Request.Intent,
		},
		"policy": map[string]any{
			"policy_id":      in.Policy.PolicyID,
			"policy_version": in.Policy.PolicyVersion,
			"policy_hash":    in.Policy.PolicyHash,
		},
		"approval":         in.Approval,
		"credential_grant": in.CredentialGrant,
		"outcome": map[string]any{
			"status":     in.Outcome.Status,
			"issued_at":  in.Outcome.IssuedAt,
			"expires_at": in.Outcome.ExpiresAt,
			"error":      in.Outcome.Error,
		},
	}

	canonical, err := crypto.Canonicalize(body)
	if err != nil {
		return StoredReceipt{}, err
	}

	digestBytes := crypto.DigestBytes(canonical)
	bodyDigest := crypto.DigestWithPrefix(canonical)

	sig, err := signer.SignEd25519(digestBytes)
	if err != nil {
		return StoredReceipt{}, err
	}

	final := isFinalOutcome(in.Outcome.Status)

	var approvalID *string
	if in.Approval != nil && in.Approval.ApprovalID != "" {
		approvalID = &in.Approval.ApprovalID
	}

	var expiresAt *string
	if in.Outcome.ExpiresAt != "" {
		expiresAt = &in.Outcome.ExpiresAt
	}

	return StoredReceipt{
		ReceiptID:           bodyDigest,
		BodyDigest:          bodyDigest,
		BodyJSON:            canonical,
		KeyID:               signer.KeyID(),
		Sig:                 sig,
		IdemKey:             in.IdemKey,
		CreatedAt:           in.CreatedAt,
		SupersedesReceiptID: in.SupersedesReceiptID,
		ContextID:           in.ContextID,
		DecisionID:          in.DecisionID,
		OutcomeStatus:       in.Outcome.Status,
		ApprovalID:          approvalID,
		PolicyHash:          in.Policy.PolicyHash,
		Final:               final,
		ExpiresAt:           expiresAt,
	}, nil
}

func validOutcome(status types.OutcomeStatus) bool {
	switch status {
	case types.OutcomeApprovalPending,
		types.OutcomeApprovalApproved,
		types.OutcomeApprovalDenied,
		types.OutcomeIssuingCredentials,
		types.OutcomeIssuedCredentials,
		types.OutcomeDenied,
		types.OutcomeIssueFailed:
		return true
	default:
		return false
	}
}

func isFinalOutcome(status types.OutcomeStatus) bool {
	switch status {
	case types.OutcomeIssuedCredentials, types.OutcomeDenied, types.OutcomeIssueFailed:
		return true
	default:
		return false
	}
}
