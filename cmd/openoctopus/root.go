package main

import (
	"github.com/spf13/cobra"
)

func NewRootCommand() *cobra.Command {
	command := &cobra.Command{
		Use:          "openoctopus",
		SilenceUsage: true,
	}
	command.AddCommand(newValidateCommand())
	command.AddCommand(newRunCommand())
	return command
}
