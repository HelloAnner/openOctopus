package main

import (
	"fmt"
	"os"

	"github.com/anner/openoctopus/internal/humangate"
	"github.com/spf13/cobra"
)

func newInterruptAllCommand() *cobra.Command {
	var sessionRef string
	var reason string
	command := &cobra.Command{
		Use:   "interrupt-all",
		Short: "中断全部未完成角色并等待人工",
		RunE: func(command *cobra.Command, _ []string) error {
			workingDir, err := os.Getwd()
			if err != nil {
				return err
			}
			sessionDir, err := humangate.ResolveSessionDir(sessionRef, workingDir)
			if err != nil {
				return err
			}
			result, err := humangate.NewService(sessionDir).InterruptAll(humangate.InterruptAllOptions{Reason: reason})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(command.OutOrStdout(), "interrupt-all requested: %d\n", result.RequestedCount)
			return nil
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&reason, "reason", "", "interrupt reason")
	_ = command.MarkFlagRequired("session")
	_ = command.MarkFlagRequired("reason")
	return command
}
