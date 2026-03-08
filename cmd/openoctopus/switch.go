/*
Package main switch 提供 tmux pane 切换命令。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"fmt"
	"strings"

	"github.com/anner/openoctopus/internal/tmux"
	"github.com/spf13/cobra"
)

func newSwitchCommand() *cobra.Command {
	var sessionRef string
	var roleID string
	var mainPane bool
	var format string
	command := &cobra.Command{
		Use:   "switch",
		Short: "切换或解析 tmux pane 目标",
		RunE: func(command *cobra.Command, _ []string) error {
			return executeSwitchCommand(command, sessionRef, roleID, mainPane, format)
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&roleID, "role", "", "target role id")
	command.Flags().BoolVar(&mainPane, "main", false, "switch to main pane")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	_ = command.MarkFlagRequired("session")
	return command
}

func executeSwitchCommand(command *cobra.Command, sessionRef string, roleID string, mainPane bool, format string) error {
	if err := validateSwitchFlags(roleID, mainPane); err != nil {
		return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "switch", err)
	}
	sessionDir, err := resolveCommandSessionDir(sessionRef)
	if err != nil {
		return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "switch", mapStatusError(err))
	}
	result, err := tmux.NewService(sessionDir).Switch(tmux.ResolveTargetOptions{RoleID: roleID, Main: mainPane})
	if err != nil {
		return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "switch", err)
	}
	return writeSwitchSuccess(command, format, result)
}

func validateSwitchFlags(roleID string, mainPane bool) error {
	hasRole := strings.TrimSpace(roleID) != ""
	if hasRole == mainPane {
		return fmt.Errorf("exactly one of role or main is required")
	}
	return nil
}

func writeSwitchSuccess(command *cobra.Command, format string, result tmux.ResolveTargetResult) error {
	text := fmt.Sprintf("switch target resolved: %s", result.TargetPaneID)
	data := map[string]any{
		"session_dir":    result.SessionDir,
		"socket_name":    result.SocketName,
		"session_name":   result.SessionName,
		"target_role":    result.TargetRole,
		"target_pane_id": result.TargetPaneID,
		"switched":       result.Switched,
	}
	return writeCommandSuccess(command.OutOrStdout(), command.ErrOrStderr(), format, "switch", text, data)
}
