package main

import (
	"fmt"

	"github.com/anner/openoctopus/internal/humangate"
	"github.com/anner/openoctopus/internal/orchestrator"
	"github.com/spf13/cobra"
)

func newResumeCommand() *cobra.Command {
	var sessionRef string
	var roleID string
	var format string
	command := &cobra.Command{
		Use:   "resume",
		Short: "恢复等待人工的会话",
		RunE: func(command *cobra.Command, _ []string) error {
			sessionDir, err := resolveCommandSessionDir(sessionRef)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "resume", mapStatusError(err))
			}
			result, err := humangate.NewService(sessionDir).Resume(humangate.ResumeOptions{RoleID: roleID})
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "resume", err)
			}
			master := orchestrator.NewEngine(sessionDir)
			tick, err := master.Tick()
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "resume", err)
			}
			if roleRuntimeLoopEnabled() {
				if err := driveRoleRuntimeLoop(sessionDir, master, tick.WorkflowStatus); err != nil {
					return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "resume", err)
				}
			}
			return writeCommandSuccess(
				command.OutOrStdout(),
				command.ErrOrStderr(),
				format,
				"resume",
				fmt.Sprintf("session resumed: %s", sessionDir),
				map[string]any{"session_dir": sessionDir, "cleared_interrupts": result.ClearedInterrupts, "requeued_stages": result.RequeuedStages},
			)
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	command.Flags().StringVar(&roleID, "role", "", "target role id")
	_ = command.MarkFlagRequired("session")
	return command
}
