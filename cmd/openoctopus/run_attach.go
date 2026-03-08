/*
Package main run_attach 负责 run 成功后的 tmux 自动进入。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	configmodel "github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/session"
	"github.com/anner/openoctopus/internal/tmux"
	"github.com/spf13/cobra"
)

type runSuccessHooks struct {
	IsInteractive func(command *cobra.Command) bool
	WriteSuccess  func(command *cobra.Command, format string, createResult session.CreateResult) error
	AttachTmux    func(result tmux.BootstrapResult) error
	WriteWarning  func(command *cobra.Command, message string) error
}

func defaultRunSuccessHooks() runSuccessHooks {
	return runSuccessHooks{
		IsInteractive: isInteractiveCommand,
		WriteSuccess:  writeRunSuccess,
		AttachTmux:    attachRunTmux,
		WriteWarning:  writeRunWarning,
	}
}

func handleRunSuccess(command *cobra.Command, format string, config configmodel.RuntimeConfig, createResult session.CreateResult, tmuxResult tmux.BootstrapResult, hooks runSuccessHooks) error {
	if err := hooks.WriteSuccess(command, format, createResult); err != nil {
		return err
	}
	if !shouldAutoAttachTmuxRun(command, format, config, tmuxResult, hooks) {
		return nil
	}
	if err := hooks.AttachTmux(tmuxResult); err != nil {
		return hooks.WriteWarning(command, fmt.Sprintf("warning: tmux attach failed, session remains available: %v", err))
	}
	return nil

}

func shouldAutoAttachTmuxRun(command *cobra.Command, format string, config configmodel.RuntimeConfig, tmuxResult tmux.BootstrapResult, hooks runSuccessHooks) bool {
	if normalizeFormat(format) == "json" {
		return false
	}
	if !config.Runtime.Tmux.Enabled || tmuxResult.SocketName == "" || tmuxResult.SessionName == "" {
		return false
	}
	return hooks.IsInteractive(command)
}

func shouldPrepareInteractiveTmux(command *cobra.Command, format string, config configmodel.RuntimeConfig, hooks runSuccessHooks) bool {
	if normalizeFormat(format) == "json" {
		return false
	}
	if !config.Runtime.Tmux.Enabled {
		return false
	}
	return hooks.IsInteractive(command)
}

func attachRunTmux(result tmux.BootstrapResult) error {
	return tmux.AttachSession(result.SocketName, result.SessionName)
}

func writeRunWarning(command *cobra.Command, message string) error {
	_, err := fmt.Fprintln(command.ErrOrStderr(), message)
	return err
}

func isInteractiveCommand(command *cobra.Command) bool {
	if !isTerminalFile(os.Stdin) {
		return false
	}
	if !isTerminalWriter(command.OutOrStdout()) {
		return false
	}
	return isTerminalWriter(command.ErrOrStderr())
}

func isTerminalWriter(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	return isTerminalFile(file)
}

func isTerminalFile(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	term := strings.TrimSpace(os.Getenv("TERM"))
	return term != "" && term != "dumb"
}
