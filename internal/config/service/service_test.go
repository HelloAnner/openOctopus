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

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "octopus.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}
