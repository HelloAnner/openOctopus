/*
Package main tmux_launch 负责构造 tmux agent pane 启动命令。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"fmt"
	"path/filepath"
	"strings"

	configmodel "github.com/anner/openoctopus/internal/config/model"
)

type tmuxLaunchPlan struct {
	MainCommand  string
	RoleCommands map[string]string
}

func buildTmuxLaunchPlan(config configmodel.RuntimeConfig, sessionDir string) tmuxLaunchPlan {
	normalizedSessionDir := normalizeTmuxSessionDir(sessionDir)
	return tmuxLaunchPlan{
		MainCommand:  buildMainTmuxLaunchCommand(config, normalizedSessionDir),
		RoleCommands: buildRoleTmuxLaunchCommands(config, normalizedSessionDir),
	}
}

func buildTmuxLaunchCommands(config configmodel.RuntimeConfig, sessionDir string) map[string]string {
	return buildTmuxLaunchPlan(config, sessionDir).RoleCommands
}

func buildRoleTmuxLaunchCommands(config configmodel.RuntimeConfig, sessionDir string) map[string]string {
	commands := make(map[string]string)
	for _, role := range config.Roles {
		profile, ok := config.LLMProfiles[role.LLMProfile]
		if !ok {
			continue
		}
		command := buildInteractiveLaunchCommand(sessionDir, role.ID, buildRoleStartupPrompt(role), profile)
		if command != "" {
			commands[role.ID] = command
		}
	}
	return commands
}

func buildMainTmuxLaunchCommand(config configmodel.RuntimeConfig, sessionDir string) string {
	profileID := strings.TrimSpace(config.Runtime.Tmux.MainLLMProfile)
	if profileID == "" {
		return ""
	}
	profile, ok := config.LLMProfiles[profileID]
	if !ok {
		return ""
	}
	return buildInteractiveLaunchCommand(sessionDir, "main", buildMainStartupPrompt(config), profile)
}

func normalizeTmuxSessionDir(sessionDir string) string {
	trimmed := strings.TrimSpace(sessionDir)
	if trimmed == "" {
		return trimmed
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return trimmed
	}
	return absolute
}

func buildInteractiveLaunchCommand(sessionDir string, targetID string, prompt string, profile configmodel.LLMProfile) string {
	configured := strings.TrimSpace(profile.TmuxCommand)
	if configured != "" {
		return buildConfiguredLaunchCommand(sessionDir, targetID, prompt, profile, configured)
	}
	if !supportsDefaultInteractivePane(profile) {
		return ""
	}
	return buildCodexLaunchCommand(sessionDir, targetID, prompt, shellQuoteValue(profile.CLIPath))
}

func supportsDefaultInteractivePane(profile configmodel.LLMProfile) bool {
	provider := strings.TrimSpace(strings.ToLower(profile.Provider))
	mode := strings.TrimSpace(strings.ToLower(profile.Mode))
	return provider == "codex" && mode == "cli" && strings.TrimSpace(profile.CLIPath) != ""
}

func buildConfiguredLaunchCommand(sessionDir string, targetID string, prompt string, profile configmodel.LLMProfile, configured string) string {
	if usesTmuxCommandTemplate(configured) {
		rendered := renderTmuxCommandTemplate(configured, sessionDir, targetID, prompt)
		return wrapRoleLaunchCommand(targetID, sessionDir, rendered)
	}
	if supportsDefaultInteractivePane(profile) {
		return buildCodexLaunchCommand(sessionDir, targetID, prompt, configured)
	}
	return wrapRoleLaunchCommand(targetID, sessionDir, configured)
}

func usesTmuxCommandTemplate(command string) bool {
	return strings.Contains(command, "{session_dir}") || strings.Contains(command, "{role_id}") || strings.Contains(command, "{prompt}")
}

func renderTmuxCommandTemplate(template string, sessionDir string, roleID string, prompt string) string {
	replacer := strings.NewReplacer(
		"{session_dir}", shellQuoteValue(sessionDir),
		"{role_id}", shellQuoteValue(roleID),
		"{prompt}", shellQuoteValue(prompt),
	)
	return replacer.Replace(template)
}

func buildCodexLaunchCommand(sessionDir string, targetID string, prompt string, baseCommand string) string {
	root := shellQuoteValue(sessionDir)
	quotedPrompt := shellQuoteValue(prompt)
	rendered := fmt.Sprintf("%s --skip-git-repo-check --no-alt-screen -C %s %s", strings.TrimSpace(baseCommand), root, quotedPrompt)
	return wrapRoleLaunchCommand(targetID, sessionDir, rendered)
}

func wrapRoleLaunchCommand(roleID string, sessionDir string, command string) string {
	root := shellQuoteValue(sessionDir)
	return fmt.Sprintf("clear; printf '[openoctopus] role=%s interactive ready\\n'; cd %s && %s", roleID, root, strings.TrimSpace(command))
}

func buildRoleStartupPrompt(role configmodel.RoleConfig) string {
	paths := []string{
		filepath.ToSlash(filepath.Join("planner", "requirement.snapshot.md")),
		filepath.ToSlash(filepath.Join("roles", role.ID, "context.md")),
		filepath.ToSlash(filepath.Join("roles", role.ID, "inbox.md")),
	}
	lines := []string{
		fmt.Sprintf("You are the interactive standby assistant for role %s.", role.ID),
		"Work only inside the current session directory.",
		"Read these files first:",
	}
	if strings.TrimSpace(role.SystemPrompt) != "" {
		lines = append(lines, "System prompt:", role.SystemPrompt)
	}
	for _, path := range paths {
		lines = append(lines, "- "+path)
	}
	lines = append(lines,
		"After reading, briefly summarize what you are ready to help with.",
		"Wait for the human's next instruction in this interactive session.",
		"Do not modify any files until the human explicitly asks you here.",
	)
	return strings.Join(lines, "\n")
}

func buildMainStartupPrompt(config configmodel.RuntimeConfig) string {
	paths := []string{
		filepath.ToSlash(filepath.Join("planner", "requirement.snapshot.md")),
		filepath.ToSlash(filepath.Join("planner", "master_schedule.md")),
		filepath.ToSlash(filepath.Join("planner", "task_board.md")),
		filepath.ToSlash(filepath.Join("planner", "global_progress.md")),
		filepath.ToSlash(filepath.Join("planner", "blockers.md")),
	}
	lines := []string{
		fmt.Sprintf("You are the interactive main assistant for workflow %s.", config.Meta.WorkflowID),
		"Work only inside the current session directory.",
		"Read these files first:",
	}
	for _, path := range paths {
		lines = append(lines, "- "+path)
	}
	lines = append(lines,
		"Help the human inspect workflow status, role handoff, blockers, and artifacts.",
		"Wait for the human's next instruction in this interactive session.",
		"Do not modify any files until the human explicitly asks you here.",
	)
	return strings.Join(lines, "\n")
}

func shellQuoteValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
