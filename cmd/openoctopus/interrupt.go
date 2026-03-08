package main

import (
	"fmt"
	"os"

	"github.com/anner/openoctopus/internal/humangate"
	"github.com/spf13/cobra"
)

func newInterruptCommand() *cobra.Command {
	var sessionRef string
	var roleID string
	var reason string
	command := &cobra.Command{
		Use:   "interrupt",
		Short: "中断指定角色并进入人工等待",
		RunE: func(command *cobra.Command, _ []string) error {
			workingDir, err := os.Getwd()
			if err != nil {
				return err
			}
			sessionDir, err := humangate.ResolveSessionDir(sessionRef, workingDir)
			if err != nil {
				return err
			}
			record, err := humangate.NewService(sessionDir).Interrupt(humangate.InterruptOptions{RoleID: roleID, Reason: reason})
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(command.OutOrStdout(), "interrupt requested: %s\n", record.InterruptID)
			return nil
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&roleID, "role", "", "target role id")
	command.Flags().StringVar(&reason, "reason", "", "interrupt reason")
	_ = command.MarkFlagRequired("session")
	_ = command.MarkFlagRequired("role")
	_ = command.MarkFlagRequired("reason")
	return command
}
