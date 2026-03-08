package roleruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func roleDirPath(root string, roleID string) string {
	return filepath.Join(root, roleID)
}

func statePath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "state.md")
}

func resetPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "session.reset.md")
}

func heartbeatPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "heartbeat.md")
}

func outboxPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "outbox.md")
}

func conclusionPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "conclusion.md")
}

func eventsPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "events.md")
}

func turnsDirPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "turns")
}

func contextPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "context.md")
}

func inboxPath(root string, roleID string) string {
	return filepath.Join(roleDirPath(root, roleID), "inbox.md")
}

func ensureRoleLayout(root string, roleID string) error {
	return atomicWrite(eventsPath(root, roleID), []byte(defaultRoleEvents(readMarkdown(eventsPath(root, roleID)))))
}

func ensureRoleFiles(root string, roleID string) error {
	if err := atomicWriteIfMissing(statePath(root, roleID), []byte(renderState(defaultRoleState(roleID)))); err != nil {
		return err
	}
	if err := atomicWriteIfMissing(resetPath(root, roleID), []byte(renderReset(defaultRoleReset()))); err != nil {
		return err
	}
	if err := atomicWriteIfMissing(eventsPath(root, roleID), []byte(defaultRoleEvents(""))); err != nil {
		return err
	}
	return os.MkdirAll(turnsDirPath(root, roleID), 0o755)
}

func atomicWriteIfMissing(path string, content []byte) error {
	existing, err := readFile(path)
	if err == nil && strings.TrimSpace(existing) != "" {
		return nil
	}
	return atomicWrite(path, content)
}

func defaultRoleState(roleID string) roleState {
	return roleState{RoleID: roleID, SessionGeneration: 1, Status: statusIdle, UpdatedAt: utcNow()}
}

func defaultRoleReset() roleReset {
	return roleReset{SessionGeneration: 1, Status: resetStatusIdle}
}

func defaultRoleEvents(_ string) string {
	return "# Role Events\n"
}

func readRoleState(root string, roleID string) (roleState, error) {
	content, err := readFile(statePath(root, roleID))
	if err != nil {
		return defaultRoleState(roleID), nil
	}
	values := leadingValues(content)
	return roleState{
		RoleID:               roleID,
		SessionGeneration:    maxInt(1, atoi(values["session_generation"])),
		Status:               defaultString(values["status"], statusIdle),
		CurrentStageID:       values["current_stage_id"],
		CurrentTaskID:        values["current_task_id"],
		CurrentTurnSeq:       atoi(values["current_turn_seq"]),
		ContextVersion:       atoi(values["context_version"]),
		InboxVersion:         atoi(values["inbox_version"]),
		LastConsumedEventID:  values["last_consumed_event_id"],
		LastConclusionStatus: values["last_conclusion_status"],
		ExecutorProvider:     values["executor_provider"],
		UpdatedAt:            values["updated_at"],
	}, nil
}

func writeRoleState(root string, state roleState) error {
	state.UpdatedAt = utcNow()
	return atomicWrite(statePath(root, state.RoleID), []byte(renderState(state)))
}

func readRoleReset(root string, roleID string) (roleReset, error) {
	content, err := readFile(resetPath(root, roleID))
	if err != nil {
		return defaultRoleReset(), nil
	}
	values := leadingValues(content)
	return roleReset{
		SessionGeneration: maxInt(1, atoi(values["session_generation"])),
		Status:            defaultString(values["status"], resetStatusIdle),
		RequestedBy:       values["requested_by"],
		Reason:            values["reason"],
		RequestedAt:       values["requested_at"],
		AppliedAt:         values["applied_at"],
		LastClearedTaskID: values["last_cleared_task_id"],
	}, nil
}

func writeRoleReset(root string, roleID string, reset roleReset) error {
	return atomicWrite(resetPath(root, roleID), []byte(renderReset(reset)))
}

func writeHeartbeat(root string, heartbeat roleHeartbeat) error {
	heartbeat.UpdatedAt = utcNow()
	return atomicWrite(heartbeatPath(root, heartbeat.RoleID), []byte(renderHeartbeat(heartbeat)))
}

func writeOutbox(root string, outbox roleOutbox) error {
	outbox.UpdatedAt = utcNow()
	return atomicWrite(outboxPath(root, outbox.RoleID), []byte(renderOutbox(outbox)))
}

