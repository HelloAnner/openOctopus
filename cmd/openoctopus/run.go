package main

import (
	"fmt"
	"os"

	"github.com/anner/openoctopus/internal/artifact"
	configerrors "github.com/anner/openoctopus/internal/config/errors"
	configmodel "github.com/anner/openoctopus/internal/config/model"
	configservice "github.com/anner/openoctopus/internal/config/service"
	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/orchestrator"
	"github.com/anner/openoctopus/internal/roleruntime"
	"github.com/anner/openoctopus/internal/session"
	"github.com/anner/openoctopus/internal/tmux"
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
			return executeRunCommand(command, configPath, format, workspaceRoot, maxParallelRoles)
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "path to octopus.yaml")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	command.Flags().StringVar(&workspaceRoot, "workspace-root", "", "override runtime.workspace.root")
	command.Flags().IntVar(&maxParallelRoles, "max-parallel-roles", 0, "override runtime.scheduler.max_parallel_roles")
	_ = command.MarkFlagRequired("config")
	return command
}

func executeRunCommand(command *cobra.Command, configPath string, format string, workspaceRoot string, maxParallelRoles int) error {
	result, err := loadRunConfig(configPath, workspaceRoot, maxParallelRoles)
	if err != nil {
		return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
	}
	if len(result.Errors) != 0 {
		return writeRunValidationFailure(command, format, result.Errors)
	}
	return executeValidatedRun(command, configPath, format, result)
}

func loadRunConfig(configPath string, workspaceRoot string, maxParallelRoles int) (configservice.ValidateResult, error) {
	return configservice.LoadForValidate(configservice.LoadOptions{ConfigPath: configPath, FlagOverrides: buildFlagOverrides(workspaceRoot, maxParallelRoles)})
}

func executeValidatedRun(command *cobra.Command, configPath string, format string, result configservice.ValidateResult) error {
	createResult, err := session.Create(session.CreateOptions{Config: result.Config, ConfigPath: configPath, AppliedDefaults: result.AppliedDefaults})
	if err != nil {
		return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
	}
	tmuxResult, err := bootstrapTmuxIfNeeded(createResult, result.Config)
	if err != nil {
		cleanupRunFailure(createResult.SessionDir, tmuxResult)
		return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
	}
	if err := bootstrapRunDependencies(createResult, result.Config, tmuxResult); err != nil {
		return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", err)
	}
	return writeRunSuccess(command, format, createResult)
}

func writeRunValidationFailure(command *cobra.Command, format string, items []configerrors.ConfigError) error {
	if normalizeFormat(format) == "text" {
		if err := writeValidationTextErrors(command.ErrOrStderr(), items); err != nil {
			return err
		}
	}
	return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "run", newConfigValidationError(items))
}

func bootstrapTmuxIfNeeded(createResult session.CreateResult, config configmodel.RuntimeConfig) (tmux.BootstrapResult, error) {
	if !config.Runtime.Tmux.Enabled {
		return tmux.BootstrapResult{}, nil
	}
	service := tmux.NewService(createResult.SessionDir)
	return service.Bootstrap(tmux.BootstrapOptions{
		SessionID:      createResult.SessionID,
		RoleIDs:        roleIDs(config.Roles),
		SocketTemplate: config.Runtime.Tmux.SocketName,
		MainPaneRatio:  config.Runtime.Tmux.MainPaneRatio,
		RoleLayout:     config.Runtime.Tmux.RoleLayout,
	})
}

func roleIDs(roles []configmodel.RoleConfig) []string {
	values := make([]string, 0, len(roles))
	for _, role := range roles {
		values = append(values, role.ID)
	}
	return values
}

func bootstrapRunDependencies(createResult session.CreateResult, config configmodel.RuntimeConfig, tmuxResult tmux.BootstrapResult) error {
	if err := bootstrapEventBus(createResult, config); err != nil {
		cleanupRunFailure(createResult.SessionDir, tmuxResult)
		return err
	}
	if err := bootstrapArtifacts(createResult, tmuxResult); err != nil {
		return err
	}
	return bootstrapOrchestrator(createResult, tmuxResult)
}

func bootstrapEventBus(createResult session.CreateResult, config configmodel.RuntimeConfig) error {
	store := eventbus.NewStore(createResult.SessionDir)
	return store.Bootstrap(eventbus.BootstrapOptions{SessionID: createResult.SessionID, SessionDir: createResult.SessionDir, WorkflowID: config.Meta.WorkflowID, MetadataRef: "metadata.md"})
}

func bootstrapArtifacts(createResult session.CreateResult, tmuxResult tmux.BootstrapResult) error {
	if err := artifact.NewStore(createResult.SessionDir).Bootstrap(); err != nil {
		cleanupRunFailure(createResult.SessionDir, tmuxResult)
		return err
	}
	return nil
}

func bootstrapOrchestrator(createResult session.CreateResult, tmuxResult tmux.BootstrapResult) error {
	master := orchestrator.NewEngine(createResult.SessionDir)
	if err := master.Bootstrap(); err != nil {
		cleanupRunFailure(createResult.SessionDir, tmuxResult)
		return err
	}
	tick, err := master.Tick()
	if err != nil {
		cleanupRunFailure(createResult.SessionDir, tmuxResult)
		return err
	}
	if !roleRuntimeLoopEnabled() {
		return nil
	}
	if err := driveRoleRuntimeLoop(createResult.SessionDir, master, tick.WorkflowStatus); err != nil {
		cleanupRunFailure(createResult.SessionDir, tmuxResult)
		return err
	}
	return nil
}

func cleanupRunFailure(sessionDir string, tmuxResult tmux.BootstrapResult) {
	if tmuxResult.SessionName != "" {
		_ = tmux.NewService(sessionDir).KillSession(tmuxResult.SocketName, tmuxResult.SessionName)
	}
	_ = os.RemoveAll(sessionDir)
}

func writeRunSuccess(command *cobra.Command, format string, createResult session.CreateResult) error {
	return writeCommandSuccess(command.OutOrStdout(), command.ErrOrStderr(), format, "run", fmt.Sprintf("session created: %s", createResult.SessionDir), map[string]any{"session_id": createResult.SessionID, "session_dir": createResult.SessionDir})
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
