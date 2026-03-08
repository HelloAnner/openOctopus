/*
Package tmux attach 负责 tmux 客户端进入策略。
Author: Anner
Created on 2026/3/8
*/
package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AttachSession 根据当前终端环境进入已创建的 tmux session。
func AttachSession(socketName string, sessionName string) error {
	if err := validateAttachTarget(socketName, sessionName); err != nil {
		return err
	}
	if insideTmuxClient() {
		return openNestedWindow(socketName, sessionName)
	}
	return attachCurrentTerminal(socketName, sessionName)
}

func validateAttachTarget(socketName string, sessionName string) error {
	if strings.TrimSpace(socketName) == "" {
		return fmt.Errorf("tmux socket name is required")
	}
	if strings.TrimSpace(sessionName) == "" {
		return fmt.Errorf("tmux session name is required")
	}
	return nil
}

func insideTmuxClient() bool {
	return strings.TrimSpace(os.Getenv("TMUX")) != ""
}

func openNestedWindow(socketName string, sessionName string) error {
	windowName := nestedWindowName(sessionName)
	shellCommand := buildNestedAttachCommand(socketName, sessionName)
	command := exec.Command("tmux", "new-window", "-n", windowName, shellCommand)
	output, err := command.CombinedOutput()
	if err != nil {
		return buildCommandError([]string{"new-window", "-n", windowName, shellCommand}, err, output)
	}
	return nil
}

func nestedWindowName(sessionName string) string {
	trimmed := strings.TrimSpace(sessionName)
	if trimmed == "" {
		return "openoctopus"
	}
	return "oo:" + trimmed
}

func buildNestedAttachCommand(socketName string, sessionName string) string {
	return fmt.Sprintf("exec env TMUX= tmux -L %s attach-session -t %s", shellQuote(socketName), shellQuote(sessionName))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func attachCurrentTerminal(socketName string, sessionName string) error {
	command := exec.Command("tmux", buildCommandArgs(socketName, "attach-session", "-t", sessionName)...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("tmux attach-session -t %s: %w", sessionName, err)
	}
	return nil
}
