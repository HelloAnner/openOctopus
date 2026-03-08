package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/tmux"
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

func TestRunCommandCreatesSessionSkeleton(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "valid-run"
  name: "Valid Run"

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

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	err := command.Execute()
	if err != nil {
		t.Fatalf("expected run command to succeed: %v, stderr=%q", err, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("session created:")) {
		t.Fatalf("expected stdout to contain session path, got %q", stdout.String())
	}
	sessionsDir := filepath.Join(filepath.Dir(configPath), ".octopus", "sessions")
	entries, readErr := os.ReadDir(sessionsDir)
	if readErr != nil {
		t.Fatalf("read sessions dir: %v", readErr)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one session dir, got %d", len(entries))
	}
	sessionDir := filepath.Join(sessionsDir, entries[0].Name())
	requiredFiles := []string{
		filepath.Join(sessionDir, "metadata.md"),
		filepath.Join(sessionDir, "session.state.md"),
		filepath.Join(sessionDir, "timeline.md"),
		filepath.Join(sessionDir, "state", "effective_config.yaml"),
		filepath.Join(sessionDir, "state", "checkpoints", "0000-init.md"),
	}
	for _, item := range requiredFiles {
		if _, statErr := os.Stat(item); statErr != nil {
			t.Fatalf("expected %q to exist: %v", item, statErr)
		}
	}
	metadata, readErr := os.ReadFile(filepath.Join(sessionDir, "metadata.md"))
	if readErr != nil {
		t.Fatalf("read metadata: %v", readErr)
	}
	if !bytes.Contains(metadata, []byte("- applied_defaults_count: 1")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 2")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 3")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 4")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 5")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 6")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 7")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 8")) && !bytes.Contains(metadata, []byte("- applied_defaults_count: 9")) {
		t.Fatalf("expected metadata to record non-zero applied defaults count, got %q", string(metadata))
	}
}

func TestRunCommandBootstrapsArtifactStore(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "valid-run-artifact"
  name: "Valid Run Artifact"

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

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	err := command.Execute()
	if err != nil {
		t.Fatalf("expected run command to succeed: %v, stderr=%q", err, stderr.String())
	}

	sessionsDir := filepath.Join(filepath.Dir(configPath), ".octopus", "sessions")
	entries, readErr := os.ReadDir(sessionsDir)
	if readErr != nil {
		t.Fatalf("read sessions dir: %v", readErr)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one session dir, got %d", len(entries))
	}
	sessionDir := filepath.Join(sessionsDir, entries[0].Name())

	indexFile, readErr := os.ReadFile(filepath.Join(sessionDir, "artifacts", "index.md"))
	if readErr != nil {
		t.Fatalf("read artifact index: %v", readErr)
	}
	if bytes.Contains(indexFile, []byte("Initialized by session 001.")) || !bytes.Contains(indexFile, []byte("# Artifact Index")) {
		t.Fatalf("expected bootstrapped artifact index, got %q", string(indexFile))
	}
}

func TestRunCommandBootstrapsEventBus(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "valid-run-eventbus"
  name: "Valid Run Event Bus"

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

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	err := command.Execute()
	if err != nil {
		t.Fatalf("expected run command to succeed: %v, stderr=%q", err, stderr.String())
	}

	sessionsDir := filepath.Join(filepath.Dir(configPath), ".octopus", "sessions")
	entries, readErr := os.ReadDir(sessionsDir)
	if readErr != nil {
		t.Fatalf("read sessions dir: %v", readErr)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one session dir, got %d", len(entries))
	}
	sessionDir := filepath.Join(sessionsDir, entries[0].Name())

	eventsFile, readErr := os.ReadFile(filepath.Join(sessionDir, "bus", "events.md"))
	if readErr != nil {
		t.Fatalf("read bus events: %v", readErr)
	}
	if !bytes.Contains(eventsFile, []byte("SESSION_CREATED")) || !bytes.Contains(eventsFile, []byte("event-000001")) {
		t.Fatalf("expected bootstrap event in bus/events.md, got %q", string(eventsFile))
	}

	lockFile, readErr := os.ReadFile(filepath.Join(sessionDir, "bus", "lock.md"))
	if readErr != nil {
		t.Fatalf("read lock file: %v", readErr)
	}
	if bytes.Contains(lockFile, []byte("Initialized by session 001.")) || !bytes.Contains(lockFile, []byte("- status: FREE")) {
		t.Fatalf("expected initialized bus lock file, got %q", string(lockFile))
	}
}

