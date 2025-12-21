package api

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/davidahmann/relia_oss/internal/context"
	"github.com/davidahmann/relia_oss/internal/decision"
	"github.com/davidahmann/relia_oss/internal/ledger"
	"github.com/davidahmann/relia_oss/internal/policy"
	"github.com/davidahmann/relia_oss/pkg/types"
)

type AuthorizeService struct {
	PolicyPath string
	Store      *InMemoryIdemStore
	Signer     ledger.Signer
}

type AuthorizeResponse struct {
	Verdict    string `json:"verdict"`
	ContextID  string `json:"context_id"`
	DecisionID string `json:"decision_id"`
	ReceiptID  string `json:"receipt_id"`
	Approval   *struct {
		ApprovalID string `json:"approval_id"`
		Status     string `json:"status"`
	} `json:"approval,omitempty"`
	Error string `json:"error,omitempty"`
}

func NewAuthorizeService(policyPath string) (*AuthorizeService, error) {
	seed := make([]byte, ed25519.SeedSize)
	if _, err := rand.Read(seed); err != nil {
		return nil, err
	}
	priv := ed25519.NewKeyFromSeed(seed)

	return &AuthorizeService{
		PolicyPath: policyPath,
		Store:      NewInMemoryIdemStore(),
		Signer:     devSigner{keyID: "dev", priv: priv},
	}, nil
}

