package main

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	command := &cobra.Command{
		Use:          "openoctopus",
		SilenceErrors: true,
		SilenceUsage: true,
	}
	command.AddCommand(newValidateCommand())
	command.AddCommand(newRunCommand())
	command.AddCommand(newStatusCommand())
	command.AddCommand(newInterruptCommand())
	command.AddCommand(newInterruptAllCommand())
	command.AddCommand(newInjectCommand())
	command.AddCommand(newResumeCommand())
	return command
}
