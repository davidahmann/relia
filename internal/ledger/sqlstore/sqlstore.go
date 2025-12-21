package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/davidahmann/relia/internal/ledger"
)

type Store struct {
	db *sql.DB
}

func OpenSQLite(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, err
	}
	return New(db), nil
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) ApplySchema(schema string) error {
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) WithTx(fn func(ledger.Tx) error) error {
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{})
	if err != nil {
		return err
	}
	if _, err := tx.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		_ = tx.Rollback()
		return err
	}
	wrapped := &Tx{tx: tx}
	if err := fn(wrapped); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Store) PutKey(key ledger.KeyRecord) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutKey(key) })
}

func (s *Store) GetKey(keyID string) (ledger.KeyRecord, bool) {
	var rec ledger.KeyRecord
	row := s.db.QueryRow(`SELECT key_id, public_key, created_at, rotated_at FROM keys WHERE key_id = ?`, keyID)
	if err := row.Scan(&rec.KeyID, &rec.PublicKey, &rec.CreatedAt, &rec.RotatedAt); err != nil {
		return ledger.KeyRecord{}, false
	}
	return rec, true
}

func (s *Store) PutSlackOutbox(rec ledger.SlackOutboxRecord) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutSlackOutbox(rec) })
}

func (s *Store) GetSlackOutbox(notificationID string) (ledger.SlackOutboxRecord, bool) {
	var rec ledger.SlackOutboxRecord
	var msg string
	row := s.db.QueryRow(`SELECT notification_id, approval_id, channel, message_json, status, attempt_count, next_attempt_at, last_error, sent_at, created_at, updated_at
FROM slack_outbox WHERE notification_id = ?`, notificationID)
	if err := row.Scan(&rec.NotificationID, &rec.ApprovalID, &rec.Channel, &msg, &rec.Status, &rec.AttemptCount, &rec.NextAttemptAt, &rec.LastError, &rec.SentAt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return ledger.SlackOutboxRecord{}, false
	}
	rec.MessageJSON = []byte(msg)
	return rec, true
}

func (s *Store) ListSlackOutboxDue(now string, limit int) ([]ledger.SlackOutboxRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`SELECT notification_id, approval_id, channel, message_json, status, attempt_count, next_attempt_at, last_error, sent_at, created_at, updated_at
FROM slack_outbox
WHERE status = 'pending' AND next_attempt_at <= ?
ORDER BY created_at ASC
LIMIT ?`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ledger.SlackOutboxRecord{}
	for rows.Next() {
		var rec ledger.SlackOutboxRecord
		var msg string
		if err := rows.Scan(&rec.NotificationID, &rec.ApprovalID, &rec.Channel, &msg, &rec.Status, &rec.AttemptCount, &rec.NextAttemptAt, &rec.LastError, &rec.SentAt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		rec.MessageJSON = []byte(msg)
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *Store) PutPolicyVersion(policy ledger.PolicyVersionRecord) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutPolicyVersion(policy) })
}

func (s *Store) GetPolicyVersion(policyHash string) (ledger.PolicyVersionRecord, bool) {
	var rec ledger.PolicyVersionRecord
	row := s.db.QueryRow(`SELECT policy_hash, policy_id, policy_version, policy_yaml, created_at FROM policy_versions WHERE policy_hash = ?`, policyHash)
	if err := row.Scan(&rec.PolicyHash, &rec.PolicyID, &rec.PolicyVersion, &rec.PolicyYAML, &rec.CreatedAt); err != nil {
		return ledger.PolicyVersionRecord{}, false
	}
	return rec, true
}

func (s *Store) PutContext(ctx ledger.ContextRecord) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutContext(ctx) })
}

func (s *Store) GetContext(contextID string) (ledger.ContextRecord, bool) {
	var rec ledger.ContextRecord
	var body string
	row := s.db.QueryRow(`SELECT context_id, body_json, created_at FROM contexts WHERE context_id = ?`, contextID)
	if err := row.Scan(&rec.ContextID, &body, &rec.CreatedAt); err != nil {
		return ledger.ContextRecord{}, false
	}
	rec.BodyJSON = []byte(body)
	return rec, true
}

