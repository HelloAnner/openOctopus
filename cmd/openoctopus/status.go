/*
Package main status 提供 session 状态读取命令。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"errors"

	clisupport "github.com/anner/openoctopus/internal/cli"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var sessionRef string
	var format string
	command := &cobra.Command{
		Use:   "status",
		Short: "读取 session 当前状态",
		RunE: func(command *cobra.Command, _ []string) error {
			service, err := newStatusService()
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "status", err)
			}
			sessionDir, err := service.ResolveSessionDir(sessionRef)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "status", mapStatusError(err))
			}
			summary, err := service.ReadStatus(sessionDir)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "status", err)
			}
			return writeCommandSuccess(command.OutOrStdout(), command.ErrOrStderr(), format, "status", renderStatusText(summary), summary)
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	_ = command.MarkFlagRequired("session")
	return command
}

func mapStatusError(err error) error {
	if errors.Is(err, clisupport.ErrSessionNotFound) {
		return newCLIError("session_not_found", err.Error(), exitCodeSessionNotFound, err, nil)
	}
	return err
}
