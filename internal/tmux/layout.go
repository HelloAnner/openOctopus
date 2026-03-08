/*
Package tmux layout 负责 layout.md 的渲染与读取。
Author: Anner
Created on 2026/3/8
*/
package tmux

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	layoutDirName  = "tmux"
	layoutFileName = "layout.md"
	mainRoleID     = "main"
)

// ReadLayout 读取 session 下的 tmux layout 协议文件。
func ReadLayout(sessionDir string) (Layout, error) {
	content, err := os.ReadFile(layoutPath(sessionDir))
	if err != nil {
		return Layout{}, err
	}
	return parseLayout(string(content), sessionDir)
}

func writeLayout(sessionDir string, layout Layout) error {
	if err := os.MkdirAll(filepath.Dir(layoutPath(sessionDir)), 0o755); err != nil {
		return err
	}
	return os.WriteFile(layoutPath(sessionDir), renderLayout(layout), 0o644)
}

func layoutPath(sessionDir string) string {
	return filepath.Join(sessionDir, "state", layoutDirName, layoutFileName)
}

func renderLayout(layout Layout) []byte {
	lines := renderLayoutHeader(layout)
	lines = append(lines, "", "## Pane Map", "")
	lines = append(lines, renderPaneBinding(PaneBinding{RoleID: mainRoleID, PaneID: layout.MainPaneID, Title: mainPaneTitle()})...)
	for _, roleID := range sortedRoleIDs(layout.RolePanes) {
		lines = append(lines, renderPaneBinding(layout.RolePanes[roleID])...)
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderLayoutHeader(layout Layout) []string {
	return []string{
		"# TMUX Layout",
		"",
		fmt.Sprintf("- session_id: %s", layout.SessionID),
		fmt.Sprintf("- socket_name: %s", layout.SocketName),
		fmt.Sprintf("- session_name: %s", layout.SessionName),
		fmt.Sprintf("- window_name: %s", layout.WindowName),
		fmt.Sprintf("- role_layout: %s", layout.RoleLayout),
		fmt.Sprintf("- main_pane_ratio: %.2f", layout.MainPaneRatio),
		fmt.Sprintf("- main_pane_id: %s", layout.MainPaneID),
		fmt.Sprintf("- updated_at: %s", layout.UpdatedAt),
	}
}

func renderPaneBinding(binding PaneBinding) []string {
	lines := []string{"### role"}
	if binding.RoleID == mainRoleID {
		lines[0] = "### main"
	}
	lines = append(lines,
		fmt.Sprintf("- role_id: %s", binding.RoleID),
		fmt.Sprintf("- pane_id: %s", binding.PaneID),
		fmt.Sprintf("- title: %s", binding.Title),
		"",
	)
	return lines
}

func sortedRoleIDs(values map[string]PaneBinding) []string {
	roleIDs := make([]string, 0, len(values))
	for roleID := range values {
		roleIDs = append(roleIDs, roleID)
	}
	sort.Strings(roleIDs)
	return roleIDs
}

func parseLayout(content string, sessionDir string) (Layout, error) {
	layout := Layout{SessionDir: sessionDir, RolePanes: make(map[string]PaneBinding)}
	section := ""
	binding := PaneBinding{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		section, binding = parseLayoutLine(line, section, binding, &layout)
	}
	if scannerErr := scanner.Err(); scannerErr != nil {
		return Layout{}, scannerErr
	}
	flushPaneBinding(section, binding, &layout)
	layout.SessionDir = sessionDir
	return layout, nil
}

func parseLayoutLine(line string, section string, binding PaneBinding, layout *Layout) (string, PaneBinding) {
	if line == "" || line == "## Pane Map" || strings.HasPrefix(line, "# ") {
		return section, binding
	}
	if strings.HasPrefix(line, "### ") {
		flushPaneBinding(section, binding, layout)
		return strings.TrimPrefix(line, "### "), PaneBinding{}
	}
	if section == "" {
		key, value := parseKeyValue(line)
		assignLayoutValue(key, value, layout)
		return section, binding
	}
	key, value := parseKeyValue(line)
	assignBindingValue(key, value, &binding)
	return section, binding
}

func flushPaneBinding(section string, binding PaneBinding, layout *Layout) {
	if section == "" || binding.RoleID == "" {
		return
	}
	if binding.RoleID == mainRoleID {
		layout.MainPaneID = binding.PaneID
		return
	}
	layout.RolePanes[binding.RoleID] = binding
}

func parseKeyValue(line string) (string, string) {
	trimmed := strings.TrimPrefix(line, "- ")
	parts := strings.SplitN(trimmed, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func assignLayoutValue(key string, value string, layout *Layout) {
	switch key {
	case "session_id":
		layout.SessionID = value
	case "socket_name":
		layout.SocketName = value
	case "session_name":
		layout.SessionName = value
	case "window_name":
		layout.WindowName = value
	case "role_layout":
		layout.RoleLayout = value
	case "main_pane_ratio":
		layout.MainPaneRatio, _ = strconv.ParseFloat(value, 64)
	case "main_pane_id":
		layout.MainPaneID = value
	case "updated_at":
		layout.UpdatedAt = value
	}
}

func assignBindingValue(key string, value string, binding *PaneBinding) {
	switch key {
	case "role_id":
		binding.RoleID = value
	case "pane_id":
		binding.PaneID = value
	case "title":
		binding.Title = value
	}
}
