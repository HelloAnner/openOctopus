package roleruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anner/openoctopus/internal/eventbus"
)

func TestInterruptAcknowledgedBlocksExecutionUntilClear(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")
	sessionDir := prepareDispatchedSession(t)
	store := eventbus.NewStore(sessionDir)

	lease, err := store.AcquireLock("human-gate", 30*time.Second)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	record, err := store.RequestInterrupt(lease, eventbus.InterruptRequest{
		Scope:        eventbus.InterruptScopeRole,
		TargetRoleID: "agent_a",
		Source:       "human-gate",
		Reason:       "manual review",
	})
	if err != nil {
		t.Fatalf("request interrupt: %v", err)
	}
	if err := store.ReleaseLock(lease); err != nil {
		t.Fatalf("release lock: %v", err)
	}

	engine := NewEngine(sessionDir)
	first, err := engine.TickRole("agent_a")
	if err != nil {
		t.Fatalf("first tick: %v", err)
	}
	if !first.Progressed || first.Status != statusInterrupted {
		t.Fatalf("expected interrupt progress, got %+v", first)
	}
	assertNoTurnFiles(t, sessionDir)
	assertInterruptFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "state.md"), "- status: INTERRUPTED")

	second, err := engine.TickRole("agent_a")
	if err != nil {
		t.Fatalf("second tick: %v", err)
	}
	if second.Progressed {
		t.Fatalf("expected acknowledged interrupt to block execution, got %+v", second)
	}
	assertNoTurnFiles(t, sessionDir)

	lease, err = store.AcquireLock("human-gate", 30*time.Second)
	if err != nil {
		t.Fatalf("reacquire lock: %v", err)
	}
	if _, err := store.ClearInterrupt(lease, record.InterruptID); err != nil {
		t.Fatalf("clear interrupt: %v", err)
	}
	if err := store.ReleaseLock(lease); err != nil {
		t.Fatalf("release lock after clear: %v", err)
	}

	third, err := engine.TickRole("agent_a")
	if err != nil {
		t.Fatalf("third tick: %v", err)
	}
	if !third.Progressed || third.TurnSeq != 1 {
		t.Fatalf("expected execution after clear, got %+v", third)
	}
	assertInterruptFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "conclusion.md"), "- status: SUCCESS")
}

func assertNoTurnFiles(t *testing.T, sessionDir string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(sessionDir, "roles", "agent_a", "turns"))
	if err != nil {
		t.Fatalf("read turns dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no turn files, got %d", len(entries))
	}
}

func assertInterruptFileContains(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), expected) {
		t.Fatalf("expected %s to contain %q, got %q", path, expected, string(content))
	}
}
