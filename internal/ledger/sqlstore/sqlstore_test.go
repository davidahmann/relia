package sqlstore

import (
	"errors"
	"fmt"
	"testing"

	"github.com/davidahmann/relia/internal/ledger"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	s, err := OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := ledger.Migrate(s.DB(), ledger.DBSQLite); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return s
}

func TestStoreCRUD(t *testing.T) {
	s := openTestStore(t)

	key := ledger.KeyRecord{KeyID: "kid", PublicKey: []byte("pub"), CreatedAt: "2025-12-20T00:00:00Z"}
	if err := s.PutKey(key); err != nil {
		t.Fatalf("put key: %v", err)
	}
	if got, ok := s.GetKey("kid"); !ok || got.KeyID != "kid" {
		t.Fatalf("get key mismatch: ok=%v got=%+v", ok, got)
	}

	policy := ledger.PolicyVersionRecord{
		PolicyHash:    "ph",
		PolicyID:      "pid",
		PolicyVersion: "1",
		PolicyYAML:    "policy_id: pid\npolicy_version: \"1\"\n",
		CreatedAt:     "2025-12-20T00:00:00Z",
	}
	if err := s.PutPolicyVersion(policy); err != nil {
		t.Fatalf("put policy: %v", err)
	}
	if got, ok := s.GetPolicyVersion("ph"); !ok || got.PolicyID != "pid" {
		t.Fatalf("get policy mismatch: ok=%v got=%+v", ok, got)
	}

	ctx := ledger.ContextRecord{ContextID: "ctx1", BodyJSON: []byte(`{"context_id":"ctx1"}`), CreatedAt: "2025-12-20T00:00:01Z"}
	if err := s.PutContext(ctx); err != nil {
		t.Fatalf("put context: %v", err)
	}
	if got, ok := s.GetContext("ctx1"); !ok || string(got.BodyJSON) != string(ctx.BodyJSON) {
		t.Fatalf("get context mismatch: ok=%v got=%+v", ok, got)
	}

	dec := ledger.DecisionRecord{DecisionID: "dec1", ContextID: "ctx1", PolicyHash: "ph", Verdict: "allow", BodyJSON: []byte(`{"decision_id":"dec1"}`), CreatedAt: "2025-12-20T00:00:02Z"}
	if err := s.PutDecision(dec); err != nil {
		t.Fatalf("put decision: %v", err)
	}
	if got, ok := s.GetDecision("dec1"); !ok || got.ContextID != "ctx1" {
		t.Fatalf("get decision mismatch: ok=%v got=%+v", ok, got)
	}

	idem := ledger.IdempotencyKey{
		IdemKey:   "idem1",
		Status:    "pending_approval",
		CreatedAt: "2025-12-20T00:00:03Z",
		UpdatedAt: "2025-12-20T00:00:03Z",
	}
	if err := s.PutIdempotencyKey(idem); err != nil {
		t.Fatalf("put idem: %v", err)
	}

	approval := ledger.ApprovalRecord{
		ApprovalID: "a1",
		IdemKey:    "idem1",
		Status:     "pending",
		CreatedAt:  "2025-12-20T00:00:04Z",
		UpdatedAt:  "2025-12-20T00:00:04Z",
	}
	if err := s.PutApproval(approval); err != nil {
		t.Fatalf("put approval: %v", err)
	}
	if got, ok := s.GetApproval("a1"); !ok || got.IdemKey != "idem1" {
		t.Fatalf("get approval mismatch: ok=%v got=%+v", ok, got)
	}
	if got, ok := s.GetApprovalByIdemKey("idem1"); !ok || got.ApprovalID != "a1" {
		t.Fatalf("get approval by idem mismatch: ok=%v got=%+v", ok, got)
	}

	outbox := ledger.SlackOutboxRecord{
		NotificationID: "slack:a1",
		ApprovalID:     "a1",
		Channel:        "C123",
		MessageJSON:    []byte(`{"approval_id":"a1"}`),
		Status:         "pending",
		AttemptCount:   0,
		NextAttemptAt:  "2025-12-20T00:00:04Z",
		CreatedAt:      "2025-12-20T00:00:04Z",
		UpdatedAt:      "2025-12-20T00:00:04Z",
	}
	if err := s.PutSlackOutbox(outbox); err != nil {
		t.Fatalf("put outbox: %v", err)
	}
	if got, ok := s.GetSlackOutbox("slack:a1"); !ok || got.ApprovalID != "a1" {
		t.Fatalf("get outbox mismatch: ok=%v got=%+v", ok, got)
	}
	if due, err := s.ListSlackOutboxDue("2025-12-21T00:00:00Z", 10); err != nil || len(due) != 1 {
		t.Fatalf("list due mismatch: err=%v len=%d", err, len(due))
	}

	receipt := ledger.ReceiptRecord{
		ReceiptID:     "r1",
		IdemKey:       "idem1",
		CreatedAt:     "2025-12-20T00:00:03Z",
		ContextID:     "ctx1",
		DecisionID:    "dec1",
		PolicyHash:    "ph",
		ApprovalID:    &approval.ApprovalID,
		OutcomeStatus: "approval_pending",
		Final:         true,
		BodyJSON:      []byte(`{"receipt_id":"r1"}`),
		BodyDigest:    "digest",
		KeyID:         "kid",
		Sig:           []byte("sig"),
	}
	if err := s.PutReceipt(receipt); err != nil {
		t.Fatalf("put receipt: %v", err)
	}
	if got, ok := s.GetReceipt("r1"); !ok || got.BodyDigest != "digest" || !got.Final {
		t.Fatalf("get receipt mismatch: ok=%v got=%+v", ok, got)
	}

	channel := "C123"
	ts := "1700000000.1234"
	approval.SlackChannel = &channel
	approval.SlackMsgTS = &ts
	approval.UpdatedAt = "2025-12-20T00:00:05Z"
	if err := s.PutApproval(approval); err != nil {
		t.Fatalf("put approval update: %v", err)
	}
	if got, ok := s.GetApproval("a1"); !ok || got.SlackChannel == nil || *got.SlackMsgTS != ts {
		t.Fatalf("approval update mismatch: ok=%v got=%+v", ok, got)
	}

	idem.Status = "allowed"
	idem.ApprovalID = &approval.ApprovalID
	idem.LatestReceiptID = &receipt.ReceiptID
	idem.FinalReceiptID = &receipt.ReceiptID
	idem.UpdatedAt = "2025-12-20T00:00:06Z"
	if err := s.PutIdempotencyKey(idem); err != nil {
		t.Fatalf("put idem: %v", err)
	}
	if got, ok := s.GetIdempotencyKey("idem1"); !ok || got.Status != "allowed" {
		t.Fatalf("get idem mismatch: ok=%v got=%+v", ok, got)
	}
}

