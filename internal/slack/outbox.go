package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/davidahmann/relia/internal/ledger"
)

type OutboxPoster interface {
	PostApproval(channel string, message ApprovalMessageInput) (msgTS string, err error)
}

const (
	OutboxStatusPending = "pending"
	OutboxStatusSent    = "sent"
)

// ProcessOutboxDue sends due pending outbox records and updates both the approval and outbox.
// It applies exponential backoff when posting fails.
func ProcessOutboxDue(ctx context.Context, store ledger.Store, poster OutboxPoster, now time.Time, limit int) (int, error) {
	if store == nil {
		return 0, fmt.Errorf("missing store")
	}
	if poster == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 50
	}

	due, err := store.ListSlackOutboxDue(now.UTC().Format(time.RFC3339), limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, rec := range due {
		if err := ctx.Err(); err != nil {
			return processed, err
		}

		if rec.Status != OutboxStatusPending {
			continue
		}

		approval, ok := store.GetApproval(rec.ApprovalID)
		if ok && approval.SlackMsgTS != nil && *approval.SlackMsgTS != "" {
			// Already posted; mark outbox as sent.
			rec.Status = OutboxStatusSent
			sentAt := now.UTC().Format(time.RFC3339)
			rec.SentAt = &sentAt
			rec.UpdatedAt = sentAt
			if err := store.PutSlackOutbox(rec); err != nil {
				return processed, err
			}
			processed++
			continue
		}

		var input ApprovalMessageInput
		if err := json.Unmarshal(rec.MessageJSON, &input); err != nil {
			// Bad payload; mark as sent to prevent infinite retries.
			msg := "invalid message_json: " + err.Error()
			rec.LastError = &msg
			rec.Status = OutboxStatusSent
			sentAt := now.UTC().Format(time.RFC3339)
			rec.SentAt = &sentAt
			rec.UpdatedAt = sentAt
			if err := store.PutSlackOutbox(rec); err != nil {
				return processed, err
			}
			processed++
			continue
		}

		msgTS, err := poster.PostApproval(rec.Channel, input)
		if err != nil {
			next := nextAttempt(rec.AttemptCount)
			rec.AttemptCount++
			nextAt := now.UTC().Add(next).Format(time.RFC3339)
			rec.NextAttemptAt = nextAt
			msg := err.Error()
			rec.LastError = &msg
			rec.UpdatedAt = now.UTC().Format(time.RFC3339)
			if err := store.PutSlackOutbox(rec); err != nil {
				return processed, err
			}
			processed++
			continue
		}

		// Update approval with Slack metadata (best-effort).
		if ok {
			channel := rec.Channel
			approval.SlackChannel = &channel
			approval.SlackMsgTS = &msgTS
			approval.UpdatedAt = now.UTC().Format(time.RFC3339)
			_ = store.PutApproval(approval)
		}

		rec.Status = OutboxStatusSent
		sentAt := now.UTC().Format(time.RFC3339)
		rec.SentAt = &sentAt
		rec.UpdatedAt = sentAt
		if err := store.PutSlackOutbox(rec); err != nil {
			return processed, err
		}
		processed++
	}

	return processed, nil
}

func nextAttempt(attemptCount int) time.Duration {
	// 5s, 10s, 20s, 40s, 80s, 160s, ... capped at 5m.
	base := 5 * time.Second
	if attemptCount <= 0 {
		return base
	}
	d := base << attemptCount
	max := 5 * time.Minute
	if d > max {
		return max
	}
	return d
}

// RunOutboxWorker polls and processes due Slack outbox entries until ctx is cancelled.
func RunOutboxWorker(ctx context.Context, store ledger.Store, poster OutboxPoster, pollInterval time.Duration) {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_, _ = ProcessOutboxDue(ctx, store, poster, now, 25)
		}
	}
}
