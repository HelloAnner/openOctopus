package main

import (
	"fmt"

	"github.com/anner/openoctopus/internal/humangate"
	"github.com/spf13/cobra"
)

func newInterruptCommand() *cobra.Command {
	var sessionRef string
	var roleID string
	var reason string
	var format string
	command := &cobra.Command{
		Use:   "interrupt",
		Short: "中断指定角色并进入人工等待",
		RunE: func(command *cobra.Command, _ []string) error {
			sessionDir, err := resolveCommandSessionDir(sessionRef)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "interrupt", mapStatusError(err))
			}
			record, err := humangate.NewService(sessionDir).Interrupt(humangate.InterruptOptions{RoleID: roleID, Reason: reason})
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "interrupt", err)
			}
			return writeCommandSuccess(
				command.OutOrStdout(),
				command.ErrOrStderr(),
				format,
				"interrupt",
				fmt.Sprintf("interrupt requested: %s", record.InterruptID),
				map[string]any{"interrupt_id": record.InterruptID, "target_role_id": record.TargetRoleID, "status": record.Status, "session_dir": sessionDir},
			)
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	command.Flags().StringVar(&roleID, "role", "", "target role id")
	command.Flags().StringVar(&reason, "reason", "", "interrupt reason")
	_ = command.MarkFlagRequired("session")
	_ = command.MarkFlagRequired("role")
	_ = command.MarkFlagRequired("reason")
	return command
}