func (s *AuthorizeService) Authorize(claims ActorContext, req AuthorizeRequest, createdAt string) (AuthorizeResponse, error) {
	idemKey, err := ComputeIdemKey(claims, req)
	if err != nil {
		return AuthorizeResponse{}, err
	}

	if rec, ok := s.Store.Get(idemKey); ok {
		switch rec.Status {
		case IdemAllowed:
			return AuthorizeResponse{Verdict: string(VerdictAllow), ContextID: rec.ContextID, DecisionID: rec.DecisionID, ReceiptID: rec.ReceiptID}, nil
		case IdemDenied:
			return AuthorizeResponse{Verdict: string(VerdictDeny), ContextID: rec.ContextID, DecisionID: rec.DecisionID, ReceiptID: rec.ReceiptID}, nil
		case IdemPendingApproval:
			return AuthorizeResponse{Verdict: string(VerdictRequireApproval), ContextID: rec.ContextID, DecisionID: rec.DecisionID, ReceiptID: rec.ReceiptID, Approval: &struct {
				ApprovalID string `json:"approval_id"`
				Status     string `json:"status"`
			}{ApprovalID: rec.ApprovalID, Status: string(ApprovalPending)}}, nil
		case IdemApprovedReady, IdemIssuing:
			return AuthorizeResponse{Verdict: string(VerdictAllow), ContextID: rec.ContextID, DecisionID: rec.DecisionID, ReceiptID: rec.ReceiptID}, nil
		case IdemErrored:
			return AuthorizeResponse{Verdict: string(VerdictDeny), Error: "previous error"}, nil
		}
	}

	loaded, err := policy.LoadPolicy(s.PolicyPath)
	if err != nil {
		return AuthorizeResponse{}, err
	}

	input := policy.Input{Action: req.Action, Resource: req.Resource, Env: req.Env}
	decisionResult := policy.Evaluate(loaded.Policy, loaded.Hash, input)

	source := types.ContextSource{
		Kind:     "github_actions",
		Repo:     claims.Repo,
		Workflow: claims.Workflow,
		RunID:    claims.RunID,
		Actor:    claims.Subject,
		Ref:      "",
		SHA:      claims.SHA,
	}
	inputs := types.ContextInputs{Action: req.Action, Resource: req.Resource, Env: req.Env, Intent: req.Intent}
	evidence := types.ContextEvidence{PlanDigest: req.Evidence.PlanDigest, DiffURL: req.Evidence.DiffURL}

	ctxRecord, err := context.BuildContext(source, inputs, evidence, createdAt)
	if err != nil {
		return AuthorizeResponse{}, err
	}

	policyMeta := types.DecisionPolicy{PolicyID: loaded.Policy.PolicyID, PolicyVersion: loaded.Policy.PolicyVersion, PolicyHash: loaded.Hash}
	decRecord, err := decision.BuildDecision(ctxRecord.ContextID, policyMeta, decisionResult.Verdict, decisionResult.ReasonCodes, decisionResult.RequireApproval, decisionResult.Risk, createdAt)
	if err != nil {
		return AuthorizeResponse{}, err
	}

	verdict := DecisionVerdict(decisionResult.Verdict)
	status, action := TransitionFromDecision(verdict)

	approvalID := ""
	var approval *types.ReceiptApproval
	if action == ActionReturnPending {
		approvalID = newApprovalID()
		approval = &types.ReceiptApproval{Required: true, ApprovalID: approvalID, Status: string(ApprovalPending)}
	}

	outcome := types.ReceiptOutcome{Status: types.OutcomeDenied}
	switch action {
	case ActionReturnDenied:
		outcome.Status = types.OutcomeDenied
	case ActionReturnPending:
		outcome.Status = types.OutcomeApprovalPending
	case ActionIssueCredentials:
		outcome.Status = types.OutcomeIssuedCredentials
	default:
		outcome.Status = types.OutcomeIssueFailed
	}

	receiptPolicy := types.ReceiptPolicy(policyMeta)

	receipt, err := ledger.MakeReceipt(ledger.MakeReceiptInput{
		CreatedAt:  createdAt,
		IdemKey:    idemKey,
		ContextID:  ctxRecord.ContextID,
		DecisionID: decRecord.DecisionID,
		Actor: types.ReceiptActor{
			Kind:     "workload",
			Subject:  claims.Subject,
			Issuer:   claims.Issuer,
			Repo:     claims.Repo,
			Workflow: claims.Workflow,
			RunID:    claims.RunID,
			SHA:      claims.SHA,
		},
		Request: types.ReceiptRequest{
			RequestID: req.RequestID,
			Action:    req.Action,
			Resource:  req.Resource,
			Env:       req.Env,
			Intent:    req.Intent,
		},
		Policy:   receiptPolicy,
		Approval: approval,
		Outcome:  outcome,
	}, s.Signer)
	if err != nil {
		return AuthorizeResponse{}, err
	}

	rec := IdemRecord{
		IdemKey:    idemKey,
		Status:     status,
		ApprovalID: approvalID,
		ReceiptID:  receipt.ReceiptID,
		ContextID:  ctxRecord.ContextID,
		DecisionID: decRecord.DecisionID,
	}
	s.Store.Put(rec)

	resp := AuthorizeResponse{
		Verdict:    string(verdict),
		ContextID:  ctxRecord.ContextID,
		DecisionID: decRecord.DecisionID,
		ReceiptID:  receipt.ReceiptID,
	}
	if approvalID != "" {
		resp.Verdict = string(VerdictRequireApproval)
		resp.Approval = &struct {
			ApprovalID string `json:"approval_id"`
			Status     string `json:"status"`
		}{ApprovalID: approvalID, Status: string(ApprovalPending)}
	}

	if action == ActionReturnDenied {
		resp.Verdict = string(VerdictDeny)
	}
	if action == ActionIssueCredentials {
		resp.Verdict = string(VerdictAllow)
	}

	return resp, nil
}

type devSigner struct {
	keyID string
	priv  ed25519.PrivateKey
}

func (s devSigner) KeyID() string {
	return s.keyID
}

func (s devSigner) SignEd25519(message []byte) ([]byte, error) {
	return ed25519.Sign(s.priv, message), nil
}

func newApprovalID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("approval-%d", time.Now().UnixNano())
	}
	return "approval-" + hex.EncodeToString(buf)
}
