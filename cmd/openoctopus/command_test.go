package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCommandFailsForInvalidConfig(t *testing.T) {
	t.Parallel()

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "invalid"
  name: "Invalid"

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

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"validate", "--config", configPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected validate command to fail for invalid config")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("stages[0].role")) {
		t.Fatalf("expected stderr to contain failing path, got %q", stderr.String())
	}
}

func TestRunCommandDoesNotCreateSessionForInvalidConfig(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	configPath := filepath.Join(workingDir, "octopus.yaml")
	if err := os.WriteFile(configPath, []byte(`
version: "2.1"

meta:
  workflow_id: "invalid-run"
  name: "Invalid Run"

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
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected run command to fail for invalid config")
	}
	if _, statErr := os.Stat(filepath.Join(workingDir, ".octopus", "sessions")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no sessions directory to be created, stat err=%v", statErr)
	}
}

func writeCommandConfig(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "octopus.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}
