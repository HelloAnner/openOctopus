package roleruntime

import (
	"errors"
	"fmt"
	"strings"

	"github.com/anner/openoctopus/internal/config/model"
)

var ErrUnsupportedExecutor = errors.New("unsupported executor")

func resolveExecutor(profile model.LLMProfile) (executor, error) {
	provider := strings.TrimSpace(strings.ToLower(profile.Provider))
	if provider == "deterministic" {
		return deterministicExecutor{}, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrUnsupportedExecutor, profile.Provider)
}

func parseRoleResult(output string) roleResult {
	status := ""
	summary := ""
	outputRefs := ""
	capture := false
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			capture = trimmed == "## role_result"
			continue
		}
		if !capture || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(trimmed, "- "), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "status":
			status = value
		case "summary":
			summary = value
		case "output_refs":
			outputRefs = value
		}
	}
	if status == "" {
		status = statusFailed
		summary = defaultString(summary, "missing role_result status")
	}
	if summary == "" {
		summary = strings.ToLower(status)
	}
	return roleResult{Status: status, Summary: summary, OutputRefs: outputRefs}
}

func mapConclusionToRoleStatus(status string) string {
	switch status {
	case "SUCCESS":
		return statusCompleted
	case "BLOCKED":
		return statusBlocked
	case "FAILED":
		return statusFailed
	default:
		return statusCompleted
	}
}
