package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/eventbus"
)

func TestRequirementSnapshotAdvancesOnlyWhenNewMessagesArrive(t *testing.T) {
	useFixedOrchestratorClock(t)
	result := createOrchestratorTestSession(t)
	store := eventbus.NewStore(result.SessionDir)
	if err := store.Bootstrap(eventbus.BootstrapOptions{SessionID: result.SessionID, SessionDir: result.SessionDir, WorkflowID: "orchestrator-workflow", MetadataRef: "metadata.md"}); err != nil {
		t.Fatalf("bootstrap bus: %v", err)
	}
	engine := NewEngine(result.SessionDir)
	if err := engine.Bootstrap(); err != nil {
		t.Fatalf("bootstrap orchestrator: %v", err)
	}

	snapshotPath := filepath.Join(result.SessionDir, "planner", "requirement.snapshot.md")
	first, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read first snapshot: %v", err)
	}
	if !strings.Contains(string(first), "snapshot_version: 1") {
		t.Fatalf("expected initial snapshot version, got %q", string(first))
	}

	messagePath := filepath.Join(result.SessionDir, "planner", "human_messages.md")
	content := "# Human Messages\n\n## message: msg-000001\n- message_id: msg-000001\n- source: user\n- created_at: 2026-03-08T09:00:00Z\n\n### content\n继续设计 orchestrator。\n"
	if err := os.WriteFile(messagePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write messages: %v", err)
	}

	if _, err := engine.Tick(); err != nil {
		t.Fatalf("tick after new message: %v", err)
	}
	second, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read second snapshot: %v", err)
	}
	if !strings.Contains(string(second), "snapshot_version: 2") || !strings.Contains(string(second), "human_message_cursor: msg-000001") {
		t.Fatalf("expected updated snapshot, got %q", string(second))
	}

	if _, err := engine.Tick(); err != nil {
		t.Fatalf("tick without new messages: %v", err)
	}
	third, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read third snapshot: %v", err)
	}
	if !strings.Contains(string(third), "snapshot_version: 2") {
		t.Fatalf("expected snapshot version unchanged without new messages, got %q", string(third))
	}
}