func writeConclusion(root string, roleID string, inbox roleInbox, result roleResult) error {
	lines := []string{
		"# Role Conclusion",
		"",
		fmt.Sprintf("- role_id: %s", roleID),
		fmt.Sprintf("- stage_id: %s", inbox.StageID),
		fmt.Sprintf("- task_id: %s", inbox.TaskID),
		fmt.Sprintf("- status: %s", result.Status),
		fmt.Sprintf("- summary: %s", result.Summary),
		fmt.Sprintf("- output_refs: %s", result.OutputRefs),
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}
	return atomicWrite(conclusionPath(root, roleID), []byte(strings.Join(lines, "\n")+"\n"))
}

func renderState(state roleState) string {
	lines := []string{
		"# Role State",
		"",
		fmt.Sprintf("- role_id: %s", state.RoleID),
		fmt.Sprintf("- session_generation: %d", maxInt(1, state.SessionGeneration)),
		fmt.Sprintf("- status: %s", defaultString(state.Status, statusIdle)),
		fmt.Sprintf("- current_stage_id: %s", state.CurrentStageID),
		fmt.Sprintf("- current_task_id: %s", state.CurrentTaskID),
		fmt.Sprintf("- current_turn_seq: %d", state.CurrentTurnSeq),
		fmt.Sprintf("- context_version: %d", state.ContextVersion),
		fmt.Sprintf("- inbox_version: %d", state.InboxVersion),
		fmt.Sprintf("- last_consumed_event_id: %s", state.LastConsumedEventID),
		fmt.Sprintf("- last_conclusion_status: %s", state.LastConclusionStatus),
		fmt.Sprintf("- executor_provider: %s", state.ExecutorProvider),
		fmt.Sprintf("- updated_at: %s", state.UpdatedAt),
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderReset(reset roleReset) string {
	lines := []string{
		"# Session Reset",
		"",
		fmt.Sprintf("- session_generation: %d", maxInt(1, reset.SessionGeneration)),
		fmt.Sprintf("- status: %s", defaultString(reset.Status, resetStatusIdle)),
		fmt.Sprintf("- requested_by: %s", reset.RequestedBy),
		fmt.Sprintf("- reason: %s", reset.Reason),
		fmt.Sprintf("- requested_at: %s", reset.RequestedAt),
		fmt.Sprintf("- applied_at: %s", reset.AppliedAt),
		fmt.Sprintf("- last_cleared_task_id: %s", reset.LastClearedTaskID),
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderHeartbeat(heartbeat roleHeartbeat) string {
	lines := []string{
		"# Role Heartbeat",
		"",
		fmt.Sprintf("- heartbeat_version: %d", heartbeat.HeartbeatVersion),
		fmt.Sprintf("- role_id: %s", heartbeat.RoleID),
		fmt.Sprintf("- status: %s", heartbeat.Status),
		fmt.Sprintf("- current_task_id: %s", heartbeat.CurrentTaskID),
		fmt.Sprintf("- current_turn_seq: %d", heartbeat.CurrentTurnSeq),
		fmt.Sprintf("- last_seen_at: %s", heartbeat.LastSeenAt),
		fmt.Sprintf("- expire_at: %s", heartbeat.ExpireAt),
		fmt.Sprintf("- session_generation: %d", maxInt(1, heartbeat.SessionGeneration)),
		fmt.Sprintf("- updated_at: %s", heartbeat.UpdatedAt),
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderOutbox(outbox roleOutbox) string {
	lines := []string{
		"# Role Outbox",
		"",
		fmt.Sprintf("- outbox_version: %d", outbox.OutboxVersion),
		fmt.Sprintf("- role_id: %s", outbox.RoleID),
		fmt.Sprintf("- stage_id: %s", outbox.StageID),
		fmt.Sprintf("- task_id: %s", outbox.TaskID),
		fmt.Sprintf("- turn_seq: %d", outbox.TurnSeq),
		fmt.Sprintf("- status: %s", outbox.Status),
		fmt.Sprintf("- conclusion_ref: %s", outbox.ConclusionRef),
		fmt.Sprintf("- turn_output_ref: %s", outbox.TurnOutputRef),
		fmt.Sprintf("- updated_at: %s", outbox.UpdatedAt),
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildHeartbeat(roleID string, state roleState, timeoutSeconds int) roleHeartbeat {
	now := time.Now().UTC()
	return roleHeartbeat{
		HeartbeatVersion: state.CurrentTurnSeq,
		RoleID:           roleID,
		Status:           state.Status,
		CurrentTaskID:    state.CurrentTaskID,
		CurrentTurnSeq:   state.CurrentTurnSeq,
		LastSeenAt:       now.Format(time.RFC3339),
		ExpireAt:         now.Add(time.Duration(timeoutSeconds) * time.Second).Format(time.RFC3339),
		SessionGeneration: state.SessionGeneration,
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