func (s *Store) PutDecision(decision ledger.DecisionRecord) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutDecision(decision) })
}

func (s *Store) GetDecision(decisionID string) (ledger.DecisionRecord, bool) {
	var rec ledger.DecisionRecord
	var body string
	row := s.db.QueryRow(`SELECT decision_id, created_at, context_id, policy_hash, verdict, body_json FROM decisions WHERE decision_id = ?`, decisionID)
	if err := row.Scan(&rec.DecisionID, &rec.CreatedAt, &rec.ContextID, &rec.PolicyHash, &rec.Verdict, &body); err != nil {
		return ledger.DecisionRecord{}, false
	}
	rec.BodyJSON = []byte(body)
	return rec, true
}

func (s *Store) PutReceipt(receipt ledger.ReceiptRecord) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutReceipt(receipt) })
}

func (s *Store) GetReceipt(receiptID string) (ledger.ReceiptRecord, bool) {
	var rec ledger.ReceiptRecord
	var finalInt int
	var body string
	row := s.db.QueryRow(`SELECT receipt_id, idem_key, created_at, supersedes_receipt_id, context_id, decision_id, policy_hash, approval_id, outcome_status, final, expires_at, body_json, body_digest, key_id, sig
FROM receipts WHERE receipt_id = ?`, receiptID)
	if err := row.Scan(
		&rec.ReceiptID,
		&rec.IdemKey,
		&rec.CreatedAt,
		&rec.SupersedesReceiptID,
		&rec.ContextID,
		&rec.DecisionID,
		&rec.PolicyHash,
		&rec.ApprovalID,
		&rec.OutcomeStatus,
		&finalInt,
		&rec.ExpiresAt,
		&body,
		&rec.BodyDigest,
		&rec.KeyID,
		&rec.Sig,
	); err != nil {
		return ledger.ReceiptRecord{}, false
	}
	rec.Final = finalInt != 0
	rec.BodyJSON = []byte(body)
	return rec, true
}

func (s *Store) PutApproval(approval ledger.ApprovalRecord) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutApproval(approval) })
}

func (s *Store) GetApproval(approvalID string) (ledger.ApprovalRecord, bool) {
	var rec ledger.ApprovalRecord
	row := s.db.QueryRow(`SELECT approval_id, idem_key, status, slack_channel, slack_msg_ts, approved_by, approved_at, created_at, updated_at FROM approvals WHERE approval_id = ?`, approvalID)
	if err := row.Scan(&rec.ApprovalID, &rec.IdemKey, &rec.Status, &rec.SlackChannel, &rec.SlackMsgTS, &rec.ApprovedBy, &rec.ApprovedAt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return ledger.ApprovalRecord{}, false
	}
	return rec, true
}

func (s *Store) GetApprovalByIdemKey(idemKey string) (ledger.ApprovalRecord, bool) {
	var rec ledger.ApprovalRecord
	row := s.db.QueryRow(`SELECT approval_id, idem_key, status, slack_channel, slack_msg_ts, approved_by, approved_at, created_at, updated_at FROM approvals WHERE idem_key = ?`, idemKey)
	if err := row.Scan(&rec.ApprovalID, &rec.IdemKey, &rec.Status, &rec.SlackChannel, &rec.SlackMsgTS, &rec.ApprovedBy, &rec.ApprovedAt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return ledger.ApprovalRecord{}, false
	}
	return rec, true
}

func (s *Store) PutIdempotencyKey(key ledger.IdempotencyKey) error {
	return s.WithTx(func(tx ledger.Tx) error { return tx.PutIdempotencyKey(key) })
}

