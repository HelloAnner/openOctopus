/*
Package cli service 负责 session 解析与 status 聚合读取。
Author: Anner
Created on 2026/3/8
*/
package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// NewService 创建一个 CLI 只读服务。
func NewService(workingDir string) *Service {
	return &Service{workingDir: workingDir}
}

// ResolveSessionDir 将 session id 或路径解析为真实目录。
func (s *Service) ResolveSessionDir(sessionRef string) (string, error) {
	trimmed := strings.TrimSpace(sessionRef)
	if trimmed == "" {
		return "", sessionNotFoundError{Ref: sessionRef}
	}
	for _, candidate := range resolveCandidates(trimmed, s.workingDir) {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return filepath.Clean(candidate), nil
		}
	}
	return "", sessionNotFoundError{Ref: sessionRef}
}

// ReadStatus 聚合 session 当前展示态。
func (s *Service) ReadStatus(sessionDir string) (StatusSummary, error) {
	metadataValues, err := s.readMetadata(filepath.Join(sessionDir, "metadata.md"))
	if err != nil {
		return StatusSummary{}, err
	}
	stateValues, err := s.readState(filepath.Join(sessionDir, "session.state.md"))
	if err != nil {
		return StatusSummary{}, err
	}
	scheduleVersion, activeDispatchCount, err := s.readSchedule(filepath.Join(sessionDir, "planner", "master_schedule.md"))
	if err != nil {
		return StatusSummary{}, err
	}
	blockerSummary, err := s.readBlockers(filepath.Join(sessionDir, "planner", "blockers.md"))
	if err != nil {
		return StatusSummary{}, err
	}
	return StatusSummary{
		SessionID:           metadataValues["session_id"],
		SessionDir:          filepath.Clean(sessionDir),
		WorkflowID:          metadataValues["workflow_id"],
		WorkflowName:        metadataValues["workflow_name"],
		WorkflowStatus:      firstNonEmpty(stateValues["status"], stateValues["workflow_status"]),
		CurrentStageID:      stateValues["current_stage_id"],
		CurrentRoleID:       stateValues["current_role_id"],
		ScheduleVersion:     scheduleVersion,
		ActiveDispatchCount: activeDispatchCount,
		BlockerSummary:      blockerSummary,
		UpdatedAt:           stateValues["updated_at"],
	}, nil
}

func (s *Service) readMetadata(path string) (map[string]string, error) {
	content, err := readTextFile(path)
	if err != nil {
		return nil, err
	}
	return leadingValues(content), nil
}

func (s *Service) readState(path string) (map[string]string, error) {
	content, err := readTextFile(path)
	if err != nil {
		return nil, err
	}
	return leadingValues(content), nil
}

func (s *Service) readSchedule(path string) (int, int, error) {
	content, err := readTextFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	if placeholderDocument(content) {
		return 0, 0, nil
	}
	values := leadingValues(content)
	return atoi(values["schedule_version"]), atoi(values["active_dispatch_count"]), nil
}

func (s *Service) readBlockers(path string) (string, error) {
	content, err := readTextFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "clear", nil
		}
		return "", err
	}
	return firstNonEmpty(defaultBlockerSummary(content), "clear"), nil
}

func firstNonEmpty(values ...string) string {
	for _, item := range values {
		if strings.TrimSpace(item) != "" {
			return item
		}
	}
	return ""
}
