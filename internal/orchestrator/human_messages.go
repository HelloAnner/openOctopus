package orchestrator

import (
	"fmt"
	"strings"
)

func parseHumanMessages(content string) ([]HumanMessage, error) {
	if isPlaceholder(content) {
		return nil, nil
	}
	blocks := splitBlocks(content, "## message: ")
	messages := make([]HumanMessage, 0, len(blocks))
	for _, block := range blocks {
		values := blockValues(block)
		if values["message_id"] == "" || values["created_at"] == "" {
			return nil, ErrPlannerNotInitialized
		}
		contentLines := make([]string, 0)
		collect := false
		for _, line := range block {
			if strings.HasPrefix(line, "### content") {
				collect = true
				continue
			}
			if collect {
				contentLines = append(contentLines, strings.TrimSpace(line))
			}
		}
		messages = append(messages, HumanMessage{
			MessageID: values["message_id"],
			Source:    values["source"],
			CreatedAt: values["created_at"],
			Content:   strings.TrimSpace(strings.Join(contentLines, "\n")),
		})
	}
	return messages, nil
}

func renderRequirementSnapshot(snapshot RequirementSnapshot) []byte {
	lines := []string{
		"# Requirement Snapshot",
		"",
		fmt.Sprintf("- snapshot_version: %d", snapshot.SnapshotVersion),
		fmt.Sprintf("- workflow_id: %s", snapshot.WorkflowID),
		fmt.Sprintf("- workflow_name: %s", snapshot.WorkflowName),
		fmt.Sprintf("- human_message_cursor: %s", snapshot.HumanMessageCursor),
		fmt.Sprintf("- source_message_count: %d", snapshot.SourceMessageCount),
		fmt.Sprintf("- planner_status: %s", snapshot.PlannerStatus),
		fmt.Sprintf("- updated_at: %s", snapshot.UpdatedAt),
		"",
		"## workflow_summary",
		snapshot.WorkflowSummary,
		"",
		"## latest_messages",
		snapshot.LatestMessages,
		"",
		"## current_dispatch_brief",
		snapshot.DispatchBrief,
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func readRequirementSnapshot(path string) (RequirementSnapshot, error) {
	content, err := readFile(path)
	if err != nil {
		return RequirementSnapshot{}, err
	}
	if isPlaceholder(content) {
		return RequirementSnapshot{}, ErrPlannerNotInitialized
	}
	values := leadingValues(content)
	return RequirementSnapshot{
		SnapshotVersion:    atoi(values["snapshot_version"]),
		WorkflowID:         values["workflow_id"],
		WorkflowName:       values["workflow_name"],
		HumanMessageCursor: values["human_message_cursor"],
		SourceMessageCount: atoi(values["source_message_count"]),
		PlannerStatus:      values["planner_status"],
		UpdatedAt:          values["updated_at"],
	}, nil
}

func writeRequirementSnapshot(path string, snapshot RequirementSnapshot) error {
	return atomicWrite(path, renderRequirementSnapshot(snapshot))
}
