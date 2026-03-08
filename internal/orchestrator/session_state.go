package orchestrator

import (
	"fmt"
	"strings"
)

func readSessionState(path string) (sessionState, error) {
	content, err := readFile(path)
	if err != nil {
		return sessionState{}, err
	}
	values := leadingValues(content)
	return sessionState{
		SessionID:          values["session_id"],
		Status:             values["status"],
		CurrentStageID:     values["current_stage_id"],
		CurrentRoleID:      values["current_role_id"],
		CheckpointSeq:      values["checkpoint_seq"],
		LastEvent:          values["last_event"],
		CreatedAt:          values["created_at"],
		UpdatedAt:          values["updated_at"],
		HumanMessageCursor: values["human_message_cursor"],
	}, nil
}

func writeSessionState(path string, state sessionState) error {
	lines := []string{
		"# Session State",
		"",
		fmt.Sprintf("- session_id: %s", state.SessionID),
		fmt.Sprintf("- status: %s", state.Status),
		fmt.Sprintf("- current_stage_id: %s", state.CurrentStageID),
		fmt.Sprintf("- current_role_id: %s", state.CurrentRoleID),
		fmt.Sprintf("- checkpoint_seq: %s", defaultString(state.CheckpointSeq, "0")),
		fmt.Sprintf("- last_event: %s", state.LastEvent),
		fmt.Sprintf("- created_at: %s", state.CreatedAt),
		fmt.Sprintf("- updated_at: %s", state.UpdatedAt),
		fmt.Sprintf("- human_message_cursor: %s", state.HumanMessageCursor),
	}
	return atomicWrite(path, []byte(strings.Join(lines, "\n")+"\n"))
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
