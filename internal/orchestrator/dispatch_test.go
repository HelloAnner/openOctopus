package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/session"
)


func TestTickDispatchesRequirementFileIntoSessionContext(t *testing.T) {
	useFixedOrchestratorClock(t)
	result := createRequirementOrchestratorSession(t)
	store := eventbus.NewStore(result.SessionDir)
	if err := store.Bootstrap(eventbus.BootstrapOptions{SessionID: result.SessionID, SessionDir: result.SessionDir, WorkflowID: "orchestrator-workflow", MetadataRef: "metadata.md"}); err != nil {
		t.Fatalf("bootstrap bus: %v", err)
	}
	requirementsDir := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(result.SessionDir))), "docs", "prd")
	if err := os.MkdirAll(requirementsDir, 0o755); err != nil {
		t.Fatalf("mkdir requirements: %v", err)
	}
	sourcePath := filepath.Join(requirementsDir, "prd-001.md")
	if err := os.WriteFile(sourcePath, []byte("# PRD\n"), 0o644); err != nil {
		t.Fatalf("write requirement source: %v", err)
	}
	engine := NewEngine(result.SessionDir)
	if err := engine.Bootstrap(); err != nil {
		t.Fatalf("bootstrap orchestrator: %v", err)
	}

	if _, err := engine.Tick(); err != nil {
		t.Fatalf("tick: %v", err)
	}

	contextFile := filepath.Join(result.SessionDir, "roles", "agent_a", "context.md")
	contextContent, err := os.ReadFile(contextFile)
	if err != nil {
		t.Fatalf("read context: %v", err)
	}
	body := string(contextContent)
	if !strings.Contains(body, "roles/agent_a/inputs/prd-001.md") {
		t.Fatalf("expected session requirement ref in context, got %q", body)
	}
	copiedPath := filepath.Join(result.SessionDir, "roles", "agent_a", "inputs", "prd-001.md")
	copiedContent, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatalf("read copied requirement: %v", err)
	}
	if string(copiedContent) != "# PRD\n" {
		t.Fatalf("expected copied requirement content, got %q", string(copiedContent))
	}
}

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


func createRequirementOrchestratorSession(t *testing.T) session.CreateResult {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "octopus.yaml")
	config := model.RuntimeConfig{
		Version: model.SupportedConfigVersion,
		Meta: model.MetaConfig{WorkflowID: "orchestrator-workflow", Name: "Orchestrator Workflow"},
		Runtime: model.RuntimeSection{
			Workspace: model.WorkspaceConfig{Root: ".octopus", SessionsDir: ".octopus/sessions"},
			Scheduler: model.SchedulerConfig{MaxParallelRoles: 1},
			MasterWatch: model.MasterWatchConfig{Enabled: true, ProgressFile: "planner/global_progress.md", BlockerFile: "planner/blockers.md", MaxNoProgressRounds: 3},
		},
		Policies: model.PoliciesConfig{Retry: model.RetryPolicy{MaxRetryPerStage: 2}, LoopGuard: model.LoopGuardPolicy{MaxRoundsPerTask: 6}},
		Roles: []model.RoleConfig{{ID: "agent_a", Name: "Agent A", Type: "react", LLMProfile: "codex_cli", SystemPrompt: "你负责执行任务。", Tools: []string{"file_read"}}},
		Stages: []model.StageConfig{{
			ID:   "stage_a",
			Name: "Stage A",
			Role: "agent_a",
			Input: []model.StageIO{{Type: "requirement_file", Path: "./docs/prd/prd-001.md"}},
			Output: []model.StageIO{{Type: "artifact", Name: "artifact_a"}},
		}},
		Transitions: []model.TransitionConfig{{From: "stage_a", To: model.EndStage}},
	}
	result, err := session.Create(session.CreateOptions{Config: config, ConfigPath: configPath})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return result
}

