-- Durable outbound Slack notifications (retry/backoff).
CREATE TABLE IF NOT EXISTS relia_slack_outbox (
  notification_id TEXT PRIMARY KEY,
  approval_id     TEXT NOT NULL UNIQUE REFERENCES relia_approvals(approval_id),
  channel         TEXT NOT NULL,
  message_json    JSONB NOT NULL,
  status          TEXT NOT NULL CHECK (status IN ('pending','sent')),
  attempt_count   INTEGER NOT NULL,
  next_attempt_at TIMESTAMPTZ NOT NULL,
  last_error      TEXT,
  sent_at         TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL,
  updated_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rel_slack_outbox_due ON relia_slack_outbox(status, next_attempt_at);

