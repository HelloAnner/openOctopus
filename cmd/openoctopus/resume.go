package main

import (
	"fmt"
	"os"

	"github.com/anner/openoctopus/internal/humangate"
	"github.com/anner/openoctopus/internal/orchestrator"
	"github.com/spf13/cobra"
)

func newResumeCommand() *cobra.Command {
	var sessionRef string
	var roleID string
	command := &cobra.Command{
		Use:   "resume",
		Short: "恢复等待人工的会话",
		RunE: func(command *cobra.Command, _ []string) error {
			workingDir, err := os.Getwd()
			if err != nil {
				return err
			}
			sessionDir, err := humangate.ResolveSessionDir(sessionRef, workingDir)
			if err != nil {
				return err
			}
			if _, err := humangate.NewService(sessionDir).Resume(humangate.ResumeOptions{RoleID: roleID}); err != nil {
				return err
			}
			master := orchestrator.NewEngine(sessionDir)
			tick, err := master.Tick()
			if err != nil {
				return err
			}
			if roleRuntimeLoopEnabled() {
				if err := driveRoleRuntimeLoop(sessionDir, master, tick.WorkflowStatus); err != nil {
					return err
				}
			}
			_, _ = fmt.Fprintf(command.OutOrStdout(), "session resumed: %s\n", sessionDir)
			return nil
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&roleID, "role", "", "target role id")
	_ = command.MarkFlagRequired("session")
	return command
}
