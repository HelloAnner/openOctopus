/*
Package cli 提供 CLI 展示层只读支撑能力。
Author: Anner
Created on 2026/3/8
*/
package cli

import "errors"

var ErrSessionNotFound = errors.New("session not found")

// Service 提供 session 解析与状态读取能力。
type Service struct {
	workingDir string
}

// StatusSummary 描述 CLI status 命令的聚合输出。
type StatusSummary struct {
	SessionID           string `json:"session_id"`
	SessionDir          string `json:"session_dir"`
	WorkflowID          string `json:"workflow_id"`
	WorkflowName        string `json:"workflow_name"`
	WorkflowStatus      string `json:"workflow_status"`
	CurrentStageID      string `json:"current_stage_id"`
	CurrentRoleID       string `json:"current_role_id"`
	ScheduleVersion     int    `json:"schedule_version"`
	ActiveDispatchCount int    `json:"active_dispatch_count"`
	BlockerSummary      string `json:"blocker_summary"`
	UpdatedAt           string `json:"updated_at"`
}

type sessionNotFoundError struct {
	Ref string
}

func (e sessionNotFoundError) Error() string {
	return "session not found: " + e.Ref
}

func (e sessionNotFoundError) Is(target error) bool {
	return target == ErrSessionNotFound
}
