/*
Package tmux service 负责 tmux session 创建、切换与清理。
Author: Anner
Created on 2026/3/8
*/
package tmux

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const defaultWindowName = "workspace"

// NewService 创建一个 tmux 服务实例。
func NewService(sessionDir string) *Service {
	return &Service{sessionDir: sessionDir, runner: shellRunner{}}
}

// Bootstrap 为 session 创建真实 tmux 布局与 layout 协议文件。
func (s *Service) Bootstrap(options BootstrapOptions) (BootstrapResult, error) {
	if err := ensureTmuxInstalled(); err != nil {
		return BootstrapResult{}, err
	}
	layout, err := s.bootstrapLayout(options)
	if err != nil {
		return BootstrapResult{}, err
	}
	if err := writeLayout(s.sessionDir, layout); err != nil {
		_ = s.KillSession(layout.SocketName, layout.SessionName)
		return BootstrapResult{}, err
	}
	return buildBootstrapResult(layout), nil
}

func (s *Service) bootstrapLayout(options BootstrapOptions) (Layout, error) {
	socketName, err := expandSocketName(options.SocketTemplate, options.SessionID)
	if err != nil {
		return Layout{}, err
	}
	sessionName := socketName
	mainPaneID, err := s.createSession(socketName, sessionName)
	if err != nil {
		return Layout{}, err
	}
	rolePanes, err := s.createRolePanes(socketName, mainPaneID, options.RoleIDs, options.RoleLayout)
	if err != nil {
		_ = s.KillSession(socketName, sessionName)
		return Layout{}, err
	}
	if err := s.finalizeLayout(socketName, sessionName, mainPaneID, rolePanes, options); err != nil {
		_ = s.KillSession(socketName, sessionName)
		return Layout{}, err
	}
	return newLayout(options, socketName, sessionName, mainPaneID, rolePanes), nil
}

func buildBootstrapResult(layout Layout) BootstrapResult {
	return BootstrapResult{SocketName: layout.SocketName, SessionName: layout.SessionName, WindowName: layout.WindowName, MainPaneID: layout.MainPaneID, RolePanes: layout.RolePanes}
}

func (s *Service) createSession(socketName string, sessionName string) (string, error) {
	return s.runner.Run(socketName, "new-session", "-d", "-P", "-F", "#{pane_id}", "-s", sessionName, "-n", defaultWindowName)
}

func (s *Service) createRolePanes(socketName string, mainPaneID string, roleIDs []string, roleLayout string) (map[string]PaneBinding, error) {
	bindings := make(map[string]PaneBinding)
	if len(roleIDs) == 0 {
		return bindings, nil
	}
	firstPaneID, err := s.createRolePane(socketName, mainPaneID, 0, roleLayout)
	if err != nil {
		return nil, err
	}
	bindings[roleIDs[0]] = newRoleBinding(roleIDs[0], firstPaneID)
	for index := 1; index < len(roleIDs); index++ {
		paneID, splitErr := s.createRolePane(socketName, firstPaneID, index, roleLayout)
		if splitErr != nil {
			return nil, splitErr
		}
		bindings[roleIDs[index]] = newRoleBinding(roleIDs[index], paneID)
	}
	return bindings, nil
}

func (s *Service) createRolePane(socketName string, targetPaneID string, index int, roleLayout string) (string, error) {
	args := []string{"split-window", "-d", "-P", "-F", "#{pane_id}", splitDirection(index, roleLayout), "-t", targetPaneID}
	if index == 0 {
		args = append(args, "-p", "50")
	}
	return s.runner.Run(socketName, args...)
}

func splitDirection(index int, roleLayout string) string {
	if roleLayout == "tiled" && index%2 == 1 {
		return "-h"
	}
	return "-v"
}

func (s *Service) finalizeLayout(socketName string, sessionName string, mainPaneID string, rolePanes map[string]PaneBinding, options BootstrapOptions) error {
	if err := s.runnerCommand(socketName, "select-layout", "-t", sessionName, "main-vertical"); err != nil {
		return err
	}
	if err := s.runnerCommand(socketName, "resize-pane", "-t", mainPaneID, "-x", paneWidthPercent(options.MainPaneRatio)); err != nil {
		return err
	}
	mainCommand := paneStartupCommand(mainRoleID, options.SessionID, options.MainLaunchCommand)
	if err := s.preparePane(socketName, mainPaneID, mainPaneTitle(), mainCommand, mainRoleID); err != nil {
		return err
	}
	for _, roleID := range sortedBindingKeys(rolePanes) {
		command := paneStartupCommand(roleID, options.SessionID, options.LaunchCommands[roleID])
		if err := s.preparePane(socketName, rolePanes[roleID].PaneID, paneTitle(roleID), command, roleID); err != nil {
			return err
		}
	}
	return s.activateDefaultPane(socketName, mainPaneID, rolePanes, options.RoleIDs)
}

func paneWidthPercent(ratio float64) string {
	return fmt.Sprintf("%.0f%%", ratio*100)
}

