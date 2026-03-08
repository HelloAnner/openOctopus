package eventbus

import (
	"errors"
	"testing"
	"time"
)

func TestCommitOffsetPreventsRegression(t *testing.T) {
	store, result := bootstrapEventBusStore(t)

	lease, err := store.AcquireLock("orchestrator/master", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	first, err := store.Append(lease, AppendEvent{
		EventType:  "SCHEDULE_UPDATED",
		Producer:   "orchestrator",
		SessionID:  result.SessionID,
		PayloadRef: "planner/master_schedule.md",
		Summary:    "schedule updated",
	})
	if err != nil {
		t.Fatalf("append event: %v", err)
	}
	second, err := store.Append(lease, AppendEvent{
		EventType:  "ROLE_DISPATCHED",
		Producer:   "orchestrator",
		SessionID:  result.SessionID,
		PayloadRef: "roles/agent_a/inbox.md",
		Summary:    "role dispatched",
	})
	if err != nil {
		t.Fatalf("append second event: %v", err)
	}

	err = store.CommitOffset(lease, OffsetCommit{
		ConsumerID:   "orchestrator/master",
		LastEventID:  second.EventID,
		LastSequence: second.Sequence,
		Note:         "schedule applied",
	})
	if err != nil {
		t.Fatalf("commit offset: %v", err)
	}

	offsets, err := store.ReadOffsets()
	if err != nil {
		t.Fatalf("read offsets: %v", err)
	}
	if len(offsets) != 1 || offsets[0].LastEventID != second.EventID {
		t.Fatalf("unexpected offsets after commit: %+v", offsets)
	}

	err = store.CommitOffset(lease, OffsetCommit{
		ConsumerID:   "orchestrator/master",
		LastEventID:  first.EventID,
		LastSequence: first.Sequence,
		Note:         "regression",
	})
	if !errors.Is(err, ErrOffsetRegression) {
		t.Fatalf("expected ErrOffsetRegression, got %v", err)
	}

	offsets, err = store.ReadOffsets()
	if err != nil {
		t.Fatalf("read offsets after failed regression: %v", err)
	}
	if len(offsets) != 1 || offsets[0].LastEventID != second.EventID {
		t.Fatalf("expected offsets to stay on latest commit, got %+v", offsets)
	}
}

func TestInterruptLifecycle(t *testing.T) {
	store, _ := bootstrapEventBusStore(t)

	lease, err := store.AcquireLock("human-gate", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}

	record, err := store.RequestInterrupt(lease, InterruptRequest{
		Scope:        InterruptScopeRole,
		TargetRoleID: "agent_a",
		Source:       "cli",
		Reason:       "waiting for approval",
	})
	if err != nil {
		t.Fatalf("request interrupt: %v", err)
	}
	if record.Status != InterruptStatusRequested {
		t.Fatalf("unexpected requested interrupt: %+v", record)
	}

	record, err = store.AcknowledgeInterrupt(lease, record.InterruptID)
	if err != nil {
		t.Fatalf("ack interrupt: %v", err)
	}
	if record.Status != InterruptStatusAcknowledged {
		t.Fatalf("unexpected acknowledged interrupt: %+v", record)
	}

	record, err = store.ClearInterrupt(lease, record.InterruptID)
	if err != nil {
		t.Fatalf("clear interrupt: %v", err)
	}
	if record.Status != InterruptStatusCleared {
		t.Fatalf("unexpected cleared interrupt: %+v", record)
	}

	records, err := store.ReadInterrupts()
	if err != nil {
		t.Fatalf("read interrupts: %v", err)
	}
	if len(records) != 1 || records[0].Status != InterruptStatusCleared {
		t.Fatalf("unexpected interrupt records: %+v", records)
	}

	_, err = store.AcknowledgeInterrupt(lease, "interrupt-missing")
	if !errors.Is(err, ErrInterruptNotFound) {
		t.Fatalf("expected ErrInterruptNotFound, got %v", err)
	}
	_, err = store.AcknowledgeInterrupt(lease, record.InterruptID)
	if err == nil {
		t.Fatal("expected illegal state transition after clear")
	}
	_ = time.Second
}