func TestWithTxRollback(t *testing.T) {
	s := openTestStore(t)

	err := s.WithTx(func(tx ledger.Tx) error {
		if err := tx.PutContext(ledger.ContextRecord{ContextID: "ctx-rollback", BodyJSON: []byte(`{}`), CreatedAt: "now"}); err != nil {
			return err
		}
		return errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, ok := s.GetContext("ctx-rollback"); ok {
		t.Fatalf("expected rollback to discard context")
	}
}

func TestApplySchema(t *testing.T) {
	s, err := OpenSQLite("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.ApplySchema(`CREATE TABLE IF NOT EXISTS tmp_schema_test (id INTEGER);`); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := s.DB().Exec(`INSERT INTO tmp_schema_test(id) VALUES (1);`); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

func TestTxGetters(t *testing.T) {
	s := openTestStore(t)

	err := s.WithTx(func(tx ledger.Tx) error {
		key := ledger.KeyRecord{KeyID: "kid-tx", PublicKey: []byte("pub"), CreatedAt: "now"}
		if err := tx.PutKey(key); err != nil {
			return err
		}
		if _, ok := tx.GetKey("kid-tx"); !ok {
			t.Fatalf("expected key")
		}

		policy := ledger.PolicyVersionRecord{PolicyHash: "ph-tx", PolicyID: "pid", PolicyVersion: "1", PolicyYAML: "y", CreatedAt: "now"}
		if err := tx.PutPolicyVersion(policy); err != nil {
			return err
		}
		if _, ok := tx.GetPolicyVersion("ph-tx"); !ok {
			t.Fatalf("expected policy")
		}

		ctx := ledger.ContextRecord{ContextID: "ctx-tx", BodyJSON: []byte(`{}`), CreatedAt: "now"}
		if err := tx.PutContext(ctx); err != nil {
			return err
		}
		if _, ok := tx.GetContext("ctx-tx"); !ok {
			t.Fatalf("expected context")
		}

		dec := ledger.DecisionRecord{DecisionID: "dec-tx", ContextID: "ctx-tx", PolicyHash: "ph-tx", Verdict: "allow", BodyJSON: []byte(`{}`), CreatedAt: "now"}
		if err := tx.PutDecision(dec); err != nil {
			return err
		}
		if _, ok := tx.GetDecision("dec-tx"); !ok {
			t.Fatalf("expected decision")
		}

		idem := ledger.IdempotencyKey{IdemKey: "idem-tx", Status: "pending_approval", CreatedAt: "now", UpdatedAt: "now"}
		if err := tx.PutIdempotencyKey(idem); err != nil {
			return err
		}
		if _, ok := tx.GetIdempotencyKey("idem-tx"); !ok {
			t.Fatalf("expected idem")
		}

		approval := ledger.ApprovalRecord{ApprovalID: "a-tx", IdemKey: "idem-tx", Status: "pending", CreatedAt: "now", UpdatedAt: "now"}
		if err := tx.PutApproval(approval); err != nil {
			return err
		}
		if _, ok := tx.GetApproval("a-tx"); !ok {
			t.Fatalf("expected approval")
		}
		if _, ok := tx.GetApprovalByIdemKey("idem-tx"); !ok {
			t.Fatalf("expected approval by idem")
		}

		outbox := ledger.SlackOutboxRecord{
			NotificationID: "slack:a-tx",
			ApprovalID:     "a-tx",
			Channel:        "C1",
			MessageJSON:    []byte(`{"approval_id":"a-tx"}`),
			Status:         "pending",
			AttemptCount:   0,
			NextAttemptAt:  "now",
			CreatedAt:      "now",
			UpdatedAt:      "now",
		}
		if err := tx.PutSlackOutbox(outbox); err != nil {
			return err
		}
		if _, ok := tx.GetSlackOutbox("slack:a-tx"); !ok {
			t.Fatalf("expected outbox")
		}

		receipt := ledger.ReceiptRecord{
			ReceiptID:     "r-tx",
			IdemKey:       "idem-tx",
			CreatedAt:     "now",
			ContextID:     "ctx-tx",
			DecisionID:    "dec-tx",
			PolicyHash:    "ph-tx",
			ApprovalID:    &approval.ApprovalID,
			OutcomeStatus: "approval_pending",
			Final:         true,
			BodyJSON:      []byte(`{}`),
			BodyDigest:    "digest",
			KeyID:         "kid-tx",
			Sig:           []byte("sig"),
		}
		if err := tx.PutReceipt(receipt); err != nil {
			return err
		}
		if _, ok := tx.GetReceipt("r-tx"); !ok {
			t.Fatalf("expected receipt")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("withtx: %v", err)
	}
}
