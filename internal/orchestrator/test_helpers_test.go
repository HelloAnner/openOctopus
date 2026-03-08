package orchestrator

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/session"
)

func useFixedOrchestratorClock(t *testing.T) {
	t.Helper()
	fixed := time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC)
	nowFunc = func() time.Time {
		return fixed
	}
	t.Cleanup(func() {
		nowFunc = time.Now
	})
}

func createOrchestratorTestSession(t *testing.T) session.CreateResult {
	t.Helper()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "octopus.yaml")
	config := model.RuntimeConfig{
		Version: model.SupportedConfigVersion,
		Meta: model.MetaConfig{
			WorkflowID: "orchestrator-workflow",
			Name:       "Orchestrator Workflow",
		},
		Runtime: model.RuntimeSection{
			Workspace: model.WorkspaceConfig{
				Root:        ".octopus",
				SessionsDir: ".octopus/sessions",
			},
			Scheduler: model.SchedulerConfig{
				MaxParallelRoles: 1,
			},
			MasterWatch: model.MasterWatchConfig{
				Enabled:             true,
				ProgressFile:        "planner/global_progress.md",
				BlockerFile:         "planner/blockers.md",
				MaxNoProgressRounds: 3,
			},
		},
		Policies: model.PoliciesConfig{
			Retry:     model.RetryPolicy{MaxRetryPerStage: 2},
			LoopGuard: model.LoopGuardPolicy{MaxRoundsPerTask: 6},
		},
		Roles: []model.RoleConfig{
			{
				ID:           "agent_a",
				Name:         "Agent A",
				Type:         "react",
				LLMProfile:   "codex_cli",
				SystemPrompt: "你负责执行任务。",
				Tools:        []string{"file_read"},
			},
		},
		Stages: []model.StageConfig{
			{
				ID:     "stage_a",
				Name:   "Stage A",
				Role:   "agent_a",
				Output: []model.StageIO{{Type: "artifact", Name: "artifact_a"}},
			},
		},
		Transitions: []model.TransitionConfig{{
			From: "stage_a",
			To:   model.EndStage,
		}},
	}
	result, err := session.Create(session.CreateOptions{Config: config, ConfigPath: configPath})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return result
}
