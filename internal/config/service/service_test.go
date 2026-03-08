package service_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anner/openoctopus/internal/config/service"
)

func TestLoadForValidateMinimalConfigAppliesDefaults(t *testing.T) {
	t.Parallel()

	configPath := writeConfig(t, `
version: "2.1"

meta:
  workflow_id: "minimal-flow"
  name: "Minimal Flow"

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"

roles:
  - id: "agent_a"
    name: "Agent A"
    type: "react"
    llm_profile: "codex_cli"
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
`)

	result, err := service.LoadForValidate(service.LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadForValidate returned error: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no validation errors, got %d: %+v", len(result.Errors), result.Errors)
	}
	if result.Config.Runtime.Workspace.Root != ".octopus" {
		t.Fatalf("expected default workspace root .octopus, got %q", result.Config.Runtime.Workspace.Root)
	}
	if result.Config.Runtime.Tmux.Enabled {
		t.Fatal("expected tmux to be disabled by default")
	}
	if result.Config.Runtime.Tmux.SocketName != "octopus-{session_id}" {
		t.Fatalf("expected default tmux socket name, got %q", result.Config.Runtime.Tmux.SocketName)
	}
	if result.Config.Runtime.Tmux.MainPaneRatio != 0.5 {
		t.Fatalf("expected default main pane ratio 0.5, got %v", result.Config.Runtime.Tmux.MainPaneRatio)
	}
	if result.Config.Runtime.Tmux.RoleLayout != "adaptive_grid" {
		t.Fatalf("expected default tmux role layout adaptive_grid, got %q", result.Config.Runtime.Tmux.RoleLayout)
	}
	if len(result.AppliedDefaults) == 0 {
		t.Fatal("expected applied defaults to be recorded")
	}
}

func TestLoadForValidateReportsMissingStageRoleReference(t *testing.T) {
	t.Parallel()

	configPath := writeConfig(t, `
version: "2.1"

meta:
  workflow_id: "invalid-ref"
  name: "Invalid Ref"

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"

roles:
  - id: "agent_a"
    name: "Agent A"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责执行任务。"
    tools: ["file_read"]

stages:
  - id: "stage_a"
    name: "Stage A"
    role: "missing_role"
    output:
      - type: "artifact"
        name: "artifact_a"

transitions:
  - from: "stage_a"
    to: "__END__"
`)

	result, err := service.LoadForValidate(service.LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadForValidate returned error: %v", err)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 validation error, got %d: %+v", len(result.Errors), result.Errors)
	}
	validationError := result.Errors[0]
	if validationError.RuleID != "STAGE-004" {
		t.Fatalf("expected rule STAGE-004, got %q", validationError.RuleID)
	}
	if validationError.Path != "stages[0].role" {
		t.Fatalf("expected path stages[0].role, got %q", validationError.Path)
	}
}

func TestLoadForValidateSupportsEnvironmentOverride(t *testing.T) {
	t.Parallel()

	configPath := writeConfig(t, `
version: "2.1"

meta:
  workflow_id: "env-override"
  name: "Env Override"

runtime:
  scheduler:
    max_parallel_roles: 0

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"

roles:
  - id: "agent_a"
    name: "Agent A"
    type: "react"
    llm_profile: "codex_cli"
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
`)

	result, err := service.LoadForValidate(service.LoadOptions{
		ConfigPath: configPath,
		Environment: map[string]string{
			"OPENOCTOPUS_RUNTIME__SCHEDULER__MAX_PARALLEL_ROLES": "2",
		},
	})
	if err != nil {
		t.Fatalf("LoadForValidate returned error: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected env override to fix config, got errors: %+v", result.Errors)
	}
	if result.Config.Runtime.Scheduler.MaxParallelRoles != 2 {
		t.Fatalf("expected env override to set max_parallel_roles to 2, got %d", result.Config.Runtime.Scheduler.MaxParallelRoles)
	}
}

func TestLoadForValidateParsesLLMProfileTmuxCommand(t *testing.T) {
	t.Parallel()

	configPath := writeConfig(t, `
version: "2.1"

meta:
  workflow_id: "tmux-command"
  name: "Tmux Command"

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"
    tmux_command: "codex --dangerously-bypass-approvals-and-sandbox"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"

roles:
  - id: "agent_a"
    name: "Agent A"
    type: "react"
    llm_profile: "codex_cli"
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
`)

	result, err := service.LoadForValidate(service.LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadForValidate returned error: %v", err)
	}
	if result.Config.LLMProfiles["codex_cli"].TmuxCommand != "codex --dangerously-bypass-approvals-and-sandbox" {
		t.Fatalf("expected tmux_command to be parsed, got %q", result.Config.LLMProfiles["codex_cli"].TmuxCommand)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "octopus.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}
