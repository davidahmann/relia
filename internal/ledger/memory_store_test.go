package ledger

import (
	"errors"
	"testing"
)

func TestInMemoryStore_CRUD(t *testing.T) {
	s := NewInMemoryStore()

	key := KeyRecord{KeyID: "kid", PublicKey: []byte("pub"), CreatedAt: "now"}
	if err := s.PutKey(key); err != nil {
		t.Fatalf("put key: %v", err)
	}
	if got, ok := s.GetKey("kid"); !ok || got.KeyID != "kid" {
		t.Fatalf("get key mismatch: ok=%v got=%+v", ok, got)
	}

	outbox := SlackOutboxRecord{
		NotificationID: "n1",
		ApprovalID:     "a1",
		Channel:        "C1",
		MessageJSON:    []byte(`{"approval_id":"a1"}`),
		Status:         "pending",
		AttemptCount:   0,
		NextAttemptAt:  "now",
		CreatedAt:      "now",
		UpdatedAt:      "now",
	}
	if err := s.PutSlackOutbox(outbox); err != nil {
		t.Fatalf("put outbox: %v", err)
	}
	if got, ok := s.GetSlackOutbox("n1"); !ok || got.ApprovalID != "a1" {
		t.Fatalf("get outbox mismatch: ok=%v got=%+v", ok, got)
	}
	if due, err := s.ListSlackOutboxDue("now", 10); err != nil || len(due) != 1 {
		t.Fatalf("list due mismatch: err=%v len=%d", err, len(due))
	}

	policy := PolicyVersionRecord{PolicyHash: "ph", PolicyID: "pid", PolicyVersion: "1", PolicyYAML: "y", CreatedAt: "now"}
	if err := s.PutPolicyVersion(policy); err != nil {
		t.Fatalf("put policy: %v", err)
	}
	if got, ok := s.GetPolicyVersion("ph"); !ok || got.PolicyID != "pid" {
		t.Fatalf("get policy mismatch: ok=%v got=%+v", ok, got)
	}

	ctx := ContextRecord{ContextID: "c1", BodyJSON: []byte(`{}`), CreatedAt: "now"}
	if err := s.PutContext(ctx); err != nil {
		t.Fatalf("put context: %v", err)
	}
	if got, ok := s.GetContext("c1"); !ok || string(got.BodyJSON) != "{}" {
		t.Fatalf("get context mismatch: ok=%v got=%+v", ok, got)
	}

	dec := DecisionRecord{DecisionID: "d1", ContextID: "c1", PolicyHash: "ph", Verdict: "allow", BodyJSON: []byte(`{}`), CreatedAt: "now"}
	if err := s.PutDecision(dec); err != nil {
		t.Fatalf("put decision: %v", err)
	}
	if got, ok := s.GetDecision("d1"); !ok || got.ContextID != "c1" {
		t.Fatalf("get decision mismatch: ok=%v got=%+v", ok, got)
	}

	rec := ReceiptRecord{ReceiptID: "r1", IdemKey: "i1", ContextID: "c1", DecisionID: "d1", PolicyHash: "ph", BodyJSON: []byte(`{}`), CreatedAt: "now"}
	if err := s.PutReceipt(rec); err != nil {
		t.Fatalf("put receipt: %v", err)
	}
	if got, ok := s.GetReceipt("r1"); !ok || got.IdemKey != "i1" {
		t.Fatalf("get receipt mismatch: ok=%v got=%+v", ok, got)
	}

	approval := ApprovalRecord{ApprovalID: "a1", IdemKey: "i1", Status: "pending", CreatedAt: "now", UpdatedAt: "now"}
	if err := s.PutApproval(approval); err != nil {
		t.Fatalf("put approval: %v", err)
	}
	if got, ok := s.GetApproval("a1"); !ok || got.Status != "pending" {
		t.Fatalf("get approval mismatch: ok=%v got=%+v", ok, got)
	}
	if got, ok := s.GetApprovalByIdemKey("i1"); !ok || got.ApprovalID != "a1" {
		t.Fatalf("get approval by idem mismatch: ok=%v got=%+v", ok, got)
	}

	idem := IdempotencyKey{IdemKey: "i1", Status: "pending", CreatedAt: "now", UpdatedAt: "now"}
	if err := s.PutIdempotencyKey(idem); err != nil {
		t.Fatalf("put idem: %v", err)
	}
	if got, ok := s.GetIdempotencyKey("i1"); !ok || got.Status != "pending" {
		t.Fatalf("get idem mismatch: ok=%v got=%+v", ok, got)
	}
}