func TestRunCommandDoesNotCreateSessionForInvalidConfig(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")

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

func TestRunCommandBootstrapsOrchestrator(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "valid-run-orchestrator"
  name: "Valid Run Orchestrator"

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

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected run command to succeed: %v, stderr=%q", err, stderr.String())
	}

	sessionsDir := filepath.Join(filepath.Dir(configPath), ".octopus", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("read sessions dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one session dir, got %d", len(entries))
	}
	sessionDir := filepath.Join(sessionsDir, entries[0].Name())

	scheduleContent, err := os.ReadFile(filepath.Join(sessionDir, "planner", "master_schedule.md"))
	if err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	if bytes.Contains(scheduleContent, []byte("Initialized by session 001.")) || !bytes.Contains(scheduleContent, []byte("stage_id: stage_a")) {
		t.Fatalf("expected orchestrator schedule to be bootstrapped, got %q", string(scheduleContent))
	}

	contextContent, err := os.ReadFile(filepath.Join(sessionDir, "roles", "agent_a", "context.md"))
	if err != nil {
		t.Fatalf("read context: %v", err)
	}
	if !bytes.Contains(contextContent, []byte("task_id: task-stage_a-01")) {
		t.Fatalf("expected role context to be dispatched, got %q", string(contextContent))
	}
}

func TestRunCommandCompletesWorkflowWithDeterministicRoleRuntime(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "run-deterministic-success"
  name: "Run Deterministic Success"

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
`)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected run command to succeed: %v, stderr=%q", err, stderr.String())
	}

	sessionsDir := filepath.Join(filepath.Dir(configPath), ".octopus", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("read sessions dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one session dir, got %d", len(entries))
	}
	sessionDir := filepath.Join(sessionsDir, entries[0].Name())

	turnsDir := filepath.Join(sessionDir, "roles", "agent_a", "turns")
	if _, err := os.Stat(filepath.Join(turnsDir, "0001-input.md")); err != nil {
		t.Fatalf("expected turn input file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(turnsDir, "0001-output.md")); err != nil {
		t.Fatalf("expected turn output file: %v", err)
	}
	conclusionContent, err := os.ReadFile(filepath.Join(sessionDir, "roles", "agent_a", "conclusion.md"))
	if err != nil {
		t.Fatalf("read conclusion: %v", err)
	}
	if !bytes.Contains(conclusionContent, []byte("- status: SUCCESS")) {
		t.Fatalf("expected successful conclusion, got %q", string(conclusionContent))
	}
	stateContent, err := os.ReadFile(filepath.Join(sessionDir, "session.state.md"))
	if err != nil {
		t.Fatalf("read session state: %v", err)
	}
	if !bytes.Contains(stateContent, []byte("- status: COMPLETED")) {
		t.Fatalf("expected completed session state, got %q", string(stateContent))
	}
}

func TestRunCommandRetriesDeterministicRoleRuntime(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "NEEDS_RETRY,SUCCESS")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "run-deterministic-retry"
  name: "Run Deterministic Retry"

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
`)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected run command to succeed: %v, stderr=%q", err, stderr.String())
	}

	sessionsDir := filepath.Join(filepath.Dir(configPath), ".octopus", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("read sessions dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one session dir, got %d", len(entries))
	}
	sessionDir := filepath.Join(sessionsDir, entries[0].Name())

	for _, name := range []string{"0001-input.md", "0001-output.md", "0002-input.md", "0002-output.md"} {
		if _, err := os.Stat(filepath.Join(sessionDir, "roles", "agent_a", "turns", name)); err != nil {
			t.Fatalf("expected retry turn file %s: %v", name, err)
		}
	}
	conclusionContent, err := os.ReadFile(filepath.Join(sessionDir, "roles", "agent_a", "conclusion.md"))
	if err != nil {
		t.Fatalf("read conclusion: %v", err)
	}
	if !bytes.Contains(conclusionContent, []byte("- status: SUCCESS")) {
		t.Fatalf("expected successful conclusion after retry, got %q", string(conclusionContent))
	}
}

