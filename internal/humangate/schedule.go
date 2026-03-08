/*
Package humangate schedule 提供 resume 与 interrupt-all 所需的调度文件解析能力。
Author: Anner
Created on 2026/3/8
*/
package humangate

import (
	"fmt"
	"strings"
)

func readSchedule(path string) (scheduleState, error) {
	content, err := readFile(path)
	if err != nil {
		return scheduleState{}, err
	}
	values := leadingValues(content)
	schedule := scheduleState{
		ScheduleVersion:     atoi(values["schedule_version"]),
		WorkflowStatus:      values["workflow_status"],
		ActiveDispatchCount: atoi(values["active_dispatch_count"]),
		UpdatedAt:           values["updated_at"],
		Stages:              make([]stageEntry, 0),
	}
	for _, block := range splitBlocks(content, "## stage: ") {
		stage := blockValues(block)
		schedule.Stages = append(schedule.Stages, stageEntry{
			StageID:           stage["stage_id"],
			StageName:         stage["stage_name"],
			RoleID:            stage["role_id"],
			Status:            stage["status"],
			Attempt:           atoi(stage["attempt"]),
			LastTaskID:        stage["last_task_id"],
			LastConclusionRef: stage["last_conclusion_ref"],
			NextStageID:       stage["next_stage_id"],
			UpdatedAt:         stage["updated_at"],
		})
	}
	return schedule, nil
}

func writeSchedule(path string, schedule scheduleState) error {
	lines := []string{
		"# Master Schedule",
		"",
		fmt.Sprintf("- schedule_version: %d", schedule.ScheduleVersion),
		fmt.Sprintf("- workflow_status: %s", schedule.WorkflowStatus),
		fmt.Sprintf("- active_dispatch_count: %d", schedule.ActiveDispatchCount),
		fmt.Sprintf("- updated_at: %s", schedule.UpdatedAt),
	}
	for _, stage := range schedule.Stages {
		lines = append(lines,
			"",
			fmt.Sprintf("## stage: %s", stage.StageID),
			fmt.Sprintf("- stage_id: %s", stage.StageID),
			fmt.Sprintf("- stage_name: %s", stage.StageName),
			fmt.Sprintf("- role_id: %s", stage.RoleID),
			fmt.Sprintf("- status: %s", stage.Status),
			fmt.Sprintf("- attempt: %d", stage.Attempt),
			fmt.Sprintf("- last_task_id: %s", stage.LastTaskID),
			fmt.Sprintf("- last_conclusion_ref: %s", stage.LastConclusionRef),
			fmt.Sprintf("- next_stage_id: %s", stage.NextStageID),
			fmt.Sprintf("- updated_at: %s", stage.UpdatedAt),
		)
	}
	return atomicWrite(path, []byte(strings.Join(lines, "\n")+"\n"))
}

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

func writeBlockers(path string, summary string) error {
	content := strings.Join([]string{
		"# Blockers",
		"",
		fmt.Sprintf("- summary: %s", defaultString(summary, "clear")),
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}, "\n") + "\n"
	return atomicWrite(path, []byte(content))
}

func appendDecision(path string, name string, decisionType string, details ...string) error {
	lines := []string{fmt.Sprintf("## decision: %s", name), fmt.Sprintf("- type: %s", decisionType)}
	lines = append(lines, details...)
	lines = append(lines, fmt.Sprintf("- at: %s", utcNow()))
	return appendBlock(path, "# Decision Log", lines)
}

func interruptTargets(schedule scheduleState) []string {
	roles := make([]string, 0)
	seen := make(map[string]struct{})
	for _, stage := range schedule.Stages {
		if !interruptibleStage(stage.Status) {
			continue
		}
		if _, exists := seen[stage.RoleID]; exists {
			continue
		}
		seen[stage.RoleID] = struct{}{}
		roles = append(roles, stage.RoleID)
	}
	return roles
}

func interruptibleStage(status string) bool {
	switch status {
	case "READY", "DISPATCHED", "RETRY_PENDING", "BLOCKED":
		return true
	default:
		return false
	}
}

func requeueBlockedStages(schedule *scheduleState, roleID string) int {
	updated := 0
	for index := range schedule.Stages {
		stage := &schedule.Stages[index]
		if stage.Status != "BLOCKED" {
			continue
		}
		if roleID != "" && stage.RoleID != roleID {
			continue
		}
		stage.Status = "RETRY_PENDING"
		stage.Attempt++
		stage.UpdatedAt = utcNow()
		updated++
	}
	if updated > 0 {
		schedule.WorkflowStatus = "READY"
		schedule.UpdatedAt = utcNow()
		schedule.ActiveDispatchCount = countActiveDispatches(schedule.Stages)
	}
	return updated
}

func countActiveDispatches(stages []stageEntry) int {
	count := 0
	for _, stage := range stages {
		if stage.Status == "DISPATCHED" {
			count++
		}
	}
	return count
}
