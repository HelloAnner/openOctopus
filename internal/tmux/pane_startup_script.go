/*
Package tmux pane_startup_script 负责为 pane 生成稳定启动脚本。
Author: Anner
Created on 2026/3/8
*/
package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) writePaneStartupScript(paneKey string, command string) (string, error) {
	scriptPath := paneStartupScriptPath(s.sessionDir, paneKey)
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return "", err
	}
	content := buildPaneStartupScript(s.sessionDir, paneKey, command)
	if err := os.WriteFile(scriptPath, content, 0o755); err != nil {
		return "", err
	}
	return scriptPath, nil
}

func paneStartupScriptPath(sessionDir string, paneKey string) string {
	return filepath.Join(sessionDir, "state", "tmux", "scripts", sanitizePaneKey(paneKey)+".sh")
}

func sanitizePaneKey(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_")
	return replacer.Replace(strings.TrimSpace(value))
}

func buildPaneStartupScript(sessionDir string, paneKey string, command string) []byte {
	lines := []string{"#!/bin/sh"}
	lines = append(lines, readinessLines(sessionDir, paneKey)...)
	lines = append(lines,
		strings.TrimSpace(command),
		"status=$?",
		`if [ "$status" -ne 0 ]; then`,
		fmt.Sprintf("  printf %s \"$status\"", shellQuoteValue("[openoctopus] pane command exited with status=%s\\n")),
		"fi",
		`exec "${SHELL:-/bin/zsh}" -l`,
		"",
	)
	return []byte(strings.Join(lines, "\n"))
}

func readinessLines(sessionDir string, paneKey string) []string {
	if paneKey == mainRoleID {
		return nil
	}
	refs := roleReadyRefs(sessionDir, paneKey)
	return []string{
		fmt.Sprintf("printf %s", shellQuoteValue("[openoctopus] role="+paneKey+" waiting for first dispatch\\n")),
		"ready=0",
		"while [ \"$ready\" -eq 0 ]; do",
		fmt.Sprintf("  if [ -f %s ] && [ -f %s ] && [ -f %s ]; then", refs[0], refs[1], refs[2]),
		"    ready=1",
		"  else",
		"    sleep 1",
		"  fi",
		"done",
	}
}

func roleReadyRefs(sessionDir string, paneKey string) []string {
	root := normalizePaneSessionDir(sessionDir)
	return []string{
		shellQuoteValue(filepath.Join(root, "planner", "requirement.snapshot.md")),
		shellQuoteValue(filepath.Join(root, "roles", paneKey, "context.md")),
		shellQuoteValue(filepath.Join(root, "roles", paneKey, "inbox.md")),
	}
}

func normalizePaneSessionDir(sessionDir string) string {
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

func buildPaneScriptCommand(scriptPath string) string {
	return "sh " + shellQuoteValue(scriptPath)
}

func shellQuoteValue(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