func TestValidateCommandWritesJSONSuccess(t *testing.T) {
	configPath := writeCommandConfig(t, validCodexCommandConfig("json-validate-success", "JSON Validate Success"))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"validate", "--config", configPath, "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("validate command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeJSONPayload(t, stdout.Bytes())
	assertJSONFieldString(t, payload, "command", "validate")
	assertJSONFieldBool(t, payload, "ok", true)
	data := payload["data"].(map[string]any)
	if int(data["applied_defaults_count"].(float64)) <= 0 {
		t.Fatalf("expected applied defaults count > 0, got %#v", data["applied_defaults_count"])
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestValidateCommandWritesJSONFailure(t *testing.T) {
	configPath := writeCommandConfig(t, invalidCommandConfig("json-validate-failure", "JSON Validate Failure"))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"validate", "--config", configPath, "--format", "json"})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected validate command to fail")
	}
	payload := decodeJSONPayload(t, stderr.Bytes())
	assertJSONFieldString(t, payload, "command", "validate")
	assertJSONFieldBool(t, payload, "ok", false)
	errorBody := payload["error"].(map[string]any)
	if errorBody["code"] != "config_validation_failed" {
		t.Fatalf("expected config_validation_failed, got %#v", errorBody["code"])
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
}

func TestRunCommandWritesJSONSuccess(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeCommandConfig(t, validCodexCommandConfig("json-run-success", "JSON Run Success"))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath, "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("run command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeJSONPayload(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["session_id"] == "" {
		t.Fatalf("expected session_id, got %#v", data["session_id"])
	}
	if data["session_dir"] == "" {
		t.Fatalf("expected session_dir, got %#v", data["session_dir"])
	}
}

func TestRunCommandCompletesRepeatedReviewLoop(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_PRD_SPLITTER", "SUCCESS,SUCCESS,SUCCESS")
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_PRD_REVIEWER", "SUCCESS,SUCCESS,SUCCESS")
	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "repeat-prd-loop"
  name: "Repeat PRD Loop"

runtime:
  scheduler:
    max_parallel_roles: 1

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
  - id: "prd_splitter"
    name: "PRD Splitter"
    type: "react"
    llm_profile: "deterministic_cli"
    system_prompt: "你负责拆分 PRD 需求文档。"
    tools: ["file_read"]

  - id: "prd_reviewer"
    name: "PRD Reviewer"
    type: "react"
    llm_profile: "deterministic_cli"
    system_prompt: "你负责 review 并给出意见。"
    tools: ["file_read"]

stages:
  - id: "split_prd"
    name: "拆分 PRD"
    role: "prd_splitter"
    output:
      - type: "artifact"
        name: "split_prd_doc"

  - id: "review_prd"
    name: "Review PRD"
    role: "prd_reviewer"
    input:
      - type: "artifact"
        ref: "split_prd_doc"
    output:
      - type: "artifact"
        name: "review_feedback"

transitions:
  - from: "split_prd"
    to: "review_prd"
  - from: "review_prd"
    to: "split_prd"
    repeat:
      max_rounds: 3
      on_complete: "__END__"
`)

	sessionDir := runCommandAndParseSessionDir(t, []string{"run", "--config", configPath, "--format", "json"})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"status", "--session", sessionDir, "--format", "json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("status command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeJSONPayload(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["workflow_status"] != "COMPLETED" {
		t.Fatalf("expected workflow_status COMPLETED, got %#v", data["workflow_status"])
	}
	scheduleContent, err := os.ReadFile(filepath.Join(sessionDir, "planner", "master_schedule.md"))
	if err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	text := string(scheduleContent)
	if !strings.Contains(text, "split_prd__round_03") || !strings.Contains(text, "review_prd__round_03") {
		t.Fatalf("expected expanded round stages in schedule, got %q", text)
	}
}

func TestRunCommandBootstrapsTmuxLayout(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "valid-run-tmux"
  name: "Valid Run Tmux"

runtime:
  tmux:
    enabled: true
    socket_name: "octopus-{session_id}"
    main_pane_ratio: 0.5
    role_layout: "adaptive_grid"

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
  - id: "agent_b"
    name: "Agent B"
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
  - id: "stage_b"
    name: "Stage B"
    role: "agent_b"
    output:
      - type: "artifact"
        name: "artifact_b"

transitions:
  - from: "stage_a"
    to: "stage_b"
  - from: "stage_b"
    to: "__END__"
`)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected run command to succeed: %v, stderr=%q", err, stderr.String())
	}

	sessionDir := mustFindSingleSessionDir(t, configPath)
	if _, err := os.Stat(filepath.Join(sessionDir, "state", "tmux", "layout.md")); err != nil {
		t.Fatalf("expected tmux layout to exist: %v", err)
	}
	layout, err := tmux.ReadLayout(sessionDir)
	if err != nil {
		t.Fatalf("ReadLayout returned error: %v", err)
	}
	cleanupTmuxSession(t, layout.SocketName, layout.SessionName)
	if layout.RolePanes["agent_a"].PaneID == "" {
		t.Fatal("expected agent_a pane binding")
	}
	if layout.RolePanes["agent_b"].PaneID == "" {
		t.Fatal("expected agent_b pane binding")
	}
}

