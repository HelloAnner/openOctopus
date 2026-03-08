package validator

import (
	"testing"

	configerrors "github.com/anner/openoctopus/internal/config/errors"
	"github.com/anner/openoctopus/internal/config/model"
)

func TestValidateTmuxMainPaneRatio(t *testing.T) {
	t.Parallel()

	config := validTmuxConfig()
	config.Runtime.Tmux.MainPaneRatio = 0

	errors := Validate(config)
	assertHasValidationPath(t, errors, "runtime.tmux.main_pane_ratio")
}

func TestValidateTmuxRoleLayout(t *testing.T) {
	t.Parallel()

	config := validTmuxConfig()
	config.Runtime.Tmux.RoleLayout = "unknown"

	errors := Validate(config)
	assertHasValidationPath(t, errors, "runtime.tmux.role_layout")
}

func TestValidateLLMProfileTmuxCommandRequiresCLIMode(t *testing.T) {
	t.Parallel()

	config := validTmuxConfig()
	profile := config.LLMProfiles["deterministic"]
	profile.Mode = "api"
	profile.TmuxCommand = "demo --interactive"
	config.LLMProfiles["deterministic"] = profile

	errors := Validate(config)
	assertHasValidationPath(t, errors, "llm_profiles.deterministic.tmux_command")
}

func TestValidateRepeatTransitionRequiresOnCompleteAndPositiveRounds(t *testing.T) {
	t.Parallel()

	config := validTmuxConfig()
	config.Transitions = []model.TransitionConfig{
		{
			From: "stage_a",
			To:   "stage_a",
			Repeat: model.RepeatConfig{
				MaxRounds:  0,
				OnComplete: model.EndStage,
			},
		},
		{
			From: "stage_a",
			To:   model.EndStage,
			Repeat: model.RepeatConfig{
				MaxRounds:  2,
				OnComplete: "",
			},
		},
	}

	errors := Validate(config)
	assertHasValidationPath(t, errors, "transitions[0].repeat.max_rounds")
	assertHasValidationPath(t, errors, "transitions[1].repeat.on_complete")
}

func validTmuxConfig() model.RuntimeConfig {
	return model.RuntimeConfig{
		Version: model.SupportedConfigVersion,
		Meta:    model.MetaConfig{WorkflowID: "tmux-valid", Name: "tmux valid"},
		Runtime: model.RuntimeSection{
			Tmux:        model.TmuxConfig{Enabled: true, SocketName: "octopus-{session_id}", MainPaneRatio: 0.5, RoleLayout: "adaptive_grid"},
			Scheduler:   model.SchedulerConfig{MaxParallelRoles: 1},
			RoleRuntime: model.RoleRuntimeConfig{IdlePollSeconds: 2},
			MasterWatch: model.MasterWatchConfig{MaxNoProgressRounds: 1},
		},
		LLMProfiles: map[string]model.LLMProfile{
			"deterministic": {Provider: "deterministic", Mode: "cli", CLIPath: "deterministic"},
		},
		ToolRegistry: model.ToolRegistry{
			Builtin: map[string]model.BuiltinToolConfig{
				"file_read": {Module: "openoctopus.tools.file", Class: "FileReadTool"},
			},
		},
		Roles:       []model.RoleConfig{{ID: "agent_a", Name: "Agent A", Type: "react", LLMProfile: "deterministic", SystemPrompt: "do it", Tools: []string{"file_read"}}},
		Stages:      []model.StageConfig{{ID: "stage_a", Name: "Stage A", Role: "agent_a", Output: []model.StageIO{{Type: "artifact", Name: "artifact_a"}}}},
		Transitions: []model.TransitionConfig{{From: "stage_a", To: model.EndStage}},
	}
}

func assertHasValidationPath(t *testing.T, errors []configerrors.ConfigError, path string) {
	t.Helper()
	for _, item := range errors {
		if item.Path == path {
			return
		}
	}
	t.Fatalf("expected validation error for %q, got %#v", path, errors)
}
