/*
Package main helpers 提供 CLI 命令层共享支撑。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	clisupport "github.com/anner/openoctopus/internal/cli"
	configerrors "github.com/anner/openoctopus/internal/config/errors"
)

func resolveCommandSessionDir(sessionRef string) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return clisupport.NewService(workingDir).ResolveSessionDir(sessionRef)
}

func newStatusService() (*clisupport.Service, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return clisupport.NewService(workingDir), nil
}

func writeValidationTextErrors(writer io.Writer, items []configerrors.ConfigError) error {
	for _, item := range items {
		if _, err := fmt.Fprintf(writer, "[%s] %s %s (%s)\n", item.Category, item.Path, item.Message, item.RuleID); err != nil {
			return err
		}
	}
	return nil
}

func newConfigValidationError(items []configerrors.ConfigError) *CLIError {
	return newCLIError(
		"config_validation_failed",
		"config validation failed",
		exitCodeConfigValidationFailed,
		nil,
		validationErrorDetails(items),
	)
}

func validationErrorDetails(items []configerrors.ConfigError) []map[string]string {
	details := make([]map[string]string, 0, len(items))
	for _, item := range items {
		details = append(details, map[string]string{
			"code":     item.Code,
			"category": string(item.Category),
			"path":     item.Path,
			"message":  item.Message,
			"rule_id":  item.RuleID,
		})
	}
	return details
}

func renderStatusText(summary clisupport.StatusSummary) string {
	lines := []string{
		fmt.Sprintf("session: %s", summary.SessionID),
		fmt.Sprintf("session dir: %s", summary.SessionDir),
		fmt.Sprintf("workflow: %s (%s)", summary.WorkflowID, summary.WorkflowName),
		fmt.Sprintf("status: %s", summary.WorkflowStatus),
		fmt.Sprintf("current stage: %s", emptyFallback(summary.CurrentStageID)),
		fmt.Sprintf("current role: %s", emptyFallback(summary.CurrentRoleID)),
		fmt.Sprintf("schedule version: %d", summary.ScheduleVersion),
		fmt.Sprintf("active dispatch count: %d", summary.ActiveDispatchCount),
		fmt.Sprintf("blocker: %s", emptyFallback(summary.BlockerSummary)),
		fmt.Sprintf("updated at: %s", emptyFallback(summary.UpdatedAt)),
	}
	return strings.Join(lines, "\n")
}

func emptyFallback(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}
