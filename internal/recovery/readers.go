/*
Package recovery readers 负责读取 recovery 需要的文档协议。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readMetadata(path string) (metadataDoc, error) {
	content, err := readFile(path)
	if err != nil {
		return metadataDoc{}, err
	}
	values := leadingValues(content)
	return metadataDoc{SessionID: values["session_id"], CreatedAt: values["created_at"]}, nil
}

func readSessionState(path string) (sessionStateDoc, bool, error) {
	content, err := readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return sessionStateDoc{}, false, nil
		}
		return sessionStateDoc{}, false, err
	}
	values := leadingValues(content)
	return sessionStateDoc{
		SessionID:          values["session_id"],
		Status:             values["status"],
		CurrentStageID:     values["current_stage_id"],
		CurrentRoleID:      values["current_role_id"],
		CheckpointSeq:      atoi(values["checkpoint_seq"]),
		LastEvent:          values["last_event"],
		CreatedAt:          values["created_at"],
		UpdatedAt:          values["updated_at"],
		HumanMessageCursor: values["human_message_cursor"],
	}, true, nil
}

func renderSessionState(state sessionStateDoc) string {
	lines := []string{
		"# Session State",
		"",
		"- session_id: " + state.SessionID,
		"- status: " + state.Status,
		"- current_stage_id: " + state.CurrentStageID,
		"- current_role_id: " + state.CurrentRoleID,
		"- checkpoint_seq: " + strconvValue(state.CheckpointSeq),
		"- last_event: " + state.LastEvent,
		"- created_at: " + state.CreatedAt,
		"- updated_at: " + state.UpdatedAt,
		"- human_message_cursor: " + state.HumanMessageCursor,
	}
	return strings.Join(lines, "\n") + "\n"
}

func readSchedule(path string) (scheduleDoc, error) {
	content, err := readFile(path)
	if err != nil {
		return scheduleDoc{}, err
	}
	values := leadingValues(content)
	schedule := scheduleDoc{
		ScheduleVersion: atoi(values["schedule_version"]),
		WorkflowStatus:  values["workflow_status"],
		Stages:          make([]stageDoc, 0),
	}
	for _, block := range splitBlocks(content, "## stage: ") {
		stageValues := blockValues(block)
		schedule.Stages = append(schedule.Stages, stageDoc{
			StageID:           stageValues["stage_id"],
			RoleID:            stageValues["role_id"],
			Status:            stageValues["status"],
			LastConclusionRef: stageValues["last_conclusion_ref"],
		})
	}
	return schedule, nil
}

func readBlockerSummary(path string) (string, error) {
	content, err := readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "clear", nil
		}
		return "", err
	}
	return defaultString(leadingValues(content)["summary"], "clear"), nil
}

func renderBlockers(summary string) string {
	lines := []string{
		"# Blockers",
		"",
		"- summary: " + defaultString(summary, "clear"),
		"- updated_at: " + utcNow(),
	}
	return strings.Join(lines, "\n") + "\n"
}

func readConclusionSummary(rolesDir string, relativeRef string) string {
	trimmed := strings.TrimSpace(relativeRef)
	if trimmed == "" {
		return ""
	}
	path := filepath.Join(filepath.Dir(rolesDir), filepath.FromSlash(trimmed))
	content, err := readFile(path)
	if err != nil {
		return ""
	}
	return leadingValues(content)["summary"]
}

func strconvValue(value int) string {
	if value <= 0 {
		return "0"
	}
	return strconv.Itoa(value)
}
