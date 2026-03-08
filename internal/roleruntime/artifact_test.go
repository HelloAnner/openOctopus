package roleruntime

import (
    "strings"
    "testing"

    "github.com/anner/openoctopus/internal/config/model"
)

func TestResolveExecutorSupportsCodexCLI(t *testing.T) {
    resolved, err := resolveExecutor(model.LLMProfile{Provider: "codex", Mode: "cli", CLIPath: "codex"})
    if err != nil {
        t.Fatalf("expected codex executor to resolve: %v", err)
    }
    if resolved == nil {
        t.Fatal("expected non-nil codex executor")
    }
}

func TestBuildExecuteRequestIncludesArtifactOutputContract(t *testing.T) {
    config := model.RuntimeConfig{
        Stages: []model.StageConfig{{
            ID:   "stage_a",
            Name: "Stage A",
            Role: "agent_a",
            Output: []model.StageIO{{Type: "artifact", Name: "artifact_a"}},
        }},
    }
    request := buildExecuteRequest(
        "/tmp/session",
        config,
        model.RoleConfig{ID: "agent_a", SystemPrompt: "你负责执行任务。"},
        model.LLMProfile{Provider: "codex", Mode: "cli", CLIPath: "codex"},
        roleContext{ContextVersion: 1, TaskID: "task-stage_a-01", StageID: "stage_a", RoleID: "agent_a"},
        roleInbox{InboxVersion: 1, TaskID: "task-stage_a-01", StageID: "stage_a", RoleID: "agent_a", Status: "DISPATCHED", DispatchEventID: "event-000002", ContextVersion: 1},
        1,
    )

    if !strings.Contains(request.Prompt, "roles/agent_a/context.md") {
        t.Fatalf("expected prompt to mention context file, got %q", request.Prompt)
    }
    if !strings.Contains(request.Prompt, "roles/agent_a/inbox.md") {
        t.Fatalf("expected prompt to mention inbox file, got %q", request.Prompt)
    }
    if !strings.Contains(request.Prompt, "artifacts/_staging/stage_a/artifact_a.md") {
        t.Fatalf("expected prompt to mention suggested artifact ref, got %q", request.Prompt)
    }
    if !strings.Contains(request.Prompt, "## role_result") {
        t.Fatalf("expected prompt to constrain final block, got %q", request.Prompt)
    }
}

func TestDeterministicExecutorUsesConfiguredOutputRefs(t *testing.T) {
    t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")
    t.Setenv("OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS_AGENT_A", "artifacts/_staging/stage_a/artifact_a.md")

    result, err := deterministicExecutor{}.Execute(ExecuteRequest{Role: model.RoleConfig{ID: "agent_a"}, TurnSeq: 1})
    if err != nil {
        t.Fatalf("execute deterministic: %v", err)
    }
    if !strings.Contains(result.RawOutput, "output_refs: artifacts/_staging/stage_a/artifact_a.md") {
        t.Fatalf("expected output refs in raw output, got %q", result.RawOutput)
    }
}
