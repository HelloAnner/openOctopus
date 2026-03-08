package orchestrator

import (
    "os"
    "path/filepath"
    "strings"
    "testing"

    artifactstore "github.com/anner/openoctopus/internal/artifact"
    "github.com/anner/openoctopus/internal/config/model"
    "github.com/anner/openoctopus/internal/eventbus"
    "github.com/anner/openoctopus/internal/session"
)

func TestTickDispatchesArtifactContractsToContext(t *testing.T) {
    useFixedOrchestratorClock(t)
    result := createArtifactOrchestratorSession(t)
    bootstrapArtifactStore(t, result.SessionDir)
    bootstrapEventBus(t, result.SessionDir, result.SessionID)

    engine := NewEngine(result.SessionDir)
    if err := engine.Bootstrap(); err != nil {
        t.Fatalf("bootstrap orchestrator: %v", err)
    }
    if _, err := engine.Tick(); err != nil {
        t.Fatalf("tick: %v", err)
    }

    contextContent, err := os.ReadFile(filepath.Join(result.SessionDir, "roles", "agent_a", "context.md"))
    if err != nil {
        t.Fatalf("read context: %v", err)
    }
    body := string(contextContent)
    if !strings.Contains(body, "## output_artifacts") {
        t.Fatalf("expected output_artifacts section, got %q", body)
    }
    if !strings.Contains(body, "artifacts/_staging/stage_a/artifact_a.md") {
        t.Fatalf("expected suggested artifact ref, got %q", body)
    }
}