func (s *Store) GetIdempotencyKey(idemKey string) (ledger.IdempotencyKey, bool) {
	var rec ledger.IdempotencyKey
	row := s.db.QueryRow(`SELECT idem_key, status, approval_id, latest_receipt_id, final_receipt_id, created_at, updated_at, ttl_expires_at FROM idempotency_keys WHERE idem_key = ?`, idemKey)
	if err := row.Scan(&rec.IdemKey, &rec.Status, &rec.ApprovalID, &rec.LatestReceiptID, &rec.FinalReceiptID, &rec.CreatedAt, &rec.UpdatedAt, &rec.TTLExpiresAt); err != nil {
		return ledger.IdempotencyKey{}, false
	}
	return rec, true
}

type Tx struct {
	tx *sql.Tx
}

func (t *Tx) PutKey(key ledger.KeyRecord) error {
	_, err := t.tx.Exec(
		`INSERT INTO keys(key_id, public_key, created_at, rotated_at)
VALUES(?,?,?,?)
ON CONFLICT(key_id) DO NOTHING`,
		key.KeyID,
		key.PublicKey,
		key.CreatedAt,
		key.RotatedAt,
	)
	return err
}

func (t *Tx) GetKey(keyID string) (ledger.KeyRecord, bool) {
	var rec ledger.KeyRecord
	row := t.tx.QueryRow(`SELECT key_id, public_key, created_at, rotated_at FROM keys WHERE key_id = ?`, keyID)
	if err := row.Scan(&rec.KeyID, &rec.PublicKey, &rec.CreatedAt, &rec.RotatedAt); err != nil {
		return ledger.KeyRecord{}, false
	}
	return rec, true
}

func (t *Tx) PutSlackOutbox(rec ledger.SlackOutboxRecord) error {
	_, err := t.tx.Exec(
		`INSERT INTO slack_outbox(notification_id, approval_id, channel, message_json, status, attempt_count, next_attempt_at, last_error, sent_at, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(notification_id) DO UPDATE SET
  status=excluded.status,
  attempt_count=excluded.attempt_count,
  next_attempt_at=excluded.next_attempt_at,
  last_error=excluded.last_error,
  sent_at=excluded.sent_at,
  updated_at=excluded.updated_at`,
		rec.NotificationID,
		rec.ApprovalID,
		rec.Channel,
		string(rec.MessageJSON),
		rec.Status,
		rec.AttemptCount,
		rec.NextAttemptAt,
		rec.LastError,
		rec.SentAt,
		rec.CreatedAt,
		rec.UpdatedAt,
	)
	return err
}

func (t *Tx) GetSlackOutbox(notificationID string) (ledger.SlackOutboxRecord, bool) {
	var rec ledger.SlackOutboxRecord
	var msg string
	row := t.tx.QueryRow(`SELECT notification_id, approval_id, channel, message_json, status, attempt_count, next_attempt_at, last_error, sent_at, created_at, updated_at
FROM slack_outbox WHERE notification_id = ?`, notificationID)
	if err := row.Scan(&rec.NotificationID, &rec.ApprovalID, &rec.Channel, &msg, &rec.Status, &rec.AttemptCount, &rec.NextAttemptAt, &rec.LastError, &rec.SentAt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return ledger.SlackOutboxRecord{}, false
	}
	rec.MessageJSON = []byte(msg)
	return rec, true
}

func (t *Tx) PutPolicyVersion(policy ledger.PolicyVersionRecord) error {
	_, err := t.tx.Exec(
		`INSERT INTO policy_versions(policy_hash, policy_id, policy_version, policy_yaml, created_at)
VALUES(?,?,?,?,?)
ON CONFLICT(policy_hash) DO NOTHING`,
		policy.PolicyHash, policy.PolicyID, policy.PolicyVersion, policy.PolicyYAML, policy.CreatedAt,
	)
	return err
}

func (t *Tx) GetPolicyVersion(policyHash string) (ledger.PolicyVersionRecord, bool) {
	var rec ledger.PolicyVersionRecord
	row := t.tx.QueryRow(`SELECT policy_hash, policy_id, policy_version, policy_yaml, created_at FROM policy_versions WHERE policy_hash = ?`, policyHash)
	if err := row.Scan(&rec.PolicyHash, &rec.PolicyID, &rec.PolicyVersion, &rec.PolicyYAML, &rec.CreatedAt); err != nil {
		return ledger.PolicyVersionRecord{}, false
	}
	return rec, true
}

