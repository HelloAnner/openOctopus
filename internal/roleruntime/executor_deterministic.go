package roleruntime

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type deterministicExecutor struct{}

func (deterministicExecutor) Execute(request ExecuteRequest) (ExecuteResult, error) {
	start := time.Now()
	status := deterministicStatus(request.Role.ID, request.TurnSeq)
	summary := fmt.Sprintf("deterministic result %s", status)
	outputRefs := deterministicOutputRefs(request.Role.ID)
	if status != "SUCCESS" {
		outputRefs = ""
	}
	raw := strings.Join([]string{
		"## role_result",
		fmt.Sprintf("- status: %s", status),
		fmt.Sprintf("- summary: %s", summary),
		fmt.Sprintf("- output_refs: %s", outputRefs),
	}, "\n")
	result := ExecuteResult{
		Provider:   "deterministic",
		Command:    "deterministic",
		ExitCode:   0,
		DurationMS: time.Since(start).Milliseconds(),
		Stdout:     raw,
		RawOutput:  raw,
	}
	if status == "FAILED" {
		result.ExitCode = 1
	}
	return result, nil
}

func deterministicStatus(roleID string, turnSeq int) string {
	keys := []string{
		"OPENOCTOPUS_DETERMINISTIC_RESULTS_" + sanitizeRoleEnvKey(roleID),
		"OPENOCTOPUS_DETERMINISTIC_RESULTS",
	}
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parts := strings.Split(value, ",")
		index := turnSeq - 1
		if index < 0 {
			index = 0
		}
		if index >= len(parts) {
			index = len(parts) - 1
		}
		return strings.TrimSpace(parts[index])
	}
	return "SUCCESS"
}

func deterministicOutputRefs(roleID string) string {
	keys := []string{
		"OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS_" + sanitizeRoleEnvKey(roleID),
		"OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS",
	}
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}
