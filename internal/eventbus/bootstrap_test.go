package eventbus

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapReplacesSessionPlaceholdersAndIsIdempotent(t *testing.T) {
	useFixedEventBusClock(t)
	result := createEventBusTestSession(t)
	store := NewStore(result.SessionDir)

	placeholderContent, err := os.ReadFile(filepath.Join(result.SessionDir, "bus", "events.md"))
	if err != nil {
		t.Fatalf("read placeholder: %v", err)
	}
	if !strings.Contains(string(placeholderContent), "Initialized by session 001.") {
		t.Fatalf("expected session placeholder in events file, got %q", string(placeholderContent))
	}

	err = store.Bootstrap(BootstrapOptions{
		SessionID:   result.SessionID,
		SessionDir:  result.SessionDir,
		WorkflowID:  "eventbus-workflow",
		MetadataRef: "metadata.md",
	})
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	eventsFile, err := os.ReadFile(filepath.Join(result.SessionDir, "bus", "events.md"))
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	content := string(eventsFile)
	if strings.Contains(content, "Initialized by session 001.") {
		t.Fatalf("expected placeholder to be replaced, got %q", content)
	}
	if !strings.Contains(content, "event-000001") || !strings.Contains(content, "SESSION_CREATED") {
		t.Fatalf("expected bootstrap event in events file, got %q", content)
	}

	events, err := store.List()
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after bootstrap, got %d", len(events))
	}

	err = store.Bootstrap(BootstrapOptions{
		SessionID:   result.SessionID,
		SessionDir:  result.SessionDir,
		WorkflowID:  "eventbus-workflow",
		MetadataRef: "metadata.md",
	})
	if err != nil {
		t.Fatalf("second bootstrap failed: %v", err)
	}

	events, err = store.List()
	if err != nil {
		t.Fatalf("list events after second bootstrap: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected bootstrap to be idempotent, got %d events", len(events))
	}
}

func TestErrorsCanBeMatchedWithErrorsIs(t *testing.T) {
	wrapped := errors.Join(ErrLeaseConflict, errors.New("wrapped"))
	if !errors.Is(wrapped, ErrLeaseConflict) {
		t.Fatal("expected ErrLeaseConflict to be matchable with errors.Is")
	}
	if !errors.Is(ErrEventChainBroken, ErrEventChainBroken) {
		t.Fatal("expected ErrEventChainBroken to match itself")
	}
}
