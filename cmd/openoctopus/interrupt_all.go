package main

import (
	"fmt"

	"github.com/anner/openoctopus/internal/humangate"
	"github.com/spf13/cobra"
)

func newInterruptAllCommand() *cobra.Command {
	var sessionRef string
	var reason string
	var format string
	command := &cobra.Command{
		Use:   "interrupt-all",
		Short: "中断全部未完成角色并等待人工",
		RunE: func(command *cobra.Command, _ []string) error {
			sessionDir, err := resolveCommandSessionDir(sessionRef)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "interrupt-all", mapStatusError(err))
			}
			result, err := humangate.NewService(sessionDir).InterruptAll(humangate.InterruptAllOptions{Reason: reason})
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "interrupt-all", err)
			}
			return writeCommandSuccess(
				command.OutOrStdout(),
				command.ErrOrStderr(),
				format,
				"interrupt-all",
				fmt.Sprintf("interrupt-all requested: %d", result.RequestedCount),
				map[string]any{"requested_count": result.RequestedCount, "session_dir": sessionDir},
			)
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	command.Flags().StringVar(&reason, "reason", "", "interrupt reason")
	_ = command.MarkFlagRequired("session")
	_ = command.MarkFlagRequired("reason")
	return command
}