func (s *Service) preparePane(socketName string, paneID string, title string, command string, paneKey string) error {
	if err := s.runnerCommand(socketName, "select-pane", "-t", paneID, "-T", title); err != nil {
		return err
	}
	scriptPath, err := s.writePaneStartupScript(paneKey, command)
	if err != nil {
		return err
	}
	return s.runnerCommand(socketName, "respawn-pane", "-k", "-t", paneID, buildPaneScriptCommand(scriptPath))
}

func (s *Service) activateDefaultPane(socketName string, mainPaneID string, rolePanes map[string]PaneBinding, roleIDs []string) error {
	paneID := defaultFocusPaneID(mainPaneID, rolePanes, roleIDs)
	return s.runnerCommand(socketName, "select-pane", "-t", paneID)
}

func paneTitle(roleID string) string {
	if roleID == mainRoleID {
		return mainPaneTitle()
	}
	return "role:" + roleID
}

func mainPaneTitle() string {
	return mainRoleID
}

func paneBanner(roleID string, sessionID string) string {
	if roleID == mainRoleID {
		return fmt.Sprintf("printf '[openoctopus] main session=%s\\n'", sessionID)
	}
	return fmt.Sprintf("printf '[openoctopus] role=%s session=%s\\n'", roleID, sessionID)
}

func newLayout(options BootstrapOptions, socketName string, sessionName string, mainPaneID string, rolePanes map[string]PaneBinding) Layout {
	return Layout{
		SessionID:     options.SessionID,
		SocketName:    socketName,
		SessionName:   sessionName,
		WindowName:    defaultWindowName,
		RoleLayout:    options.RoleLayout,
		MainPaneRatio: options.MainPaneRatio,
		MainPaneID:    mainPaneID,
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
		RolePanes:     rolePanes,
	}
}

func newRoleBinding(roleID string, paneID string) PaneBinding {
	return PaneBinding{RoleID: roleID, PaneID: paneID, Title: paneTitle(roleID)}
}

func sortedBindingKeys(values map[string]PaneBinding) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s *Service) runnerCommand(socketName string, args ...string) error {
	_, err := s.runner.Run(socketName, args...)
	return err
}

// ResolveTarget 读取 layout 协议并解析目标 pane。
func (s *Service) ResolveTarget(options ResolveTargetOptions) (ResolveTargetResult, error) {
	layout, err := ReadLayout(s.sessionDir)
	if err != nil {
		return ResolveTargetResult{}, err
	}
	return resolveTarget(layout, options)
}

func resolveTarget(layout Layout, options ResolveTargetOptions) (ResolveTargetResult, error) {
	binding, roleID, err := findBinding(layout, options)
	if err != nil {
		return ResolveTargetResult{}, err
	}
	return ResolveTargetResult{SessionDir: layout.SessionDir, SocketName: layout.SocketName, SessionName: layout.SessionName, TargetRole: roleID, TargetPaneID: binding.PaneID}, nil
}

func findBinding(layout Layout, options ResolveTargetOptions) (PaneBinding, string, error) {
	if options.Main {
		return PaneBinding{RoleID: mainRoleID, PaneID: layout.MainPaneID, Title: mainPaneTitle()}, mainRoleID, nil
	}
	binding, ok := layout.RolePanes[strings.TrimSpace(options.RoleID)]
	if ok {
		return binding, binding.RoleID, nil
	}
	return PaneBinding{}, "", fmt.Errorf("tmux role pane not found: %s", options.RoleID)
}

// Switch 解析目标 pane，并在 tmux 客户端环境中尝试切换。
func (s *Service) Switch(options ResolveTargetOptions) (ResolveTargetResult, error) {
	result, err := s.ResolveTarget(options)
	if err != nil {
		return ResolveTargetResult{}, err
	}
	if strings.TrimSpace(os.Getenv("TMUX")) == "" {
		return result, nil
	}
	if err := s.runnerCommand(result.SocketName, "select-pane", "-t", result.TargetPaneID); err != nil {
		return ResolveTargetResult{}, err
	}
	result.Switched = true
	return result, nil
}

// CapturePane 捕获目标 pane 当前可见输出。
func (s *Service) CapturePane(options ResolveTargetOptions) (string, error) {
	result, err := s.ResolveTarget(options)
	if err != nil {
		return "", err
	}
	return s.runner.Run(result.SocketName, "capture-pane", "-p", "-t", result.TargetPaneID)
}

// KillSession 删除指定 tmux session。
func (s *Service) KillSession(socketName string, sessionName string) error {
	_, err := s.runner.Run(socketName, "kill-session", "-t", sessionName)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "can't find session") {
		return nil
	}
	return err
}

func expandSocketName(template string, sessionID string) (string, error) {
	trimmedTemplate := strings.TrimSpace(template)
	trimmedID := strings.TrimSpace(sessionID)
	if trimmedTemplate == "" || trimmedID == "" {
		return "", fmt.Errorf("tmux socket template and session id are required")
	}
	value := strings.ReplaceAll(trimmedTemplate, "{session_id}", trimmedID)
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("expanded tmux socket name is empty")
	}
	return value, nil
}
