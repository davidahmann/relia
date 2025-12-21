-- Durable outbound Slack notifications (retry/backoff).
CREATE TABLE IF NOT EXISTS slack_outbox (
  notification_id TEXT PRIMARY KEY,
  approval_id     TEXT NOT NULL UNIQUE,
  channel         TEXT NOT NULL,
  message_json    TEXT NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('pending','sent')),
  attempt_count   INTEGER NOT NULL,
  next_attempt_at TEXT NOT NULL,
  last_error      TEXT,
  sent_at         TEXT,
  created_at      TEXT NOT NULL,
  updated_at      TEXT NOT NULL,
  FOREIGN KEY(approval_id) REFERENCES approvals(approval_id)
);

CREATE INDEX IF NOT EXISTS idx_slack_outbox_due ON slack_outbox(status, next_attempt_at);