func TestSwitchCommandWritesJSONTarget(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")

	configPath := writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "valid-switch-tmux"
  name: "Valid Switch Tmux"

runtime:
  tmux:
    enabled: true

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

	runStdout := &bytes.Buffer{}
	runStderr := &bytes.Buffer{}
	runCommand := NewRootCommand()
	runCommand.SetOut(runStdout)
	runCommand.SetErr(runStderr)
	runCommand.SetArgs([]string{"run", "--config", configPath})
	if err := runCommand.Execute(); err != nil {
		t.Fatalf("expected run to succeed: %v, stderr=%q", err, runStderr.String())
	}

	sessionDir := mustFindSingleSessionDir(t, configPath)
	layout, err := tmux.ReadLayout(sessionDir)
	if err != nil {
		t.Fatalf("ReadLayout returned error: %v", err)
	}
	defer cleanupTmuxSession(t, layout.SocketName, layout.SessionName)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"switch", "--session", sessionDir, "--role", "agent_a", "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected switch to succeed: %v, stderr=%q", err, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal switch payload: %v", err)
	}
	data := payload["data"].(map[string]any)
	if data["target_role"] != "agent_a" {
		t.Fatalf("expected target role agent_a, got %#v", data["target_role"])
	}
	if data["target_pane_id"] == "" {
		t.Fatal("expected target pane id")
	}
	if data["switched"] != false {
		t.Fatalf("expected switched false outside tmux client, got %#v", data["switched"])
	}
}

func TestStatusCommandWritesJSONSummary(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")
	configPath := writeCommandConfig(t, validDeterministicCommandConfig("json-status-success", "JSON Status Success"))
	sessionDir := runCommandAndParseSessionDir(t, []string{"run", "--config", configPath, "--format", "json"})

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"status", "--session", sessionDir, "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("status command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeJSONPayload(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["workflow_status"] != "COMPLETED" {
		t.Fatalf("expected workflow_status COMPLETED, got %#v", data["workflow_status"])
	}
	if data["current_stage_id"] != "stage_a" {
		t.Fatalf("expected current_stage_id stage_a, got %#v", data["current_stage_id"])
	}
}

func TestStatusCommandFailsForMissingSession(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"status", "--session", "missing", "--format", "json"})

	err := command.Execute()
	if err == nil {
		t.Fatal("expected status command to fail")
	}
	payload := decodeJSONPayload(t, stderr.Bytes())
	errorBody := payload["error"].(map[string]any)
	if errorBody["code"] != "session_not_found" {
		t.Fatalf("expected session_not_found, got %#v", errorBody["code"])
	}
}

func TestExecuteMapsExitCodes(t *testing.T) {
	invalidConfigPath := writeCommandConfig(t, invalidCommandConfig("execute-invalid", "Execute Invalid"))
	if code := execute([]string{"validate", "--config", invalidConfigPath, "--format", "json"}, &bytes.Buffer{}, &bytes.Buffer{}); code != exitCodeConfigValidationFailed {
		t.Fatalf("expected validate exit code %d, got %d", exitCodeConfigValidationFailed, code)
	}
	if code := execute([]string{"status", "--session", "missing", "--format", "json"}, &bytes.Buffer{}, &bytes.Buffer{}); code != exitCodeSessionNotFound {
		t.Fatalf("expected status exit code %d, got %d", exitCodeSessionNotFound, code)
	}
	if code := execute([]string{"recover", "--session", "missing", "--format", "json"}, &bytes.Buffer{}, &bytes.Buffer{}); code != exitCodeSessionNotFound {
		t.Fatalf("expected recover exit code %d, got %d", exitCodeSessionNotFound, code)
	}
}