func TestTickPublishesArtifactAndProjectsInputToNextStage(t *testing.T) {
    useFixedOrchestratorClock(t)
    result := createArtifactOrchestratorSession(t)
    bootstrapArtifactStore(t, result.SessionDir)
    store := bootstrapEventBus(t, result.SessionDir, result.SessionID)

    engine := NewEngine(result.SessionDir)
    if err := engine.Bootstrap(); err != nil {
        t.Fatalf("bootstrap orchestrator: %v", err)
    }
    if _, err := engine.Tick(); err != nil {
        t.Fatalf("dispatch stage_a: %v", err)
    }

    sourceRef := filepath.Join(result.SessionDir, "artifacts", "_staging", "stage_a", "artifact_a.md")
    if err := os.MkdirAll(filepath.Dir(sourceRef), 0o755); err != nil {
        t.Fatalf("mkdir source: %v", err)
    }
    if err := os.WriteFile(sourceRef, []byte("# artifact a\n"), 0o644); err != nil {
        t.Fatalf("write source: %v", err)
    }
    writeConclusionAndOutbox(t, result.SessionDir, "agent_a", "stage_a", "task-stage_a-01", "SUCCESS", "artifacts/_staging/stage_a/artifact_a.md")

    if _, err := engine.Tick(); err != nil {
        t.Fatalf("apply stage_a success: %v", err)
    }
    if _, err := engine.Tick(); err != nil {
        t.Fatalf("dispatch stage_b: %v", err)
    }

    indexContent, err := os.ReadFile(filepath.Join(result.SessionDir, "artifacts", "index.md"))
    if err != nil {
        t.Fatalf("read index: %v", err)
    }
    if !strings.Contains(string(indexContent), "artifact_name: artifact_a") {
        t.Fatalf("expected artifact index entry, got %q", string(indexContent))
    }

    contextContent, err := os.ReadFile(filepath.Join(result.SessionDir, "roles", "agent_b", "context.md"))
    if err != nil {
        t.Fatalf("read stage_b context: %v", err)
    }
    body := string(contextContent)
    if !strings.Contains(body, "## input_artifacts") {
        t.Fatalf("expected input_artifacts section, got %q", body)
    }
    if !strings.Contains(body, "artifacts/artifact_a/0001/content.md") {
        t.Fatalf("expected published artifact content ref, got %q", body)
    }

    events, err := store.List()
    if err != nil {
        t.Fatalf("list events: %v", err)
    }
    found := false
    for _, item := range events {
        if item.EventType == "ARTIFACT_PUBLISHED" {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("expected ARTIFACT_PUBLISHED event, got %+v", events)
    }
}

func createArtifactOrchestratorSession(t *testing.T) session.CreateResult {
    t.Helper()
    tempDir := t.TempDir()
    configPath := filepath.Join(tempDir, "octopus.yaml")
    config := model.RuntimeConfig{
        Version: model.SupportedConfigVersion,
        Meta: model.MetaConfig{WorkflowID: "artifact-orchestrator", Name: "Artifact Orchestrator"},
        Runtime: model.RuntimeSection{
            Workspace: model.WorkspaceConfig{Root: ".octopus", SessionsDir: ".octopus/sessions"},
            Scheduler: model.SchedulerConfig{MaxParallelRoles: 1},
            MasterWatch: model.MasterWatchConfig{Enabled: true, ProgressFile: "planner/global_progress.md", BlockerFile: "planner/blockers.md", MaxNoProgressRounds: 3},
        },
        Policies: model.PoliciesConfig{Retry: model.RetryPolicy{MaxRetryPerStage: 2}, LoopGuard: model.LoopGuardPolicy{MaxRoundsPerTask: 6}},
        Roles: []model.RoleConfig{
            {ID: "agent_a", Name: "Agent A", Type: "react", LLMProfile: "codex_cli", SystemPrompt: "你负责生成产物。", Tools: []string{"file_read"}},
            {ID: "agent_b", Name: "Agent B", Type: "react", LLMProfile: "codex_cli", SystemPrompt: "你负责消费上游产物。", Tools: []string{"file_read"}},
        },
        Stages: []model.StageConfig{
            {ID: "stage_a", Name: "Stage A", Role: "agent_a", Output: []model.StageIO{{Type: "artifact", Name: "artifact_a"}}},
            {ID: "stage_b", Name: "Stage B", Role: "agent_b", Input: []model.StageIO{{Type: "artifact", Ref: "artifact_a"}}, Output: []model.StageIO{{Type: "artifact", Name: "artifact_b"}}},
        },
        Transitions: []model.TransitionConfig{{From: "stage_a", To: "stage_b"}, {From: "stage_b", To: model.EndStage}},
    }
    result, err := session.Create(session.CreateOptions{Config: config, ConfigPath: configPath})
    if err != nil {
        t.Fatalf("create session: %v", err)
    }
    return result
}

func bootstrapArtifactStore(t *testing.T, sessionDir string) {
    t.Helper()
    if err := artifactstore.NewStore(sessionDir).Bootstrap(); err != nil {
        t.Fatalf("bootstrap artifact store: %v", err)
    }
}

func bootstrapEventBus(t *testing.T, sessionDir string, sessionID string) *eventbus.Store {
    t.Helper()
    store := eventbus.NewStore(sessionDir)
    if err := store.Bootstrap(eventbus.BootstrapOptions{SessionID: sessionID, SessionDir: sessionDir, WorkflowID: "artifact-orchestrator", MetadataRef: "metadata.md"}); err != nil {
        t.Fatalf("bootstrap bus: %v", err)
    }
    return store
}

func writeConclusionAndOutbox(t *testing.T, sessionDir string, roleID string, stageID string, taskID string, status string, outputRefs string) {
    t.Helper()
    roleDir := filepath.Join(sessionDir, "roles", roleID)
    if err := os.MkdirAll(roleDir, 0o755); err != nil {
        t.Fatalf("mkdir role dir: %v", err)
    }
    conclusion := "# Role Conclusion\n\n- role_id: " + roleID + "\n- stage_id: " + stageID + "\n- task_id: " + taskID + "\n- status: " + status + "\n- summary: done\n- output_refs: " + outputRefs + "\n- updated_at: 2026-03-08T09:00:00Z\n"
    if err := os.WriteFile(filepath.Join(roleDir, "conclusion.md"), []byte(conclusion), 0o644); err != nil {
        t.Fatalf("write conclusion: %v", err)
    }
    outbox := "# Role Outbox\n\n- outbox_version: 1\n- role_id: " + roleID + "\n- stage_id: " + stageID + "\n- task_id: " + taskID + "\n- turn_seq: 1\n- status: " + status + "\n- conclusion_ref: roles/" + roleID + "/conclusion.md\n- turn_output_ref: roles/" + roleID + "/turns/0001-output.md\n- updated_at: 2026-03-08T09:00:00Z\n"
    if err := os.MkdirAll(filepath.Join(roleDir, "turns"), 0o755); err != nil {
        t.Fatalf("mkdir turns: %v", err)
    }
    if err := os.WriteFile(filepath.Join(roleDir, "turns", "0001-output.md"), []byte("# output\n"), 0o644); err != nil {
        t.Fatalf("write turn output: %v", err)
    }
    if err := os.WriteFile(filepath.Join(roleDir, "outbox.md"), []byte(outbox), 0o644); err != nil {
        t.Fatalf("write outbox: %v", err)
    }
}
