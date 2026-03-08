/*
Package tmux command 封装 tmux 原生命令执行。
Author: Anner
Created on 2026/3/8
*/
package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

type commandRunner interface {
	Run(socketName string, args ...string) (string, error)
}

type shellRunner struct{}

func (r shellRunner) Run(socketName string, args ...string) (string, error) {
	if strings.TrimSpace(socketName) == "" {
		return "", fmt.Errorf("tmux socket name is required")
	}
	command := exec.Command("tmux", buildCommandArgs(socketName, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", buildCommandError(args, err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

func buildCommandArgs(socketName string, args ...string) []string {
	commandArgs := []string{"-L", socketName}
	return append(commandArgs, args...)
}

func buildCommandError(args []string, err error, output []byte) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return fmt.Errorf("tmux %s: %w", strings.Join(args, " "), err)
	}
	return fmt.Errorf("tmux %s: %w: %s", strings.Join(args, " "), err, message)
}

func ensureTmuxInstalled() error {
	_, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux binary not found: %w", err)
	}
	return nil
}