func TestInMemoryStore_WithTx(t *testing.T) {
	s := NewInMemoryStore()
	err := s.WithTx(func(tx Tx) error {
		if err := tx.PutKey(KeyRecord{KeyID: "tx-k", PublicKey: []byte("pub"), CreatedAt: "now"}); err != nil {
			return err
		}
		if _, ok := tx.GetKey("tx-k"); !ok {
			t.Fatalf("expected key in tx")
		}
		if err := tx.PutSlackOutbox(SlackOutboxRecord{
			NotificationID: "tx-n1",
			ApprovalID:     "a1",
			Channel:        "C1",
			MessageJSON:    []byte(`{}`),
			Status:         "pending",
			AttemptCount:   0,
			NextAttemptAt:  "now",
			CreatedAt:      "now",
			UpdatedAt:      "now",
		}); err != nil {
			return err
		}
		if _, ok := tx.GetSlackOutbox("tx-n1"); !ok {
			t.Fatalf("expected outbox in tx")
		}
		if err := tx.PutPolicyVersion(PolicyVersionRecord{PolicyHash: "tx-ph", PolicyID: "pid", PolicyVersion: "1", PolicyYAML: "y", CreatedAt: "now"}); err != nil {
			return err
		}
		if _, ok := tx.GetPolicyVersion("tx-ph"); !ok {
			t.Fatalf("expected policy in tx")
		}
		if err := tx.PutContext(ContextRecord{ContextID: "tx", BodyJSON: []byte(`{}`), CreatedAt: "now"}); err != nil {
			return err
		}
		if _, ok := tx.GetContext("tx"); !ok {
			t.Fatalf("expected context in tx")
		}
		if err := tx.PutDecision(DecisionRecord{DecisionID: "d", ContextID: "tx", BodyJSON: []byte(`{}`), CreatedAt: "now"}); err != nil {
			return err
		}
		if _, ok := tx.GetDecision("d"); !ok {
			t.Fatalf("expected decision in tx")
		}
		if err := tx.PutReceipt(ReceiptRecord{ReceiptID: "r", IdemKey: "i", BodyJSON: []byte(`{}`), CreatedAt: "now"}); err != nil {
			return err
		}
		if _, ok := tx.GetReceipt("r"); !ok {
			t.Fatalf("expected receipt in tx")
		}
		if err := tx.PutApproval(ApprovalRecord{ApprovalID: "a", IdemKey: "i", Status: "pending", CreatedAt: "now", UpdatedAt: "now"}); err != nil {
			return err
		}
		if _, ok := tx.GetApproval("a"); !ok {
			t.Fatalf("expected approval in tx")
		}
		if _, ok := tx.GetApprovalByIdemKey("i"); !ok {
			t.Fatalf("expected approval by idem in tx")
		}
		if err := tx.PutIdempotencyKey(IdempotencyKey{IdemKey: "i", Status: "pending_approval", CreatedAt: "now", UpdatedAt: "now"}); err != nil {
			return err
		}
		if _, ok := tx.GetIdempotencyKey("i"); !ok {
			t.Fatalf("expected idem in tx")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withtx: %v", err)
	}
	if _, ok := s.GetContext("tx"); !ok {
		t.Fatalf("expected context")
	}

	err = s.WithTx(func(tx Tx) error {
		_ = tx.PutContext(ContextRecord{ContextID: "rollback", BodyJSON: []byte(`{}`), CreatedAt: "now"})
		return errors.New("boom")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	// In-memory "tx" is just a lock; it doesn't rollback.
	if _, ok := s.GetContext("rollback"); !ok {
		t.Fatalf("expected in-memory tx to keep writes")
	}
}
