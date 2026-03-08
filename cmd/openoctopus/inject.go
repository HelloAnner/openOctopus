package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/anner/openoctopus/internal/humangate"
	"github.com/spf13/cobra"
)

func newInjectCommand() *cobra.Command {
	var sessionRef string
	var roleID string
	var message string
	var inputPath string
	var format string
	command := &cobra.Command{
		Use:   "inject",
		Short: "追加人工补充消息",
		RunE: func(command *cobra.Command, _ []string) error {
			content, err := loadInjectedMessage(message, inputPath)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "inject", err)
			}
			sessionDir, err := resolveCommandSessionDir(sessionRef)
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "inject", mapStatusError(err))
			}
			result, err := humangate.NewService(sessionDir).Inject(humangate.InjectOptions{RoleID: roleID, Message: content})
			if err != nil {
				return renderCommandError(command.OutOrStdout(), command.ErrOrStderr(), format, "inject", err)
			}
			return writeCommandSuccess(
				command.OutOrStdout(),
				command.ErrOrStderr(),
				format,
				"inject",
				fmt.Sprintf("message appended: %s", result.MessageID),
				map[string]any{"message_id": result.MessageID, "path": result.Path, "session_dir": sessionDir},
			)
		},
	}
	command.Flags().StringVar(&sessionRef, "session", "", "session id or session dir")
	command.Flags().StringVar(&format, "format", "text", "output format: text|json")
	command.Flags().StringVar(&roleID, "role", "", "target role id")
	command.Flags().StringVar(&message, "message", "", "inline human message")
	command.Flags().StringVar(&inputPath, "input", "", "input markdown path")
	_ = command.MarkFlagRequired("session")
	return command
}

func loadInjectedMessage(message string, inputPath string) (string, error) {
	inline := strings.TrimSpace(message)
	file := strings.TrimSpace(inputPath)
	if inline == "" && file == "" {
		return "", fmt.Errorf("message or input is required")
	}
	if inline != "" && file != "" {
		return "", fmt.Errorf("message and input cannot be used together")
	}
	if file == "" {
		return inline, nil
	}
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
