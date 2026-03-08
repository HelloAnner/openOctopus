package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/eventbus"
)

func TestTickAppliesConclusionStatuses(t *testing.T) {
	useFixedOrchestratorClock(t)
	cases := []struct {
		name           string
		status         string
		expectState    string
		expectSchedule string
	}{
		{name: "success", status: "SUCCESS", expectState: "COMPLETED", expectSchedule: "status: COMPLETED"},
		{name: "retry", status: "NEEDS_RETRY", expectState: "RUNNING", expectSchedule: "status: DISPATCHED"},
		{name: "blocked", status: "BLOCKED", expectState: "WAITING_HUMAN", expectSchedule: "status: BLOCKED"},
		{name: "failed", status: "FAILED", expectState: "FAILED", expectSchedule: "status: FAILED"},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			result := createOrchestratorTestSession(t)
			store := eventbus.NewStore(result.SessionDir)
			if err := store.Bootstrap(eventbus.BootstrapOptions{SessionID: result.SessionID, SessionDir: result.SessionDir, WorkflowID: "orchestrator-workflow", MetadataRef: "metadata.md"}); err != nil {
				t.Fatalf("bootstrap bus: %v", err)
			}
			engine := NewEngine(result.SessionDir)
			if err := engine.Bootstrap(); err != nil {
				t.Fatalf("bootstrap orchestrator: %v", err)
			}
			if _, err := engine.Tick(); err != nil {
				t.Fatalf("dispatch tick: %v", err)
			}
			conclusionPath := filepath.Join(result.SessionDir, "roles", "agent_a", "conclusion.md")
			body := "# Role Conclusion\n\n- role_id: agent_a\n- stage_id: stage_a\n- task_id: task-stage_a-01\n- status: " + item.status + "\n- summary: test summary\n- output_refs: \n- updated_at: 2026-03-08T09:00:00Z\n"
			if err := os.WriteFile(conclusionPath, []byte(body), 0o644); err != nil {
				t.Fatalf("write conclusion: %v", err)
			}
			if item.status == "SUCCESS" {
				turnDir := filepath.Join(result.SessionDir, "roles", "agent_a", "turns")
				if err := os.MkdirAll(turnDir, 0o755); err != nil {
					t.Fatalf("mkdir turns: %v", err)
				}
				if err := os.WriteFile(filepath.Join(turnDir, "0001-output.md"), []byte("# output\n"), 0o644); err != nil {
					t.Fatalf("write turn output: %v", err)
				}
				outbox := "# Role Outbox\n\n- outbox_version: 1\n- role_id: agent_a\n- stage_id: stage_a\n- task_id: task-stage_a-01\n- turn_seq: 1\n- status: SUCCESS\n- conclusion_ref: roles/agent_a/conclusion.md\n- turn_output_ref: roles/agent_a/turns/0001-output.md\n- updated_at: 2026-03-08T09:00:00Z\n"
				if err := os.WriteFile(filepath.Join(result.SessionDir, "roles", "agent_a", "outbox.md"), []byte(outbox), 0o644); err != nil {
					t.Fatalf("write outbox: %v", err)
				}
			}
			if _, err := engine.Tick(); err != nil {
				t.Fatalf("apply conclusion tick: %v", err)
			}
			stateContent, err := os.ReadFile(filepath.Join(result.SessionDir, "session.state.md"))
			if err != nil {
				t.Fatalf("read state: %v", err)
			}
			if !strings.Contains(string(stateContent), "status: "+item.expectState) {
				t.Fatalf("expected session state %q, got %q", item.expectState, string(stateContent))
			}
			scheduleContent, err := os.ReadFile(filepath.Join(result.SessionDir, "planner", "master_schedule.md"))
			if err != nil {
				t.Fatalf("read schedule: %v", err)
			}
			if !strings.Contains(string(scheduleContent), item.expectSchedule) {
				t.Fatalf("expected schedule to contain %q, got %q", item.expectSchedule, string(scheduleContent))
			}
		})
	}
}