func (t *Tx) PutContext(ctx ledger.ContextRecord) error {
	_, err := t.tx.Exec(`INSERT INTO contexts(context_id, created_at, body_json) VALUES(?,?,?) ON CONFLICT(context_id) DO NOTHING`, ctx.ContextID, ctx.CreatedAt, string(ctx.BodyJSON))
	return err
}

func (t *Tx) GetContext(contextID string) (ledger.ContextRecord, bool) {
	var rec ledger.ContextRecord
	var body string
	row := t.tx.QueryRow(`SELECT context_id, body_json, created_at FROM contexts WHERE context_id = ?`, contextID)
	if err := row.Scan(&rec.ContextID, &body, &rec.CreatedAt); err != nil {
		return ledger.ContextRecord{}, false
	}
	rec.BodyJSON = []byte(body)
	return rec, true
}

func (t *Tx) PutDecision(decision ledger.DecisionRecord) error {
	_, err := t.tx.Exec(`INSERT INTO decisions(decision_id, created_at, context_id, policy_hash, verdict, body_json) VALUES(?,?,?,?,?,?) ON CONFLICT(decision_id) DO NOTHING`,
		decision.DecisionID, decision.CreatedAt, decision.ContextID, decision.PolicyHash, decision.Verdict, string(decision.BodyJSON),
	)
	return err
}

func (t *Tx) GetDecision(decisionID string) (ledger.DecisionRecord, bool) {
	var rec ledger.DecisionRecord
	var body string
	row := t.tx.QueryRow(`SELECT decision_id, created_at, context_id, policy_hash, verdict, body_json FROM decisions WHERE decision_id = ?`, decisionID)
	if err := row.Scan(&rec.DecisionID, &rec.CreatedAt, &rec.ContextID, &rec.PolicyHash, &rec.Verdict, &body); err != nil {
		return ledger.DecisionRecord{}, false
	}
	rec.BodyJSON = []byte(body)
	return rec, true
}

func (t *Tx) PutReceipt(receipt ledger.ReceiptRecord) error {
	if receipt.ReceiptID == "" {
		return fmt.Errorf("missing receipt_id")
	}
	_, err := t.tx.Exec(`INSERT INTO receipts(receipt_id, idem_key, created_at, supersedes_receipt_id, context_id, decision_id, policy_hash, approval_id, outcome_status, final, expires_at, body_json, body_digest, key_id, sig)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(receipt_id) DO NOTHING`,
		receipt.ReceiptID,
		receipt.IdemKey,
		receipt.CreatedAt,
		receipt.SupersedesReceiptID,
		receipt.ContextID,
		receipt.DecisionID,
		receipt.PolicyHash,
		receipt.ApprovalID,
		receipt.OutcomeStatus,
		boolToInt(receipt.Final),
		receipt.ExpiresAt,
		string(receipt.BodyJSON),
		receipt.BodyDigest,
		receipt.KeyID,
		receipt.Sig,
	)
	return err
}

func (t *Tx) GetReceipt(receiptID string) (ledger.ReceiptRecord, bool) {
	var rec ledger.ReceiptRecord
	var finalInt int
	var body string
	row := t.tx.QueryRow(`SELECT receipt_id, idem_key, created_at, supersedes_receipt_id, context_id, decision_id, policy_hash, approval_id, outcome_status, final, expires_at, body_json, body_digest, key_id, sig FROM receipts WHERE receipt_id = ?`, receiptID)
	if err := row.Scan(&rec.ReceiptID, &rec.IdemKey, &rec.CreatedAt, &rec.SupersedesReceiptID, &rec.ContextID, &rec.DecisionID, &rec.PolicyHash, &rec.ApprovalID, &rec.OutcomeStatus, &finalInt, &rec.ExpiresAt, &body, &rec.BodyDigest, &rec.KeyID, &rec.Sig); err != nil {
		return ledger.ReceiptRecord{}, false
	}
	rec.Final = finalInt != 0
	rec.BodyJSON = []byte(body)
	return rec, true
}