func TestExecuteTreatsYAMLPathAsRunAlias(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeCommandConfig(t, validDeterministicCommandConfig("execute-alias", "Execute Alias"))
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := execute([]string{configPath, "--format", "json"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("expected yaml alias to succeed, got code %d, stderr=%q", code, stderr.String())
	}
	payload := decodeJSONPayload(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["session_dir"] == "" {
		t.Fatalf("expected session_dir in payload, got %#v", data["session_dir"])
	}
	if _, err := os.Stat(data["session_dir"].(string)); err != nil {
		t.Fatalf("expected session_dir to exist: %v", err)
	}
}

func TestRecoverCommandWritesJSONSummary(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeCommandConfig(t, validDeterministicCommandConfig("json-recover-success", "JSON Recover Success"))
	sessionDir := runCommandAndParseSessionDir(t, []string{"run", "--config", configPath, "--format", "json"})
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "0")
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"recover", "--session", sessionDir, "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("recover command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeJSONPayload(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["continued"] != true {
		t.Fatalf("expected continued=true, got %#v", data["continued"])
	}
	if data["recovered_status"] != "COMPLETED" {
		t.Fatalf("expected recovered_status COMPLETED, got %#v", data["recovered_status"])
	}
	if data["checkpoint_ref"] == "" {
		t.Fatalf("expected checkpoint_ref, got %#v", data["checkpoint_ref"])
	}
	if data["replay_ref"] != "audit/replay.md" {
		t.Fatalf("expected replay_ref audit/replay.md, got %#v", data["replay_ref"])
	}
}

func TestRecoverCommandFailsForBrokenEventChain(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	sessionDir := runCommandAndParseSessionDir(t, []string{"run", "--config", writeCommandConfig(t, validDeterministicCommandConfig("json-recover-broken", "JSON Recover Broken")), "--format", "json"})
	eventsPath := filepath.Join(sessionDir, "bus", "events.md")
	content, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	rewritten := bytes.Replace(content, []byte("- event_hash: "), []byte("- event_hash: broken-"), 1)
	if err := os.WriteFile(eventsPath, rewritten, 0o644); err != nil {
		t.Fatalf("rewrite events: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"recover", "--session", sessionDir, "--format", "json"})

	err = command.Execute()
	if err == nil {
		t.Fatal("expected recover command to fail")
	}
	payload := decodeJSONPayload(t, stderr.Bytes())
	errorBody := payload["error"].(map[string]any)
	if errorBody["code"] != "event_chain_broken" {
		t.Fatalf("expected event_chain_broken, got %#v", errorBody["code"])
	}
}

func runCommandAndParseSessionDir(t *testing.T, args []string) string {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("command %v failed: %v, stderr=%q", args, err, stderr.String())
	}
	payload := decodeJSONPayload(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	return data["session_dir"].(string)
}

func decodeJSONPayload(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("invalid json %q: %v", string(content), err)
	}
	return payload
}

func assertJSONFieldString(t *testing.T, payload map[string]any, key string, expected string) {
	t.Helper()
	if payload[key] != expected {
		t.Fatalf("expected %s=%q, got %#v", key, expected, payload[key])
	}
}

func assertJSONFieldBool(t *testing.T, payload map[string]any, key string, expected bool) {
	t.Helper()
	if payload[key] != expected {
		t.Fatalf("expected %s=%t, got %#v", key, expected, payload[key])
	}
}

func validCodexCommandConfig(workflowID string, workflowName string) string {
	return `
version: "2.1"

meta:
  workflow_id: "` + workflowID + `"
  name: "` + workflowName + `"

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
`
}

func validDeterministicCommandConfig(workflowID string, workflowName string) string {
	return `
version: "2.1"

meta:
  workflow_id: "` + workflowID + `"
  name: "` + workflowName + `"

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
}

func invalidCommandConfig(workflowID string, workflowName string) string {
	return `
version: "2.1"

meta:
  workflow_id: "` + workflowID + `"
  name: "` + workflowName + `"

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
`
}

func mustFindSingleSessionDir(t *testing.T, configPath string) string {
	t.Helper()
	sessionsDir := filepath.Join(filepath.Dir(configPath), ".octopus", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		t.Fatalf("read sessions dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one session dir, got %d", len(entries))
	}
	return filepath.Join(sessionsDir, entries[0].Name())
}

func cleanupTmuxSession(t *testing.T, socketName string, sessionName string) {
	t.Helper()
	service := tmux.NewService("")
	if err := service.KillSession(socketName, sessionName); err != nil {
		t.Fatalf("kill tmux session: %v", err)
	}
}
