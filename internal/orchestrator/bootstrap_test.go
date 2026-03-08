package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/eventbus"
)

func TestBootstrapReplacesPlannerPlaceholdersAndIsIdempotent(t *testing.T) {
	useFixedOrchestratorClock(t)
	result := createOrchestratorTestSession(t)
	store := eventbus.NewStore(result.SessionDir)
	if err := store.Bootstrap(eventbus.BootstrapOptions{
		SessionID:   result.SessionID,
		SessionDir:  result.SessionDir,
		WorkflowID:  "orchestrator-workflow",
		MetadataRef: "metadata.md",
	}); err != nil {
		t.Fatalf("bootstrap bus: %v", err)
	}
	engine := NewEngine(result.SessionDir)

	before, err := os.ReadFile(filepath.Join(result.SessionDir, "planner", "master_schedule.md"))
	if err != nil {
		t.Fatalf("read placeholder: %v", err)
	}
	if !strings.Contains(string(before), "Initialized by session 001.") {
		t.Fatalf("expected session placeholder, got %q", string(before))
	}

	if err := engine.Bootstrap(); err != nil {
		t.Fatalf("bootstrap orchestrator: %v", err)
	}

	after, err := os.ReadFile(filepath.Join(result.SessionDir, "planner", "master_schedule.md"))
	if err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	if strings.Contains(string(after), "Initialized by session 001.") {
		t.Fatalf("expected placeholder replaced, got %q", string(after))
	}
	if !strings.Contains(string(after), "stage_id: stage_a") {
		t.Fatalf("expected stage in schedule, got %q", string(after))
	}

	if err := engine.Bootstrap(); err != nil {
		t.Fatalf("second bootstrap: %v", err)
	}
	second, err := os.ReadFile(filepath.Join(result.SessionDir, "planner", "master_schedule.md"))
	if err != nil {
		t.Fatalf("read schedule second time: %v", err)
	}
	if string(after) != string(second) {
		t.Fatalf("expected bootstrap to be idempotent")
	}
}
