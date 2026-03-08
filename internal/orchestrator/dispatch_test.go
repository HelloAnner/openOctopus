package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/eventbus"
)

func TestTickDispatchesReadyStageAndWritesRolePackage(t *testing.T) {
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

	outcome, err := engine.Tick()
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if outcome.DispatchedCount != 1 {
		t.Fatalf("expected one dispatch, got %+v", outcome)
	}

	contextFile := filepath.Join(result.SessionDir, "roles", "agent_a", "context.md")
	inboxFile := filepath.Join(result.SessionDir, "roles", "agent_a", "inbox.md")
	contextContent, err := os.ReadFile(contextFile)
	if err != nil {
		t.Fatalf("read context: %v", err)
	}
	inboxContent, err := os.ReadFile(inboxFile)
	if err != nil {
		t.Fatalf("read inbox: %v", err)
	}
	if !strings.Contains(string(contextContent), "stage_id: stage_a") || !strings.Contains(string(inboxContent), "stage_id: stage_a") {
		t.Fatalf("expected stage id in role package")
	}

	events, err := store.List()
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	found := false
	for _, item := range events {
		if item.EventType == "TASK_DISPATCHED" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected TASK_DISPATCHED event, got %+v", events)
	}
}
