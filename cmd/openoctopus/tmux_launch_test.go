/*
Package main tmux_launch_test 验证 tmux agent pane 启动命令构造。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"path/filepath"
	"strings"
	"testing"

	configmodel "github.com/anner/openoctopus/internal/config/model"
)


func TestBuildTmuxLaunchPlanBuildsMainCodexCommand(t *testing.T) {
	config := configmodel.RuntimeConfig{
		Runtime: configmodel.RuntimeSection{Tmux: configmodel.TmuxConfig{MainLLMProfile: "codex_cli"}},
		LLMProfiles: map[string]configmodel.LLMProfile{
			"codex_cli": {Provider: "codex", Mode: "cli", CLIPath: "codex", TmuxCommand: "codex --dangerously-bypass-approvals-and-sandbox"},
		},
	}

	plan := buildTmuxLaunchPlan(config, "/tmp/session")
	command := plan.MainCommand
	if command == "" {
		t.Fatal("expected main codex launch command")
	}
	assertContainsCommandPart(t, command, "codex")
	assertContainsCommandPart(t, command, "planner/requirement.snapshot.md")
	assertContainsCommandPart(t, command, "planner/master_schedule.md")
	assertContainsCommandPart(t, command, "planner/global_progress.md")
	assertContainsCommandPart(t, command, "main interactive ready")
}

func TestBuildTmuxLaunchCommandsBuildsCodexInteractiveCommand(t *testing.T) {
	config := configmodel.RuntimeConfig{
		LLMProfiles: map[string]configmodel.LLMProfile{
			"codex_cli": {Provider: "codex", Mode: "cli", CLIPath: "codex", TmuxCommand: "codex --dangerously-bypass-approvals-and-sandbox"},
		},
		Roles: []configmodel.RoleConfig{{
			ID:           "agent_a",
			LLMProfile:   "codex_cli",
			SystemPrompt: "你负责执行任务。",
		}},
	}

	commands := buildTmuxLaunchCommands(config, "/tmp/session")
	command := commands["agent_a"]
	if command == "" {
		t.Fatal("expected codex launch command")
	}
	assertContainsCommandPart(t, command, "codex")
	assertContainsCommandPart(t, command, "--dangerously-bypass-approvals-and-sandbox")
	assertContainsCommandPart(t, command, "--skip-git-repo-check")
	assertContainsCommandPart(t, command, "--no-alt-screen")
	assertContainsCommandPart(t, command, "planner/requirement.snapshot.md")
	assertContainsCommandPart(t, command, "roles/agent_a/context.md")
	assertContainsCommandPart(t, command, "roles/agent_a/inbox.md")
	assertContainsCommandPart(t, command, "你负责执行任务。")
}

func TestBuildTmuxLaunchCommandsRendersTemplatePlaceholders(t *testing.T) {
	config := configmodel.RuntimeConfig{
		LLMProfiles: map[string]configmodel.LLMProfile{
			"codex_cli": {
				Provider:    "codex",
				Mode:        "cli",
				CLIPath:     "codex",
				TmuxCommand: "codex --dangerously-bypass-approvals-and-sandbox -C {session_dir} {prompt} --role {role_id}",
			},
		},
		Roles: []configmodel.RoleConfig{{
			ID:           "agent_a",
			LLMProfile:   "codex_cli",
			SystemPrompt: "你负责执行任务。",
		}},
	}

	commands := buildTmuxLaunchCommands(config, "/tmp/session")
	command := commands["agent_a"]
	assertContainsCommandPart(t, command, "/tmp/session")
	assertContainsCommandPart(t, command, "agent_a")
	assertContainsCommandPart(t, command, "planner/requirement.snapshot.md")
	assertContainsCommandPart(t, command, "roles/agent_a/context.md")
}

func TestBuildTmuxLaunchCommandsSkipsUnsupportedProfile(t *testing.T) {
	config := configmodel.RuntimeConfig{
		LLMProfiles: map[string]configmodel.LLMProfile{
			"demo": {Provider: "deterministic", Mode: "cli", CLIPath: "demo"},
		},
		Roles: []configmodel.RoleConfig{{ID: "agent_a", LLMProfile: "demo"}},
	}

	commands := buildTmuxLaunchCommands(config, "/tmp/session")
	if _, ok := commands["agent_a"]; ok {
		t.Fatalf("expected unsupported profile to be skipped, got %#v", commands)
	}
	if len(commands) != 0 {
		t.Fatalf("expected empty commands, got %#v", commands)
	}
}

func TestBuildTmuxLaunchCommandsNormalizesRelativeSessionDirToAbsolutePath(t *testing.T) {
	config := configmodel.RuntimeConfig{
		LLMProfiles: map[string]configmodel.LLMProfile{
			"codex_cli": {
				Provider:    "codex",
				Mode:        "cli",
				CLIPath:     "codex",
				TmuxCommand: "/usr/bin/env codex --dangerously-bypass-approvals-and-sandbox --no-alt-screen -C {session_dir} {prompt}",
			},
		},
		Roles: []configmodel.RoleConfig{{ID: "agent_a", LLMProfile: "codex_cli", SystemPrompt: "你负责执行任务。"}},
	}
	absolute, err := filepath.Abs("config/.octopus/sessions/sess_test")
	if err != nil {
		t.Fatalf("filepath.Abs returned error: %v", err)
	}
	commands := buildTmuxLaunchCommands(config, "config/.octopus/sessions/sess_test")
	command := commands["agent_a"]
	assertContainsCommandPart(t, command, absolute)
	if strings.Contains(command, "cd 'config/.octopus/sessions/sess_test'") {
		t.Fatalf("expected relative session dir to be normalized, got %q", command)
	}
}

func assertContainsCommandPart(t *testing.T, command string, expected string) {
	t.Helper()
	if !strings.Contains(command, expected) {
		t.Fatalf("expected command %q to contain %q", command, expected)
	}
}
