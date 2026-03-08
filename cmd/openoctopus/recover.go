/*
Package main recover 提供 session 恢复命令。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"errors"
	"fmt"

	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/orchestrator"
	"github.com/anner/openoctopus/internal/recovery"
	"github.com/spf13/cobra"
)

func newRecoverCommand() *cobra.Command {
	var sessionRef string
	var format string
	command := &cobra.Command{
		Use:   "recover",
		Short: "恢复中断后的 session",
		RunE: func(command *cobra.Command, _ []string) error {
			sessionDir, err := resolveCommandSessionDir(sessionRef)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "recover", mapStatusError(err))
			}
			service := recovery.NewService(sessionDir)
			result, err := service.Recover(recovery.RecoverOptions{})
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "recover", mapRecoveryError(err))
			}
			continued := false
			if result.CanContinue {
				master := orchestrator.NewEngine(sessionDir)
				tick, tickErr := master.Tick()
				if tickErr != nil {
					return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "recover", tickErr)
				}
				if roleRuntimeLoopEnabled() {
					if loopErr := driveRoleRuntimeLoop(sessionDir, master, tick.WorkflowStatus); loopErr != nil {
						return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "recover", loopErr)
					}
				}
				continued = true
			}
			statusService, err := newStatusService()
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "recover", err)
			}
			summary, err := statusService.ReadStatus(sessionDir)
			if err == nil {
				result.RecoveredStatus = summary.WorkflowStatus
			}
			result.Continued = continued
			return writeCommandSuccess(
				command.OutOrStdout(),
				command.ErrOrStderr(),
				format,
				"recover",
				fmt.Sprintf("session recovered: %s", sessionDir),
				map[string]any{
					"session_dir":      sessionDir,
					"recovered_status": result.RecoveredStatus,
					"continued":        result.Continued,
					"repaired_files":   result.RepairedFiles,
					"checkpoint_ref":   result.CheckpointRef,
					"replay_ref":       result.ReplayRef,
					"reason":           result.Reason,
				},
			)
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	_ = command.MarkFlagRequired("session")
	return command
}

func mapRecoveryError(err error) error {
	if errors.Is(err, eventbus.ErrEventChainBroken) {
		return newCLIError("event_chain_broken", err.Error(), exitCodeRecoveryValidationFailed, err, nil)
	}
	if errors.Is(err, recovery.ErrRecoveryLayoutInvalid) {
		return newCLIError("recovery_layout_invalid", err.Error(), exitCodeRecoveryValidationFailed, err, nil)
	}
	return err
}
