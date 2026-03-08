package humangate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configservice "github.com/anner/openoctopus/internal/config/service"
	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/orchestrator"
	"github.com/anner/openoctopus/internal/roleruntime"
	"github.com/anner/openoctopus/internal/session"
)

func TestResolveSessionDirSupportsAbsoluteAndSessionID(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	sessionDir := filepath.Join(workingDir, ".octopus", "sessions", "sess_test")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}

	resolved, err := ResolveSessionDir(sessionDir, workingDir)
	if err != nil {
		t.Fatalf("resolve absolute session dir: %v", err)
	}
	if resolved != sessionDir {
		t.Fatalf("expected absolute session dir %q, got %q", sessionDir, resolved)
	}

	resolved, err = ResolveSessionDir("sess_test", workingDir)
	if err != nil {
		t.Fatalf("resolve session id: %v", err)
	}
	if resolved != sessionDir {
		t.Fatalf("expected session id to resolve to %q, got %q", sessionDir, resolved)
	}
}

func TestInjectAppendsHumanMessage(t *testing.T) {
	t.Parallel()

	sessionDir := prepareHumanGateSession(t)
	service := NewService(sessionDir)

	result, err := service.Inject(InjectOptions{RoleID: "agent_a", Message: "继续推进 stage_a"})
	if err != nil {
		t.Fatalf("inject message: %v", err)
	}
	if result.MessageID != "msg-000001" {
		t.Fatalf("expected first message id msg-000001, got %q", result.MessageID)
	}

	content, err := os.ReadFile(filepath.Join(sessionDir, "planner", "human_messages.md"))
	if err != nil {
		t.Fatalf("read human messages: %v", err)
	}
	text := string(content)
	assertContains(t, text, "message_id: msg-000001")
	assertContains(t, text, "source: human-gate")
	assertContains(t, text, "kind: inject")
	assertContains(t, text, "target_role_id: agent_a")
	assertContains(t, text, "继续推进 stage_a")
	assertContains(t, text, "### content")
}

func TestResumeClearsAcknowledgedInterrupt(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")

	sessionDir := prepareHumanGateSession(t)
	service := NewService(sessionDir)
	if _, err := service.Interrupt(InterruptOptions{RoleID: "agent_a", Reason: "manual review"}); err != nil {
		t.Fatalf("request interrupt: %v", err)
	}

	if _, err := roleruntime.NewEngine(sessionDir).TickRole("agent_a"); err != nil {
		t.Fatalf("ack interrupt via role runtime: %v", err)
	}

	result, err := service.Resume(ResumeOptions{RoleID: "agent_a"})
	if err != nil {
		t.Fatalf("resume session: %v", err)
	}
	if result.ClearedInterrupts != 1 {
		t.Fatalf("expected one cleared interrupt, got %+v", result)
	}

	interrupts, err := os.ReadFile(filepath.Join(sessionDir, "bus", "interrupts.md"))
	if err != nil {
		t.Fatalf("read interrupts: %v", err)
	}
	assertContains(t, string(interrupts), "status: CLEARED")
}

func TestResumeRequeuesBlockedStage(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "BLOCKED")

	sessionDir := prepareHumanGateSession(t)
	runtime := roleruntime.NewEngine(sessionDir)
	master := orchestrator.NewEngine(sessionDir)

	if _, err := runtime.TickRole("agent_a"); err != nil {
		t.Fatalf("tick blocked role: %v", err)
	}
	if _, err := master.Tick(); err != nil {
		t.Fatalf("apply blocked conclusion: %v", err)
	}

	service := NewService(sessionDir)
	result, err := service.Resume(ResumeOptions{})
	if err != nil {
		t.Fatalf("resume blocked session: %v", err)
	}
	if result.RequeuedStages != 1 {
		t.Fatalf("expected one requeued stage, got %+v", result)
	}

	schedule, err := os.ReadFile(filepath.Join(sessionDir, "planner", "master_schedule.md"))
	if err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	assertContains(t, string(schedule), "status: RETRY_PENDING")
	assertContains(t, string(schedule), "attempt: 2")

	state, err := os.ReadFile(filepath.Join(sessionDir, "session.state.md"))
	if err != nil {
		t.Fatalf("read session state: %v", err)
	}
	assertContains(t, string(state), "status: READY")

	blockers, err := os.ReadFile(filepath.Join(sessionDir, "planner", "blockers.md"))
	if err != nil {
		t.Fatalf("read blockers: %v", err)
	}
	assertContains(t, string(blockers), "summary: clear")
}

func prepareHumanGateSession(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	configPath := filepath.Join(root, "octopus.yaml")
	config := `
version: "2.1"

meta:
  workflow_id: "human-gate-unit"
  name: "Human Gate Unit"

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

func assertContains(t *testing.T, content string, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected content to contain %q, got %q", expected, content)
	}
}
