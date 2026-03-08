package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/anner/openoctopus/internal/config/service"
	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/session"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	var configPath string
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
				return err
			}
			if len(result.Errors) != 0 {
				for _, item := range result.Errors {
					fmt.Fprintf(command.ErrOrStderr(), "[%s] %s %s (%s)\n", item.Category, item.Path, item.Message, item.RuleID)
				}
				return errors.New("config validation failed")
			}
			createResult, err := session.Create(session.CreateOptions{
				Config:          result.Config,
				ConfigPath:      configPath,
				AppliedDefaults: result.AppliedDefaults,
			})
			if err != nil {
				return err
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
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), "session created: %s\n", createResult.SessionDir)
			return nil
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "path to octopus.yaml")
	command.Flags().StringVar(&workspaceRoot, "workspace-root", "", "override runtime.workspace.root")
	command.Flags().IntVar(&maxParallelRoles, "max-parallel-roles", 0, "override runtime.scheduler.max_parallel_roles")
	_ = command.MarkFlagRequired("config")
	return command
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
