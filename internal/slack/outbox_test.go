package slack

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/davidahmann/relia/internal/ledger"
)

type flakyPoster struct {
	calls int
	fail  int
}

func (p *flakyPoster) PostApproval(channel string, message ApprovalMessageInput) (string, error) {
	p.calls++
	if p.calls <= p.fail {
		return "", errors.New("rate_limited")
	}
	return "1700000000.1234", nil
}

func TestProcessOutboxDue_RetryThenSuccess(t *testing.T) {
	store := ledger.NewInMemoryStore()

	approval := ledger.ApprovalRecord{
		ApprovalID: "a1",
		IdemKey:    "idem1",
		Status:     "pending",
		CreatedAt:  "now",
		UpdatedAt:  "now",
	}
	if err := store.PutApproval(approval); err != nil {
		t.Fatalf("put approval: %v", err)
	}

	input := ApprovalMessageInput{ApprovalID: "a1", Action: "x", Env: "prod", Resource: "r", Risk: "high"}
	msgBytes, _ := json.Marshal(input)

	now := time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC)
	rec := ledger.SlackOutboxRecord{
		NotificationID: "slack:a1",
		ApprovalID:     "a1",
		Channel:        "C1",
		MessageJSON:    msgBytes,
		Status:         OutboxStatusPending,
		AttemptCount:   0,
		NextAttemptAt:  now.Format(time.RFC3339),
		CreatedAt:      now.Format(time.RFC3339),
		UpdatedAt:      now.Format(time.RFC3339),
	}
	if err := store.PutSlackOutbox(rec); err != nil {
		t.Fatalf("put outbox: %v", err)
	}

	poster := &flakyPoster{fail: 1}
	if n, err := ProcessOutboxDue(context.Background(), store, poster, now, 10); err != nil || n != 1 {
		t.Fatalf("process: n=%d err=%v", n, err)
	}

	afterFail, ok := store.GetSlackOutbox("slack:a1")
	if !ok || afterFail.AttemptCount != 1 || afterFail.Status != OutboxStatusPending || afterFail.LastError == nil {
		t.Fatalf("unexpected after fail: %+v ok=%v", afterFail, ok)
	}

	// Move time forward to next attempt.
	now2 := now.Add(10 * time.Second)
	afterFail.NextAttemptAt = now2.Add(-1 * time.Second).Format(time.RFC3339)
	if err := store.PutSlackOutbox(afterFail); err != nil {
		t.Fatalf("put outbox: %v", err)
	}

	if n, err := ProcessOutboxDue(context.Background(), store, poster, now2, 10); err != nil || n != 1 {
		t.Fatalf("process2: n=%d err=%v", n, err)
	}

	final, ok := store.GetSlackOutbox("slack:a1")
	if !ok || final.Status != OutboxStatusSent || final.SentAt == nil {
		t.Fatalf("unexpected final: %+v ok=%v", final, ok)
	}

	updatedApproval, ok := store.GetApproval("a1")
	if !ok || updatedApproval.SlackMsgTS == nil || *updatedApproval.SlackMsgTS == "" {
		t.Fatalf("expected slack msg ts on approval: %+v ok=%v", updatedApproval, ok)
	}
}

func TestProcessOutboxDue_InvalidJSONMarksSent(t *testing.T) {
	store := ledger.NewInMemoryStore()
	_ = store.PutApproval(ledger.ApprovalRecord{ApprovalID: "a1", IdemKey: "idem1", Status: "pending", CreatedAt: "now", UpdatedAt: "now"})

	now := time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC)
	rec := ledger.SlackOutboxRecord{
		NotificationID: "slack:a1",
		ApprovalID:     "a1",
		Channel:        "C1",
		MessageJSON:    []byte("not-json"),
		Status:         OutboxStatusPending,
		AttemptCount:   0,
		NextAttemptAt:  now.Format(time.RFC3339),
		CreatedAt:      now.Format(time.RFC3339),
		UpdatedAt:      now.Format(time.RFC3339),
	}
	_ = store.PutSlackOutbox(rec)

	poster := &flakyPoster{fail: 0}
	if _, err := ProcessOutboxDue(context.Background(), store, poster, now, 10); err != nil {
		t.Fatalf("process: %v", err)
	}
	got, _ := store.GetSlackOutbox("slack:a1")
	if got.Status != OutboxStatusSent {
		t.Fatalf("expected sent, got %s", got.Status)
	}
}

func TestNextAttemptCapped(t *testing.T) {
	if got := nextAttempt(0); got != 5*time.Second {
		t.Fatalf("expected 5s, got %v", got)
	}
	if got := nextAttempt(1); got != 10*time.Second {
		t.Fatalf("expected 10s, got %v", got)
	}
	if got := nextAttempt(20); got != 5*time.Minute {
		t.Fatalf("expected cap 5m, got %v", got)
	}
}

func TestRunOutboxWorker(t *testing.T) {
	store := ledger.NewInMemoryStore()
	_ = store.PutApproval(ledger.ApprovalRecord{ApprovalID: "a1", IdemKey: "idem1", Status: "pending", CreatedAt: "now", UpdatedAt: "now"})
	_ = store.PutSlackOutbox(ledger.SlackOutboxRecord{
		NotificationID: "slack:a1",
		ApprovalID:     "a1",
		Channel:        "C1",
		MessageJSON:    []byte(`{"approval_id":"a1","action":"x","resource":"r","env":"prod","risk":"high"}`),
		Status:         OutboxStatusPending,
		AttemptCount:   0,
		NextAttemptAt:  time.Now().UTC().Add(-time.Second).Format(time.RFC3339),
		CreatedAt:      "now",
		UpdatedAt:      "now",
	})

	poster := &flakyPoster{fail: 0}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go RunOutboxWorker(ctx, store, poster, 5*time.Millisecond)

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		approval, ok := store.GetApproval("a1")
		if ok && approval.SlackMsgTS != nil && *approval.SlackMsgTS != "" {
			cancel()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("worker did not update approval in time")
}
