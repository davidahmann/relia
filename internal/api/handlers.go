package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/davidahmann/relia/internal/auth"
	"github.com/davidahmann/relia/internal/ledger"
	"github.com/davidahmann/relia/internal/pack"
	"github.com/davidahmann/relia/internal/slack"
)

type Handler struct {
	Auth             auth.Authenticator
	AuthorizeService *AuthorizeService
	SlackHandler     *slack.InteractionHandler
}

func (h *Handler) Authorize(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAuth(w, r) {
		return
	}

	if h.AuthorizeService == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "authorize service not configured"})
		return
	}

	var req AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	claims, err := h.Authenticate(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	actor := ActorContext{
		Subject:  claims.Subject,
		Issuer:   claims.Issuer,
		Repo:     claims.Repo,
		Workflow: claims.Workflow,
		RunID:    claims.RunID,
		SHA:      claims.SHA,
	}

	resp, err := h.AuthorizeService.Authorize(actor, req, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) Approvals(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAuth(w, r) {
		return
	}
	if h.AuthorizeService == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "authorize service not configured"})
		return
	}

	approvalID := strings.TrimPrefix(r.URL.Path, "/v1/approvals/")
	if approvalID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing approval_id"})
		return
	}

	approval, ok := h.AuthorizeService.GetApproval(approvalID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "approval not found"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"approval_id": approval.ApprovalID,
		"status":      string(approval.Status),
		"receipt_id":  approval.ReceiptID,
	})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAuth(w, r) {
		return
	}
	if h.AuthorizeService == nil || h.AuthorizeService.Artifacts == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "verify not implemented"})
		return
	}

	receiptID := strings.TrimPrefix(r.URL.Path, "/v1/verify/")
	if receiptID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing receipt_id"})
		return
	}

	receipt, ok := h.AuthorizeService.Artifacts.GetReceipt(receiptID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "receipt not found"})
		return
	}

	if h.AuthorizeService.PublicKey == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "public key not configured"})
		return
	}

	err := ledger.VerifyReceipt(receipt, h.AuthorizeService.PublicKey)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"receipt_id": receiptID,
			"valid":      false,
			"error":      err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"receipt_id": receiptID,
		"valid":      true,
	})
}

func (h *Handler) Pack(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAuth(w, r) {
		return
	}
	if h.AuthorizeService == nil || h.AuthorizeService.Artifacts == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "pack not implemented"})
		return
	}

	receiptID := strings.TrimPrefix(r.URL.Path, "/v1/pack/")
	if receiptID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing receipt_id"})
		return
	}

	receipt, ok := h.AuthorizeService.Artifacts.GetReceipt(receiptID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "receipt not found"})
		return
	}

	ctx, ok := h.AuthorizeService.Artifacts.GetContext(receipt.ContextID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "context not found"})
		return
	}

	decision, ok := h.AuthorizeService.Artifacts.GetDecision(receipt.DecisionID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "decision not found"})
		return
	}

	policyBytes, ok := h.AuthorizeService.Artifacts.GetPolicy(receipt.PolicyHash)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "policy not found"})
		return
	}

	approvals := []pack.ApprovalRecord{}
	if receipt.ApprovalID != nil {
		if approval, ok := h.AuthorizeService.GetApproval(*receipt.ApprovalID); ok {
			approvals = append(approvals, pack.ApprovalRecord{
				ApprovalID: approval.ApprovalID,
				Status:     string(approval.Status),
				ReceiptID:  approval.ReceiptID,
			})
		}
	}

	baseURL := ""
	if r.Host != "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		baseURL = scheme + "://" + r.Host
	}

	zipBytes, err := pack.BuildZip(pack.Input{
		Receipt:   receipt,
		Context:   ctx,
		Decision:  decision,
		Policy:    policyBytes,
		Approvals: approvals,
	}, baseURL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=relia-pack.zip")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(zipBytes)
}

func (h *Handler) SlackInteractions(w http.ResponseWriter, r *http.Request) {
	if h.SlackHandler == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "slack handler not configured"})
		return
	}
	h.SlackHandler.HandleInteractions(w, r)
}

func (h *Handler) ensureAuth(w http.ResponseWriter, r *http.Request) bool {
	_, err := h.Authenticate(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return false
	}
	return true
}

func (h *Handler) Authenticate(r *http.Request) (auth.Claims, error) {
	return h.Auth.Authenticate(r)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
}
