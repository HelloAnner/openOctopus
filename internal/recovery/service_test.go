/*
Package recovery service_test 验证 recovery 首版恢复闭环。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configservice "github.com/anner/openoctopus/internal/config/service"
	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/session"
)

func TestValidateSessionLayoutAllowsMissingSessionState(t *testing.T) {
	sessionDir := prepareRecoverySession(t)
	if err := os.Remove(filepath.Join(sessionDir, "session.state.md")); err != nil {
		t.Fatalf("remove state: %v", err)
	}

	service := NewService(sessionDir)
	result, err := service.validateSessionLayout()
	if err != nil {
		t.Fatalf("validate session layout: %v", err)
	}
	if len(result.RepairableFiles) != 1 || result.RepairableFiles[0] != "session.state.md" {
		t.Fatalf("expected missing session.state.md to be repairable, got %+v", result.RepairableFiles)
	}
}

func TestValidateEventChainReturnsStableError(t *testing.T) {
	sessionDir := prepareRecoverySession(t)
	eventsPath := filepath.Join(sessionDir, "bus", "events.md")
	content, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	rewritten := strings.Replace(string(content), "- event_hash: ", "- event_hash: broken-", 1)
	if err := os.WriteFile(eventsPath, []byte(rewritten), 0o644); err != nil {
		t.Fatalf("rewrite events: %v", err)
	}

	service := NewService(sessionDir)
	_, err = service.validateSessionLayout()
	if !errors.Is(err, eventbus.ErrEventChainBroken) {
		t.Fatalf("expected ErrEventChainBroken, got %v", err)
	}
}

func TestRecoverRepairsMissingStateAndWritesReplay(t *testing.T) {
	sessionDir := prepareRecoverySession(t)
	if err := os.Remove(filepath.Join(sessionDir, "session.state.md")); err != nil {
		t.Fatalf("remove state: %v", err)
	}

	service := NewService(sessionDir)
	result, err := service.Recover(RecoverOptions{})
	if err != nil {
		t.Fatalf("recover session: %v", err)
	}
	if !result.CanContinue {
		t.Fatalf("expected recoverable session to be continuable, got %+v", result)
	}
	if result.CheckpointRef == "" {
		t.Fatalf("expected checkpoint ref, got %+v", result)
	}
	assertFileContains(t, filepath.Join(sessionDir, "session.state.md"), "status: RUNNING")
	assertFileContains(t, filepath.Join(sessionDir, "audit", "replay.md"), "continued: true")
	assertFileContains(t, filepath.Join(sessionDir, "audit", "replay.md"), "checkpoint_ref: "+result.CheckpointRef)
	assertFileContains(t, filepath.Join(sessionDir, "state", "checkpoints", filepath.Base(result.CheckpointRef)), "kind: recover-start")
}

func TestRecoverMarksWaitingHumanAsNotContinuable(t *testing.T) {
	sessionDir := prepareRecoverySession(t)
	assertFileContains(t, filepath.Join(sessionDir, "planner", "master_schedule.md"), "status: DISPATCHED")

	rewriteWaitingHumanFiles(t, sessionDir)

	service := NewService(sessionDir)
	result, err := service.Recover(RecoverOptions{})
	if err != nil {
		t.Fatalf("recover waiting human session: %v", err)
	}
	if result.CanContinue {
		t.Fatalf("expected waiting human session not continuable, got %+v", result)
	}
	if result.RecoveredStatus != workflowStatusWaitingHuman {
		t.Fatalf("expected WAITING_HUMAN, got %+v", result)
	}
	if result.Reason != "needs_human_resume" {
		t.Fatalf("expected needs_human_resume, got %+v", result)
	}
	assertFileContains(t, filepath.Join(sessionDir, "planner", "blockers.md"), "summary: manual review")
	assertFileContains(t, filepath.Join(sessionDir, "audit", "replay.md"), "reason: needs_human_resume")
}

func prepareRecoverySession(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	configPath := filepath.Join(root, "octopus.yaml")
	config := `
version: "2.1"

meta:
  workflow_id: "recovery-unit"
  name: "Recovery Unit"

llm_profiles:
  deterministic_cli:
    provider: "deterministic"
    mode: "cli"
    cli_path: "deterministic"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"

roles:
  - id: "agent_a"
    name: "Agent A"
    type: "react"
    llm_profile: "deterministic_cli"
    system_prompt: "你负责执行任务。"
    tools: ["file_read"]

stages:
  - id: "stage_a"
    name: "Stage A"
    role: "agent_a"
    output:
      - type: "artifact"
        name: "artifact_a"

transitions:
  - from: "stage_a"
    to: "__END__"
`
	if err := os.WriteFile(configPath, []byte(strings.TrimSpace(config)+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := configservice.LoadForValidate(configservice.LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(loaded.Errors) != 0 {
		t.Fatalf("expected valid config, got %d errors", len(loaded.Errors))
	}
	created, err := session.Create(session.CreateOptions{
		Config:          loaded.Config,
		ConfigPath:      configPath,
		AppliedDefaults: loaded.AppliedDefaults,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	store := eventbus.NewStore(created.SessionDir)
	if err := store.Bootstrap(eventbus.BootstrapOptions{
		SessionID:   created.SessionID,
		SessionDir:  created.SessionDir,
		WorkflowID:  loaded.Config.Meta.WorkflowID,
		MetadataRef: "metadata.md",
	}); err != nil {
		t.Fatalf("bootstrap bus: %v", err)
	}
	if err := writeRecoveryFixtureFiles(created.SessionDir, created.SessionID); err != nil {
		t.Fatalf("write recovery fixture files: %v", err)
	}
	return created.SessionDir
}

func writeRecoveryFixtureFiles(sessionDir string, sessionID string) error {
	schedule := `# Master Schedule

- schedule_version: 1
- workflow_status: RUNNING
- active_dispatch_count: 1
- updated_at: 2026-03-08T09:00:00Z

## stage: stage_a
- stage_id: stage_a
- stage_name: Stage A
- role_id: agent_a
- status: DISPATCHED
- attempt: 1
- last_task_id: task-stage_a-01
- last_conclusion_ref:
- next_stage_id: __END__
- updated_at: 2026-03-08T09:00:00Z
`
	state := `# Session State

- session_id: ` + sessionID + `
- status: RUNNING
- current_stage_id: stage_a
- current_role_id: agent_a
- checkpoint_seq: 0
- last_event: TASK_DISPATCHED
- created_at: 2026-03-08T09:00:00Z
- updated_at: 2026-03-08T09:00:00Z
- human_message_cursor:
`
	blockers := `# Blockers

- summary: clear
- updated_at: 2026-03-08T09:00:00Z
`
	if err := os.WriteFile(filepath.Join(sessionDir, "planner", "master_schedule.md"), []byte(schedule), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "session.state.md"), []byte(state), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sessionDir, "planner", "blockers.md"), []byte(blockers), 0o644)
}

func rewriteWaitingHumanFiles(t *testing.T, sessionDir string) {
	t.Helper()
	schedule := `# Master Schedule

- schedule_version: 2
- workflow_status: WAITING_HUMAN
- active_dispatch_count: 0
- updated_at: 2026-03-08T09:00:00Z

## stage: stage_a
- stage_id: stage_a
- stage_name: Stage A
- role_id: agent_a
- status: BLOCKED
- attempt: 1
- last_task_id: task-stage_a-01
- last_conclusion_ref: roles/agent_a/conclusion.md
- next_stage_id: __END__
- updated_at: 2026-03-08T09:00:00Z
`
	if err := os.WriteFile(filepath.Join(sessionDir, "planner", "master_schedule.md"), []byte(schedule), 0o644); err != nil {
		t.Fatalf("write schedule: %v", err)
	}
	state := `# Session State

- session_id: sess_test
- status: WAITING_HUMAN
- current_stage_id: stage_a
- current_role_id: agent_a
- checkpoint_seq: 1
- last_event: STAGE_BLOCKED
- created_at: 2026-03-08T09:00:00Z
- updated_at: 2026-03-08T09:00:00Z
- human_message_cursor:
`
	if err := os.WriteFile(filepath.Join(sessionDir, "session.state.md"), []byte(state), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
	blockers := `# Blockers

- summary: manual review
- updated_at: 2026-03-08T09:00:00Z
`
	if err := os.WriteFile(filepath.Join(sessionDir, "planner", "blockers.md"), []byte(blockers), 0o644); err != nil {
		t.Fatalf("write blockers: %v", err)
	}
	conclusion := `# Role Conclusion

- role_id: agent_a
- stage_id: stage_a
- task_id: task-stage_a-01
- status: BLOCKED
- summary: manual review
- output_refs:
- updated_at: 2026-03-08T09:00:00Z
`
	if err := os.MkdirAll(filepath.Join(sessionDir, "roles", "agent_a"), 0o755); err != nil {
		t.Fatalf("mkdir role: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "roles", "agent_a", "conclusion.md"), []byte(conclusion), 0o644); err != nil {
		t.Fatalf("write conclusion: %v", err)
	}
}

func assertFileContains(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(content), expected) {
		t.Fatalf("expected %s to contain %q, got %q", path, expected, string(content))
	}
}
