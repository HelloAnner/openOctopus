package orchestrator

import (
	"fmt"
	"strings"
)

func renderTaskBoard(schedule Schedule) []byte {
	groups := map[string][]string{
		"Todo":    {},
		"Doing":   {},
		"Done":    {},
		"Blocked": {},
	}
	for _, item := range schedule.Stages {
		entry := fmt.Sprintf("- %s (%s)", item.StageID, item.Status)
		switch item.Status {
		case stageStatusCompleted:
			groups["Done"] = append(groups["Done"], entry)
		case stageStatusBlocked, stageStatusFailed:
			groups["Blocked"] = append(groups["Blocked"], entry)
		case stageStatusDispatched:
			groups["Doing"] = append(groups["Doing"], entry)
		default:
			groups["Todo"] = append(groups["Todo"], entry)
		}
	}
	lines := []string{"# Task Board", "", "## Todo"}
	lines = append(lines, ensureEntries(groups["Todo"])...)
	lines = append(lines, "", "## Doing")
	lines = append(lines, ensureEntries(groups["Doing"])...)
	lines = append(lines, "", "## Done")
	lines = append(lines, ensureEntries(groups["Done"])...)
	lines = append(lines, "", "## Blocked")
	lines = append(lines, ensureEntries(groups["Blocked"])...)
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderTaskGraph(graph Graph) []byte {
	lines := []string{"graph TD"}
	for _, stageID := range graph.Order {
		node := graph.Stages[stageID]
		to := node.NextStageID
		if to == "" {
			to = "END"
		}
		if to == "__END__" {
			to = "END"
		}
		lines = append(lines, fmt.Sprintf("    %s --> %s", stageID, to))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderGlobalProgress(schedule Schedule) []byte {
	completed := 0
	for _, item := range schedule.Stages {
		if item.Status == stageStatusCompleted {
			completed++
		}
	}
	lines := []string{
		"# Global Progress",
		"",
		fmt.Sprintf("- workflow_status: %s", schedule.WorkflowStatus),
		fmt.Sprintf("- completed_stage_count: %d", completed),
		fmt.Sprintf("- active_dispatch_count: %d", schedule.ActiveDispatchCount),
		fmt.Sprintf("- updated_at: %s", schedule.UpdatedAt),
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderBlockers(summary string) []byte {
	if summary == "" {
		summary = "clear"
	}
	lines := []string{
		"# Blockers",
		"",
		fmt.Sprintf("- summary: %s", summary),
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func ensureEntries(entries []string) []string {
	if len(entries) == 0 {
		return []string{"- none"}
	}
	return entries
}
