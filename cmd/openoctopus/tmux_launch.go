/*
Package main tmux_launch 负责构造 tmux role pane 的交互启动命令。
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

func buildTmuxLaunchCommands(config configmodel.RuntimeConfig, sessionDir string) map[string]string {
	normalizedSessionDir := normalizeTmuxSessionDir(sessionDir)
	commands := make(map[string]string)
	for _, role := range config.Roles {
		profile, ok := config.LLMProfiles[role.LLMProfile]
		if !ok {
			continue
		}
		command := buildRoleLaunchCommand(normalizedSessionDir, role, profile)
		if command != "" {
			commands[role.ID] = command
		}
	}
	return commands
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

func buildRoleLaunchCommand(sessionDir string, role configmodel.RoleConfig, profile configmodel.LLMProfile) string {
	configured := strings.TrimSpace(profile.TmuxCommand)
	if configured != "" {
		return buildConfiguredLaunchCommand(sessionDir, role, profile, configured)
	}
	if !supportsDefaultInteractivePane(profile) {
		return ""
	}
	return buildCodexLaunchCommand(sessionDir, role, shellQuoteValue(profile.CLIPath))
}

func supportsDefaultInteractivePane(profile configmodel.LLMProfile) bool {
	provider := strings.TrimSpace(strings.ToLower(profile.Provider))
	mode := strings.TrimSpace(strings.ToLower(profile.Mode))
	return provider == "codex" && mode == "cli" && strings.TrimSpace(profile.CLIPath) != ""
}

func buildConfiguredLaunchCommand(sessionDir string, role configmodel.RoleConfig, profile configmodel.LLMProfile, configured string) string {
	if usesTmuxCommandTemplate(configured) {
		rendered := renderTmuxCommandTemplate(configured, sessionDir, role.ID, buildRoleStartupPrompt(role))
		return wrapRoleLaunchCommand(role.ID, sessionDir, rendered)
	}
	if supportsDefaultInteractivePane(profile) {
		return buildCodexLaunchCommand(sessionDir, role, configured)
	}
	return wrapRoleLaunchCommand(role.ID, sessionDir, configured)
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

func buildCodexLaunchCommand(sessionDir string, role configmodel.RoleConfig, baseCommand string) string {
	prompt := buildRoleStartupPrompt(role)
	root := shellQuoteValue(sessionDir)
	quotedPrompt := shellQuoteValue(prompt)
	rendered := fmt.Sprintf("%s --skip-git-repo-check --no-alt-screen -C %s %s", strings.TrimSpace(baseCommand), root, quotedPrompt)
	return wrapRoleLaunchCommand(role.ID, sessionDir, rendered)
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

func shellQuoteValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
