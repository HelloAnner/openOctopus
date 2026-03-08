package eventbus

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendEventsMaintainsSequenceAndHashChain(t *testing.T) {
	store, result := bootstrapEventBusStore(t)

	lease, err := store.AcquireLock("orchestrator/master", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}

	first, err := store.Append(lease, AppendEvent{
		EventType:     "SCHEDULE_UPDATED",
		Producer:      "orchestrator",
		SessionID:     result.SessionID,
		CorrelationID: "task-1",
		PayloadRef:    "planner/master_schedule.md",
		Summary:       "schedule updated",
	})
	if err != nil {
		t.Fatalf("append first event: %v", err)
	}
	second, err := store.Append(lease, AppendEvent{
		EventType:     "ROLE_DISPATCHED",
		Producer:      "orchestrator",
		SessionID:     result.SessionID,
		RoleID:        "agent_a",
		CorrelationID: "task-1",
		PayloadRef:    "roles/agent_a/inbox.md",
		Summary:       "role dispatched",
	})
	if err != nil {
		t.Fatalf("append second event: %v", err)
	}

	if first.EventID != "event-000002" || second.EventID != "event-000003" {
		t.Fatalf("unexpected event ids: %+v %+v", first, second)
	}
	if first.Sequence != 2 || second.Sequence != 3 {
		t.Fatalf("unexpected sequences: %+v %+v", first, second)
	}
	if second.PrevEventHash != first.EventHash {
		t.Fatalf("expected hash chain to link, first=%q second.prev=%q", first.EventHash, second.PrevEventHash)
	}

	tail, err := store.Tail()
	if err != nil {
		t.Fatalf("tail event: %v", err)
	}
	if tail.EventID != second.EventID {
		t.Fatalf("expected tail %q, got %q", second.EventID, tail.EventID)
	}
}

func TestListAfterAndDetectBrokenEventChain(t *testing.T) {
	store, result := bootstrapEventBusStore(t)

	lease, err := store.AcquireLock("orchestrator/master", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	_, err = store.Append(lease, AppendEvent{
		EventType:  "SCHEDULE_UPDATED",
		Producer:   "orchestrator",
		SessionID:  result.SessionID,
		PayloadRef: "planner/master_schedule.md",
		Summary:    "schedule updated",
	})
	if err != nil {
		t.Fatalf("append first event: %v", err)
	}
	_, err = store.Append(lease, AppendEvent{
		EventType:  "ROLE_DISPATCHED",
		Producer:   "orchestrator",
		SessionID:  result.SessionID,
		PayloadRef: "roles/agent_a/inbox.md",
		Summary:    "role dispatched",
	})
	if err != nil {
		t.Fatalf("append second event: %v", err)
	}

	afterEvents, err := store.ListAfter("event-000001")
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(afterEvents) != 2 {
		t.Fatalf("expected 2 events after bootstrap event, got %d", len(afterEvents))
	}

	eventsPath := filepath.Join(result.SessionDir, "bus", "events.md")
	content, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("read events file: %v", err)
	}
	broken := strings.Replace(string(content), "- prev_event_hash: ", "- prev_event_hash: sha256:broken-", 1)
	if err := os.WriteFile(eventsPath, []byte(broken), 0o644); err != nil {
		t.Fatalf("write broken events file: %v", err)
	}

	_, err = store.List()
	if !errors.Is(err, ErrEventChainBroken) {
		t.Fatalf("expected ErrEventChainBroken, got %v", err)
	}
}
