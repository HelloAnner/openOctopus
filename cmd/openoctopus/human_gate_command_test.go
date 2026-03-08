package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anner/openoctopus/internal/roleruntime"
)

func TestInterruptCommandWritesInterruptRecord(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeHumanGateCommandConfig(t, false)
	sessionDir := runSessionForHumanGate(t, configPath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"interrupt", "--session", sessionDir, "--role", "agent_a", "--reason", "manual review"})

	if err := command.Execute(); err != nil {
		t.Fatalf("interrupt command failed: %v, stderr=%q", err, stderr.String())
	}

	interrupts, err := os.ReadFile(filepath.Join(sessionDir, "bus", "interrupts.md"))
	if err != nil {
		t.Fatalf("read interrupts: %v", err)
	}
	assertCommandContains(t, string(interrupts), "target_role_id: agent_a")
	assertCommandContains(t, string(interrupts), "status: REQUESTED")
}

func TestInjectCommandAppendsHumanMessageFromFile(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeHumanGateCommandConfig(t, false)
	sessionDir := runSessionForHumanGate(t, configPath)
	notePath := filepath.Join(filepath.Dir(configPath), "note.md")
	if err := os.WriteFile(notePath, []byte("请继续，但先补测试。\n"), 0o644); err != nil {
		t.Fatalf("write note file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"inject", "--session", sessionDir, "--role", "agent_a", "--input", notePath})

	if err := command.Execute(); err != nil {
		t.Fatalf("inject command failed: %v, stderr=%q", err, stderr.String())
	}

	messages, err := os.ReadFile(filepath.Join(sessionDir, "planner", "human_messages.md"))
	if err != nil {
		t.Fatalf("read human messages: %v", err)
	}
	assertCommandContains(t, string(messages), "source: human-gate")
	assertCommandContains(t, string(messages), "target_role_id: agent_a")
	assertCommandContains(t, string(messages), "请继续，但先补测试。")
}

func TestInterruptAllCommandMarksWaitingHuman(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeHumanGateCommandConfig(t, true)
	sessionDir := runSessionForHumanGate(t, configPath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"interrupt-all", "--session", sessionDir, "--reason", "manual review"})

	if err := command.Execute(); err != nil {
		t.Fatalf("interrupt-all command failed: %v, stderr=%q", err, stderr.String())
	}

	state, err := os.ReadFile(filepath.Join(sessionDir, "session.state.md"))
	if err != nil {
		t.Fatalf("read session state: %v", err)
	}
	assertCommandContains(t, string(state), "status: WAITING_HUMAN")

	blockers, err := os.ReadFile(filepath.Join(sessionDir, "planner", "blockers.md"))
	if err != nil {
		t.Fatalf("read blockers: %v", err)
	}
	assertCommandContains(t, string(blockers), "manual review")

	interrupts, err := os.ReadFile(filepath.Join(sessionDir, "bus", "interrupts.md"))
	if err != nil {
		t.Fatalf("read interrupts: %v", err)
	}
	if strings.Count(string(interrupts), "status: REQUESTED") < 2 {
		t.Fatalf("expected at least two interrupt records, got %q", string(interrupts))
	}
}

func TestResumeCommandClearsInterruptAndCompletesWorkflow(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")
	configPath := writeHumanGateCommandConfig(t, false)
	sessionDir := runSessionForHumanGate(t, configPath)

	executeHumanGateCommand(t, "interrupt", "--session", sessionDir, "--role", "agent_a", "--reason", "manual review")
	if _, err := roleruntime.NewEngine(sessionDir).TickRole("agent_a"); err != nil {
		t.Fatalf("tick role for interrupt ack: %v", err)
	}
	executeHumanGateCommand(t, "inject", "--session", sessionDir, "--role", "agent_a", "--message", "继续执行")
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "0")
	executeHumanGateCommand(t, "resume", "--session", sessionDir, "--role", "agent_a")

	interrupts, err := os.ReadFile(filepath.Join(sessionDir, "bus", "interrupts.md"))
	if err != nil {
		t.Fatalf("read interrupts: %v", err)
	}
	assertCommandContains(t, string(interrupts), "status: CLEARED")

	state, err := os.ReadFile(filepath.Join(sessionDir, "session.state.md"))
	if err != nil {
		t.Fatalf("read session state: %v", err)
	}
	assertCommandContains(t, string(state), "status: COMPLETED")
}

