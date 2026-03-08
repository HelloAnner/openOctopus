package roleruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/config/service"
	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/orchestrator"
	"github.com/anner/openoctopus/internal/session"
)

func TestTickRoleWritesTurnFilesAndConclusion(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")
	sessionDir := prepareDispatchedSession(t)

	engine := NewEngine(sessionDir)
	result, err := engine.TickRole("agent_a")
	if err != nil {
		t.Fatalf("expected tick role to succeed: %v", err)
	}
	if !result.Progressed {
		t.Fatal("expected role runtime tick to report progress")
	}
	if result.TurnSeq != 1 {
		t.Fatalf("expected turn seq 1, got %d", result.TurnSeq)
	}

	assertFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "state.md"), "- status: COMPLETED")
	assertFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "conclusion.md"), "- status: SUCCESS")
	assertFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "outbox.md"), "- turn_seq: 1")
	assertFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "heartbeat.md"), "- current_turn_seq: 1")
	assertFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "turns", "0001-input.md"), "- task_id: task-stage_a-01")
	assertFileContains(t, filepath.Join(sessionDir, "roles", "agent_a", "turns", "0001-output.md"), "## role_result")
}

func TestTickRoleIsIdempotentForSameDispatch(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")
	sessionDir := prepareDispatchedSession(t)

	engine := NewEngine(sessionDir)
	first, err := engine.TickRole("agent_a")
	if err != nil {
		t.Fatalf("first tick role: %v", err)
	}
	if !first.Progressed {
		t.Fatal("expected first tick to make progress")
	}

	second, err := engine.TickRole("agent_a")
	if err != nil {
		t.Fatalf("second tick role: %v", err)
	}
	if second.Progressed {
		t.Fatal("expected second tick to be idempotent for same dispatch")
	}

	entries, err := os.ReadDir(filepath.Join(sessionDir, "roles", "agent_a", "turns"))
	if err != nil {
		t.Fatalf("read turns dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected exactly two turn files, got %d", len(entries))
	}
}

func prepareDispatchedSession(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	configPath := filepath.Join(root, "octopus.yaml")
	config := `
version: "2.1"

meta:
  workflow_id: "role-runtime-unit"
  name: "Role Runtime Unit"

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

	loaded, err := service.LoadForValidate(service.LoadOptions{ConfigPath: configPath})
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
		t.Fatalf("bootstrap event bus: %v", err)
	}
	master := orchestrator.NewEngine(created.SessionDir)
	if err := master.Bootstrap(); err != nil {
		t.Fatalf("bootstrap orchestrator: %v", err)
	}
	if _, err := master.Tick(); err != nil {
		t.Fatalf("dispatch first task: %v", err)
	}
	return created.SessionDir
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