func (t *Tx) PutApproval(approval ledger.ApprovalRecord) error {
	_, err := t.tx.Exec(`INSERT INTO approvals(approval_id, idem_key, status, slack_channel, slack_msg_ts, approved_by, approved_at, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(approval_id) DO UPDATE SET
  status=excluded.status,
  slack_channel=COALESCE(excluded.slack_channel, approvals.slack_channel),
  slack_msg_ts=COALESCE(excluded.slack_msg_ts, approvals.slack_msg_ts),
  approved_by=COALESCE(excluded.approved_by, approvals.approved_by),
  approved_at=COALESCE(excluded.approved_at, approvals.approved_at),
  updated_at=excluded.updated_at`,
		approval.ApprovalID,
		approval.IdemKey,
		approval.Status,
		approval.SlackChannel,
		approval.SlackMsgTS,
		approval.ApprovedBy,
		approval.ApprovedAt,
		approval.CreatedAt,
		approval.UpdatedAt,
	)
	return err
}

func (t *Tx) GetApproval(approvalID string) (ledger.ApprovalRecord, bool) {
	var rec ledger.ApprovalRecord
	row := t.tx.QueryRow(`SELECT approval_id, idem_key, status, slack_channel, slack_msg_ts, approved_by, approved_at, created_at, updated_at FROM approvals WHERE approval_id = ?`, approvalID)
	if err := row.Scan(&rec.ApprovalID, &rec.IdemKey, &rec.Status, &rec.SlackChannel, &rec.SlackMsgTS, &rec.ApprovedBy, &rec.ApprovedAt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return ledger.ApprovalRecord{}, false
	}
	return rec, true
}

func (t *Tx) GetApprovalByIdemKey(idemKey string) (ledger.ApprovalRecord, bool) {
	var rec ledger.ApprovalRecord
	row := t.tx.QueryRow(`SELECT approval_id, idem_key, status, slack_channel, slack_msg_ts, approved_by, approved_at, created_at, updated_at FROM approvals WHERE idem_key = ?`, idemKey)
	if err := row.Scan(&rec.ApprovalID, &rec.IdemKey, &rec.Status, &rec.SlackChannel, &rec.SlackMsgTS, &rec.ApprovedBy, &rec.ApprovedAt, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
		return ledger.ApprovalRecord{}, false
	}
	return rec, true
}

func (t *Tx) PutIdempotencyKey(key ledger.IdempotencyKey) error {
	_, err := t.tx.Exec(`INSERT INTO idempotency_keys(idem_key, status, approval_id, latest_receipt_id, final_receipt_id, created_at, updated_at, ttl_expires_at)
VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(idem_key) DO UPDATE SET
  status=excluded.status,
  approval_id=excluded.approval_id,
  latest_receipt_id=excluded.latest_receipt_id,
  final_receipt_id=excluded.final_receipt_id,
  updated_at=excluded.updated_at,
  ttl_expires_at=excluded.ttl_expires_at`,
		key.IdemKey,
		key.Status,
		key.ApprovalID,
		key.LatestReceiptID,
		key.FinalReceiptID,
		key.CreatedAt,
		key.UpdatedAt,
		key.TTLExpiresAt,
	)
	return err
}

func (t *Tx) GetIdempotencyKey(idemKey string) (ledger.IdempotencyKey, bool) {
	var rec ledger.IdempotencyKey
	row := t.tx.QueryRow(`SELECT idem_key, status, approval_id, latest_receipt_id, final_receipt_id, created_at, updated_at, ttl_expires_at FROM idempotency_keys WHERE idem_key = ?`, idemKey)
	if err := row.Scan(&rec.IdemKey, &rec.Status, &rec.ApprovalID, &rec.LatestReceiptID, &rec.FinalReceiptID, &rec.CreatedAt, &rec.UpdatedAt, &rec.TTLExpiresAt); err != nil {
		return ledger.IdempotencyKey{}, false
	}
	return rec, true
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
