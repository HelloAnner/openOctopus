package main

import (
	"fmt"
	"os"

	"github.com/anner/openoctopus/internal/artifact"
	"github.com/anner/openoctopus/internal/config/service"
	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/orchestrator"
	"github.com/anner/openoctopus/internal/roleruntime"
	"github.com/anner/openoctopus/internal/session"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	var configPath string
	var format string
	var workspaceRoot string
	var maxParallelRoles int
	command := &cobra.Command{
		Use:   "run",
		Short: "执行 OpenOctopus 工作流",
		RunE: func(command *cobra.Command, _ []string) error {
			result, err := service.LoadForValidate(service.LoadOptions{
				ConfigPath:    configPath,
				FlagOverrides: buildFlagOverrides(workspaceRoot, maxParallelRoles),
			})
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
			}
			if len(result.Errors) != 0 {
				if normalizeFormat(format) == "text" {
					if err := writeValidationTextErrors(command.ErrOrStderr(), result.Errors); err != nil {
						return err
					}
				}
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", newConfigValidationError(result.Errors))
			}
			createResult, err := session.Create(session.CreateOptions{
				Config:          result.Config,
				ConfigPath:      configPath,
				AppliedDefaults: result.AppliedDefaults,
			})
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
			}
			store := eventbus.NewStore(createResult.SessionDir)
			err = store.Bootstrap(eventbus.BootstrapOptions{
				SessionID:   createResult.SessionID,
				SessionDir:  createResult.SessionDir,
				WorkflowID:  result.Config.Meta.WorkflowID,
				MetadataRef: "metadata.md",
			})
			if err != nil {
				_ = os.RemoveAll(createResult.SessionDir)
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
			}
			artifactStore := artifact.NewStore(createResult.SessionDir)
			if err := artifactStore.Bootstrap(); err != nil {
				_ = os.RemoveAll(createResult.SessionDir)
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
			}
			master := orchestrator.NewEngine(createResult.SessionDir)
			if err := master.Bootstrap(); err != nil {
				_ = os.RemoveAll(createResult.SessionDir)
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
			}
			tick, err := master.Tick()
			if err != nil {
				_ = os.RemoveAll(createResult.SessionDir)
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
			}
			if roleRuntimeLoopEnabled() {
				if err := driveRoleRuntimeLoop(createResult.SessionDir, master, tick.WorkflowStatus); err != nil {
					_ = os.RemoveAll(createResult.SessionDir)
					return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
				}
			}
			return writeCommandSuccess(
				command.OutOrStdout(),
				command.ErrOrStderr(),
				format,
				"run",
				fmt.Sprintf("session created: %s", createResult.SessionDir),
				map[string]any{"session_id": createResult.SessionID, "session_dir": createResult.SessionDir},
			)
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "path to octopus.yaml")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	command.Flags().StringVar(&workspaceRoot, "workspace-root", "", "override runtime.workspace.root")
	command.Flags().IntVar(&maxParallelRoles, "max-parallel-roles", 0, "override runtime.scheduler.max_parallel_roles")
	_ = command.MarkFlagRequired("config")
	return command
}

func driveRoleRuntimeLoop(sessionDir string, master *orchestrator.Engine, workflowStatus string) error {
	if isTerminalWorkflow(workflowStatus) {
		return nil
	}
	runtimeEngine := roleruntime.NewEngine(sessionDir)
	for index := 0; index < 8; index++ {
		roleTick, err := runtimeEngine.TickAll()
		if err != nil {
			return err
		}
		if !roleTick.Progressed {
			return nil
		}
		next, err := master.Tick()
		if err != nil {
			return err
		}
		workflowStatus = next.WorkflowStatus
		if isTerminalWorkflow(workflowStatus) {
			return nil
		}
	}
	return nil
}

func isTerminalWorkflow(status string) bool {
	switch status {
	case "COMPLETED", "FAILED", "WAITING_HUMAN":
		return true
	default:
		return false
	}
}

func buildFlagOverrides(workspaceRoot string, maxParallelRoles int) map[string]any {
	overrides := make(map[string]any)
	if workspaceRoot != "" {
		overrides["runtime.workspace.root"] = workspaceRoot
	}
	if maxParallelRoles > 0 {
		overrides["runtime.scheduler.max_parallel_roles"] = maxParallelRoles
	}
	return overrides
}

func roleRuntimeLoopEnabled() bool {
	return os.Getenv("OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP") != "1"
}