func TestInterruptCommandWritesJSONResponse(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeHumanGateCommandConfig(t, false)
	sessionDir := runSessionForHumanGate(t, configPath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"interrupt", "--session", sessionDir, "--role", "agent_a", "--reason", "manual review", "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("interrupt command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeHumanGateJSON(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["interrupt_id"] == "" {
		t.Fatalf("expected interrupt_id, got %#v", data["interrupt_id"])
	}
}

func TestInjectCommandWritesJSONResponse(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeHumanGateCommandConfig(t, false)
	sessionDir := runSessionForHumanGate(t, configPath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"inject", "--session", sessionDir, "--role", "agent_a", "--message", "继续执行", "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("inject command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeHumanGateJSON(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["message_id"] != "msg-000001" {
		t.Fatalf("expected message_id msg-000001, got %#v", data["message_id"])
	}
}

func TestInterruptAllCommandWritesJSONResponse(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	configPath := writeHumanGateCommandConfig(t, true)
	sessionDir := runSessionForHumanGate(t, configPath)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"interrupt-all", "--session", sessionDir, "--reason", "manual review", "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("interrupt-all command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeHumanGateJSON(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if int(data["requested_count"].(float64)) < 2 {
		t.Fatalf("expected requested_count >= 2, got %#v", data["requested_count"])
	}
}

func TestResumeCommandWritesJSONResponse(t *testing.T) {
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "1")
	t.Setenv("OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A", "SUCCESS")
	configPath := writeHumanGateCommandConfig(t, false)
	sessionDir := runSessionForHumanGate(t, configPath)

	executeHumanGateCommand(t, "interrupt", "--session", sessionDir, "--role", "agent_a", "--reason", "manual review")
	if _, err := roleruntime.NewEngine(sessionDir).TickRole("agent_a"); err != nil {
		t.Fatalf("tick role for interrupt ack: %v", err)
	}
	executeHumanGateCommand(t, "inject", "--session", sessionDir, "--role", "agent_a", "--message", "继续执行")
	t.Setenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP", "0")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"resume", "--session", sessionDir, "--role", "agent_a", "--format", "json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("resume command failed: %v, stderr=%q", err, stderr.String())
	}
	payload := decodeHumanGateJSON(t, stdout.Bytes())
	data := payload["data"].(map[string]any)
	if data["session_dir"] != sessionDir {
		t.Fatalf("expected session_dir %q, got %#v", sessionDir, data["session_dir"])
	}
}

func writeHumanGateCommandConfig(t *testing.T, twoRoles bool) string {
	t.Helper()
	if !twoRoles {
		return writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "human-gate-command"
  name: "Human Gate Command"

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
	}
	return writeCommandConfig(t, `
version: "2.1"

meta:
  workflow_id: "human-gate-command-multi"
  name: "Human Gate Command Multi"

runtime:
  scheduler:
    max_parallel_roles: 2

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
  - id: "agent_b"
    name: "Agent B"
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
  - id: "stage_b"
    name: "Stage B"
    role: "agent_b"
    output:
      - type: "artifact"
        name: "artifact_b"

transitions:
  - from: "stage_a"
    to: "__END__"
  - from: "stage_b"
    to: "__END__"
`)
}

func runSessionForHumanGate(t *testing.T, configPath string) string {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	command := NewRootCommand()
	command.SetOut(stdout)
	command.SetErr(stderr)
	command.SetArgs([]string{"run", "--config", configPath})
	if err := command.Execute(); err != nil {
		t.Fatalf("run command failed: %v, stderr=%q", err, stderr.String())
	}
	prefix := "session created: "
	for _, line := range strings.Split(stdout.String(), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	t.Fatalf("session dir not found in stdout: %q", stdout.String())
	return ""
}

func executeHumanGateCommand(t *testing.T, args ...string) {
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
}

func assertCommandContains(t *testing.T, content string, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected content to contain %q, got %q", expected, content)
	}
}

func decodeHumanGateJSON(t *testing.T, content []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(content, &payload); err != nil {
		t.Fatalf("invalid json %q: %v", string(content), err)
	}
	return payload
}
