package pgstore

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/davidahmann/relia/internal/ledger"
)

func TestWithTxCommitAndRollback(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_keys").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := s.WithTx(func(tx ledger.Tx) error {
		return tx.PutKey(ledger.KeyRecord{KeyID: "kid", PublicKey: []byte("pub"), CreatedAt: "2025-12-20T00:00:00Z"})
	}); err != nil {
		t.Fatalf("withtx: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectRollback()
	if err := s.WithTx(func(tx ledger.Tx) error {
		return errors.New("boom")
	}); err == nil {
		t.Fatalf("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestOpenPostgresReturnsErrorForBadDSN(t *testing.T) {
	_, err := OpenPostgres("postgres://user:pass@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDBAndClose(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	s := New(db)
	if s.DB() != db {
		t.Fatalf("expected same db pointer")
	}
	mock.ExpectClose()
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestKeyCRUD(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	s := New(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_keys").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutKey(ledger.KeyRecord{KeyID: "kid", PublicKey: []byte("pub"), CreatedAt: "2025-12-20T00:00:00Z"}); err != nil {
		t.Fatalf("put key: %v", err)
	}

	rows := sqlmock.NewRows([]string{"key_id", "public_key", "created_at", "rotated_at"}).
		AddRow("kid", []byte("pub"), "2025-12-20T00:00:00Z", nil)
	mock.ExpectQuery("SELECT key_id, public_key").WithArgs("kid").WillReturnRows(rows)
	if got, ok := s.GetKey("kid"); !ok || got.KeyID != "kid" {
		t.Fatalf("get key mismatch: ok=%v got=%+v", ok, got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestSlackOutboxCRUDAndList(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	s := New(db)

	// Invalid JSON should rollback.
	mock.ExpectBegin()
	mock.ExpectRollback()
	if err := s.PutSlackOutbox(ledger.SlackOutboxRecord{NotificationID: "n1", ApprovalID: "a1", Channel: "C1", MessageJSON: []byte("bad"), Status: "pending", NextAttemptAt: "now", CreatedAt: "now", UpdatedAt: "now"}); err == nil {
		t.Fatalf("expected error")
	}

	// Successful upsert.
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_slack_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutSlackOutbox(ledger.SlackOutboxRecord{NotificationID: "n1", ApprovalID: "a1", Channel: "C1", MessageJSON: []byte(`{"approval_id":"a1"}`), Status: "pending", NextAttemptAt: "2025-12-20T00:00:00Z", CreatedAt: "2025-12-20T00:00:00Z", UpdatedAt: "2025-12-20T00:00:00Z"}); err != nil {
		t.Fatalf("put outbox: %v", err)
	}

	rows := sqlmock.NewRows([]string{
		"notification_id", "approval_id", "channel", "message_json", "status", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at",
	}).AddRow(
		"n1", "a1", "C1", `{"approval_id":"a1"}`, "pending", 0, "2025-12-20T00:00:00Z", nil, nil, "2025-12-20T00:00:00Z", "2025-12-20T00:00:00Z",
	)
	mock.ExpectQuery("FROM relia_slack_outbox WHERE notification_id").WithArgs("n1").WillReturnRows(rows)
	if got, ok := s.GetSlackOutbox("n1"); !ok || got.ApprovalID != "a1" {
		t.Fatalf("get outbox mismatch: ok=%v got=%+v", ok, got)
	}

	listRows := sqlmock.NewRows([]string{
		"notification_id", "approval_id", "channel", "message_json", "status", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at",
	}).AddRow(
		"n1", "a1", "C1", `{"approval_id":"a1"}`, "pending", 0, "2025-12-20T00:00:00Z", nil, nil, "2025-12-20T00:00:00Z", "2025-12-20T00:00:00Z",
	)
	mock.ExpectQuery("FROM relia_slack_outbox").WithArgs("2025-12-21T00:00:00Z", 10).WillReturnRows(listRows)
	due, err := s.ListSlackOutboxDue("2025-12-21T00:00:00Z", 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("list due: err=%v len=%d", err, len(due))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestStoreCRUDAll(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	s := New(db)

	// PutKey
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_keys").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutKey(ledger.KeyRecord{KeyID: "kid", PublicKey: []byte("pub"), CreatedAt: "2025-12-20T00:00:00Z"}); err != nil {
		t.Fatalf("put key: %v", err)
	}

	// PutPolicyVersion
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_policy_versions").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutPolicyVersion(ledger.PolicyVersionRecord{PolicyHash: "ph", PolicyID: "pid", PolicyVersion: "1", PolicyYAML: "y", CreatedAt: "2025-12-20T00:00:00Z"}); err != nil {
		t.Fatalf("put policy: %v", err)
	}

	// PutContext
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_contexts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutContext(ledger.ContextRecord{ContextID: "ctx", BodyJSON: []byte(`{"context_id":"ctx"}`), CreatedAt: "2025-12-20T00:00:01Z"}); err != nil {
		t.Fatalf("put ctx: %v", err)
	}

	// PutDecision
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_decisions").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutDecision(ledger.DecisionRecord{DecisionID: "dec", ContextID: "ctx", PolicyHash: "ph", Verdict: "allow", BodyJSON: []byte(`{"decision_id":"dec"}`), CreatedAt: "2025-12-20T00:00:02Z"}); err != nil {
		t.Fatalf("put dec: %v", err)
	}

	// PutIdempotencyKey (no approval yet)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_idempotency_keys").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutIdempotencyKey(ledger.IdempotencyKey{IdemKey: "idem", Status: "pending_approval", CreatedAt: "2025-12-20T00:00:03Z", UpdatedAt: "2025-12-20T00:00:03Z"}); err != nil {
		t.Fatalf("put idem: %v", err)
	}

	// PutApproval
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_approvals").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutApproval(ledger.ApprovalRecord{ApprovalID: "a1", IdemKey: "idem", Status: "pending", CreatedAt: "2025-12-20T00:00:04Z", UpdatedAt: "2025-12-20T00:00:04Z"}); err != nil {
		t.Fatalf("put approval: %v", err)
	}

	// Update idempotency to link approval
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_idempotency_keys").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	approvalID := "a1"
	if err := s.PutIdempotencyKey(ledger.IdempotencyKey{IdemKey: "idem", Status: "pending_approval", ApprovalID: &approvalID, CreatedAt: "2025-12-20T00:00:03Z", UpdatedAt: "2025-12-20T00:00:05Z"}); err != nil {
		t.Fatalf("put idem2: %v", err)
	}

	// PutReceipt
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_receipts").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutReceipt(ledger.ReceiptRecord{
		ReceiptID:     "r1",
		IdemKey:       "idem",
		CreatedAt:     "2025-12-20T00:00:06Z",
		ContextID:     "ctx",
		DecisionID:    "dec",
		PolicyHash:    "ph",
		ApprovalID:    &approvalID,
		OutcomeStatus: "approval_pending",
		Final:         true,
		BodyJSON:      []byte(`{"receipt_id":"r1"}`),
		BodyDigest:    "digest",
		KeyID:         "kid",
		Sig:           []byte("sig"),
	}); err != nil {
		t.Fatalf("put receipt: %v", err)
	}

	// PutSlackOutbox
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO relia_slack_outbox").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := s.PutSlackOutbox(ledger.SlackOutboxRecord{
		NotificationID: "n1",
		ApprovalID:     "a1",
		Channel:        "C1",
		MessageJSON:    []byte(`{"approval_id":"a1"}`),
		Status:         "pending",
		AttemptCount:   0,
		NextAttemptAt:  "2025-12-20T00:00:04Z",
		CreatedAt:      "2025-12-20T00:00:04Z",
		UpdatedAt:      "2025-12-20T00:00:04Z",
	}); err != nil {
		t.Fatalf("put outbox: %v", err)
	}

	// Get methods
	mock.ExpectQuery("FROM relia_keys").WithArgs("kid").WillReturnRows(sqlmock.NewRows([]string{"key_id", "public_key", "created_at", "rotated_at"}).AddRow("kid", []byte("pub"), "2025-12-20T00:00:00Z", nil))
	if _, ok := s.GetKey("kid"); !ok {
		t.Fatalf("expected key")
	}
	mock.ExpectQuery("FROM relia_policy_versions").WithArgs("ph").WillReturnRows(sqlmock.NewRows([]string{"policy_hash", "policy_id", "policy_version", "policy_yaml", "created_at"}).AddRow("ph", "pid", "1", "y", "2025-12-20T00:00:00Z"))
	if _, ok := s.GetPolicyVersion("ph"); !ok {
		t.Fatalf("expected policy")
	}
	mock.ExpectQuery("FROM relia_contexts").WithArgs("ctx").WillReturnRows(sqlmock.NewRows([]string{"context_id", "body_json", "created_at"}).AddRow("ctx", `{"context_id":"ctx"}`, "2025-12-20T00:00:01Z"))
	if _, ok := s.GetContext("ctx"); !ok {
		t.Fatalf("expected context")
	}
	mock.ExpectQuery("FROM relia_decisions").WithArgs("dec").WillReturnRows(sqlmock.NewRows([]string{"decision_id", "created_at", "context_id", "policy_hash", "verdict", "body_json"}).AddRow("dec", "2025-12-20T00:00:02Z", "ctx", "ph", "allow", `{"decision_id":"dec"}`))
	if _, ok := s.GetDecision("dec"); !ok {
		t.Fatalf("expected decision")
	}
	mock.ExpectQuery("FROM relia_idempotency_keys").WithArgs("idem").WillReturnRows(sqlmock.NewRows([]string{"idem_key", "status", "approval_id", "latest_receipt_id", "final_receipt_id", "created_at", "updated_at", "ttl_expires_at"}).AddRow("idem", "pending_approval", "a1", nil, nil, "2025-12-20T00:00:03Z", "2025-12-20T00:00:05Z", nil))
	if _, ok := s.GetIdempotencyKey("idem"); !ok {
		t.Fatalf("expected idem")
	}
	mock.ExpectQuery("FROM relia_approvals WHERE approval_id").WithArgs("a1").WillReturnRows(sqlmock.NewRows([]string{"approval_id", "idem_key", "status", "slack_channel", "slack_msg_ts", "approved_by", "approved_at", "created_at", "updated_at"}).AddRow("a1", "idem", "pending", nil, nil, nil, nil, "2025-12-20T00:00:04Z", "2025-12-20T00:00:04Z"))
	if _, ok := s.GetApproval("a1"); !ok {
		t.Fatalf("expected approval")
	}
	mock.ExpectQuery("FROM relia_approvals WHERE idem_key").WithArgs("idem").WillReturnRows(sqlmock.NewRows([]string{"approval_id", "idem_key", "status", "slack_channel", "slack_msg_ts", "approved_by", "approved_at", "created_at", "updated_at"}).AddRow("a1", "idem", "pending", nil, nil, nil, nil, "2025-12-20T00:00:04Z", "2025-12-20T00:00:04Z"))
	if _, ok := s.GetApprovalByIdemKey("idem"); !ok {
		t.Fatalf("expected approval by idem")
	}
	mock.ExpectQuery("FROM relia_receipts").WithArgs("r1").WillReturnRows(sqlmock.NewRows([]string{"receipt_id", "idem_key", "created_at", "supersedes_receipt_id", "context_id", "decision_id", "policy_hash", "approval_id", "outcome_status", "final", "expires_at", "body_json", "body_digest", "key_id", "sig"}).AddRow("r1", "idem", "2025-12-20T00:00:06Z", nil, "ctx", "dec", "ph", "a1", "approval_pending", true, nil, `{"receipt_id":"r1"}`, "digest", "kid", []byte("sig")))
	if _, ok := s.GetReceipt("r1"); !ok {
		t.Fatalf("expected receipt")
	}
	mock.ExpectQuery("FROM relia_slack_outbox WHERE notification_id").WithArgs("n1").WillReturnRows(sqlmock.NewRows([]string{"notification_id", "approval_id", "channel", "message_json", "status", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at"}).AddRow("n1", "a1", "C1", `{"approval_id":"a1"}`, "pending", 0, "2025-12-20T00:00:04Z", nil, nil, "2025-12-20T00:00:04Z", "2025-12-20T00:00:04Z"))
	if _, ok := s.GetSlackOutbox("n1"); !ok {
		t.Fatalf("expected outbox")
	}
	mock.ExpectQuery("FROM relia_slack_outbox").WithArgs("2025-12-21T00:00:00Z", 10).WillReturnRows(sqlmock.NewRows([]string{"notification_id", "approval_id", "channel", "message_json", "status", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at"}).AddRow("n1", "a1", "C1", `{"approval_id":"a1"}`, "pending", 0, "2025-12-20T00:00:04Z", nil, nil, "2025-12-20T00:00:04Z", "2025-12-20T00:00:04Z"))
	due, err := s.ListSlackOutboxDue("2025-12-21T00:00:00Z", 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("list due: err=%v len=%d", err, len(due))
	}

	// Tx getters (exercise the Tx implementations too).
	mock.ExpectBegin()
	mock.ExpectQuery("FROM relia_keys").WithArgs("kid").WillReturnRows(sqlmock.NewRows([]string{"key_id", "public_key", "created_at", "rotated_at"}).AddRow("kid", []byte("pub"), "2025-12-20T00:00:00Z", nil))
	mock.ExpectQuery("FROM relia_policy_versions").WithArgs("ph").WillReturnRows(sqlmock.NewRows([]string{"policy_hash", "policy_id", "policy_version", "policy_yaml", "created_at"}).AddRow("ph", "pid", "1", "y", "2025-12-20T00:00:00Z"))
	mock.ExpectQuery("FROM relia_contexts").WithArgs("ctx").WillReturnRows(sqlmock.NewRows([]string{"context_id", "body_json", "created_at"}).AddRow("ctx", `{"context_id":"ctx"}`, "2025-12-20T00:00:01Z"))
	mock.ExpectQuery("FROM relia_decisions").WithArgs("dec").WillReturnRows(sqlmock.NewRows([]string{"decision_id", "created_at", "context_id", "policy_hash", "verdict", "body_json"}).AddRow("dec", "2025-12-20T00:00:02Z", "ctx", "ph", "allow", `{"decision_id":"dec"}`))
	mock.ExpectQuery("FROM relia_idempotency_keys").WithArgs("idem").WillReturnRows(sqlmock.NewRows([]string{"idem_key", "status", "approval_id", "latest_receipt_id", "final_receipt_id", "created_at", "updated_at", "ttl_expires_at"}).AddRow("idem", "pending_approval", "a1", nil, nil, "2025-12-20T00:00:03Z", "2025-12-20T00:00:05Z", nil))
	mock.ExpectQuery("FROM relia_approvals WHERE approval_id").WithArgs("a1").WillReturnRows(sqlmock.NewRows([]string{"approval_id", "idem_key", "status", "slack_channel", "slack_msg_ts", "approved_by", "approved_at", "created_at", "updated_at"}).AddRow("a1", "idem", "pending", nil, nil, nil, nil, "2025-12-20T00:00:04Z", "2025-12-20T00:00:04Z"))
	mock.ExpectQuery("FROM relia_approvals WHERE idem_key").WithArgs("idem").WillReturnRows(sqlmock.NewRows([]string{"approval_id", "idem_key", "status", "slack_channel", "slack_msg_ts", "approved_by", "approved_at", "created_at", "updated_at"}).AddRow("a1", "idem", "pending", nil, nil, nil, nil, "2025-12-20T00:00:04Z", "2025-12-20T00:00:04Z"))
	mock.ExpectQuery("FROM relia_receipts").WithArgs("r1").WillReturnRows(sqlmock.NewRows([]string{"receipt_id", "idem_key", "created_at", "supersedes_receipt_id", "context_id", "decision_id", "policy_hash", "approval_id", "outcome_status", "final", "expires_at", "body_json", "body_digest", "key_id", "sig"}).AddRow("r1", "idem", "2025-12-20T00:00:06Z", nil, "ctx", "dec", "ph", "a1", "approval_pending", true, nil, `{"receipt_id":"r1"}`, "digest", "kid", []byte("sig")))
	mock.ExpectQuery("FROM relia_slack_outbox WHERE notification_id").WithArgs("n1").WillReturnRows(sqlmock.NewRows([]string{"notification_id", "approval_id", "channel", "message_json", "status", "attempt_count", "next_attempt_at", "last_error", "sent_at", "created_at", "updated_at"}).AddRow("n1", "a1", "C1", `{"approval_id":"a1"}`, "pending", 0, "2025-12-20T00:00:04Z", nil, nil, "2025-12-20T00:00:04Z", "2025-12-20T00:00:04Z"))
	mock.ExpectCommit()
	if err := s.WithTx(func(tx ledger.Tx) error {
		if _, ok := tx.GetKey("kid"); !ok {
			t.Fatalf("expected tx key")
		}
		if _, ok := tx.GetPolicyVersion("ph"); !ok {
			t.Fatalf("expected tx policy")
		}
		if _, ok := tx.GetContext("ctx"); !ok {
			t.Fatalf("expected tx context")
		}
		if _, ok := tx.GetDecision("dec"); !ok {
			t.Fatalf("expected tx decision")
		}
		if _, ok := tx.GetIdempotencyKey("idem"); !ok {
			t.Fatalf("expected tx idem")
		}
		if _, ok := tx.GetApproval("a1"); !ok {
			t.Fatalf("expected tx approval")
		}
		if _, ok := tx.GetApprovalByIdemKey("idem"); !ok {
			t.Fatalf("expected tx approval by idem")
		}
		if _, ok := tx.GetReceipt("r1"); !ok {
			t.Fatalf("expected tx receipt")
		}
		if _, ok := tx.GetSlackOutbox("n1"); !ok {
			t.Fatalf("expected tx outbox")
		}
		return nil
	}); err != nil {
		t.Fatalf("withtx getters: %v", err)
	}

	// Invalid JSON paths (should rollback).
	mock.ExpectBegin()
	mock.ExpectRollback()
	if err := s.PutContext(ledger.ContextRecord{ContextID: "bad", BodyJSON: []byte("nope"), CreatedAt: "now"}); err == nil {
		t.Fatalf("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
