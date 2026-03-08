package main

import (
	"fmt"

	"github.com/anner/openoctopus/internal/config/service"
	"github.com/spf13/cobra"
)

func newValidateCommand() *cobra.Command {
	var configPath string
	var format string
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
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "validate", err)
			}
			if len(result.Errors) != 0 {
				if normalizeFormat(format) == "text" {
					if err := writeValidationTextErrors(command.ErrOrStderr(), result.Errors); err != nil {
						return err
					}
				}
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "validate", newConfigValidationError(result.Errors))
			}
			return writeCommandSuccess(
				command.OutOrStdout(),
				command.ErrOrStderr(),
				format,
				"validate",
				fmt.Sprintf("config is valid (%d defaults applied)", len(result.AppliedDefaults)),
				map[string]any{"applied_defaults_count": len(result.AppliedDefaults)},
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
