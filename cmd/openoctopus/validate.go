package main

import (
	"errors"
	"fmt"

	"github.com/anner/openoctopus/internal/config/service"
	"github.com/spf13/cobra"
)

func newValidateCommand() *cobra.Command {
	var configPath string
	var workspaceRoot string
	var maxParallelRoles int
	command := &cobra.Command{
		Use:   "validate",
		Short: "校验 OpenOctopus 配置",
		RunE: func(command *cobra.Command, _ []string) error {
			result, err := service.LoadForValidate(service.LoadOptions{
				ConfigPath: configPath,
				FlagOverrides: buildFlagOverrides(workspaceRoot, maxParallelRoles),
			})
			if err != nil {
				return err
			}
			if len(result.Errors) == 0 {
				fmt.Fprintf(command.OutOrStdout(), "config is valid (%d defaults applied)\n", len(result.AppliedDefaults))
				return nil
			}
			for _, item := range result.Errors {
				fmt.Fprintf(command.ErrOrStderr(), "[%s] %s %s (%s)\n", item.Category, item.Path, item.Message, item.RuleID)
			}
			return errors.New("config validation failed")
		},
	}
	command.Flags().StringVar(&configPath, "config", "", "path to octopus.yaml")
	command.Flags().StringVar(&workspaceRoot, "workspace-root", "", "override runtime.workspace.root")
	command.Flags().IntVar(&maxParallelRoles, "max-parallel-roles", 0, "override runtime.scheduler.max_parallel_roles")
	_ = command.MarkFlagRequired("config")
	return command
}
