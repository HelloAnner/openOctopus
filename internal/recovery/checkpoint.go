/*
Package recovery checkpoint 负责首版 checkpoint 追加写。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RecordCheckpoint 根据当前 session 文档生成一个新的 checkpoint。
func RecordCheckpoint(sessionDir string, input CheckpointInput) (CheckpointRecord, error) {
	service := NewService(sessionDir)
	if strings.TrimSpace(input.Kind) == "" {
		return CheckpointRecord{}, fmt.Errorf("checkpoint kind is required")
	}
	view, err := service.buildRecoveryView("")
	if err != nil {
		return CheckpointRecord{}, err
	}
	sequence, err := nextCheckpointSequence(service.paths.checkpointsDir)
	if err != nil {
		return CheckpointRecord{}, err
	}
	fileName := formatCheckpointFileName(sequence, input.Kind)
	path := filepath.Join(service.paths.checkpointsDir, fileName)
	if err := atomicWrite(path, []byte(renderCheckpoint(sequence, input, view))); err != nil {
		return CheckpointRecord{}, err
	}
	if err := service.updateSessionCheckpoint(sequence, view.LastEvent); err != nil {
		return CheckpointRecord{}, err
	}
	return CheckpointRecord{Sequence: sequence, Path: path, Ref: relativeSessionPath(sessionDir, path), Kind: input.Kind}, nil
}

func nextCheckpointSequence(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	sequences := make([]int, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 4 {
			continue
		}
		sequences = append(sequences, atoi(name[:4]))
	}
	if len(sequences) == 0 {
		return 1, nil
	}
	sort.Ints(sequences)
	return sequences[len(sequences)-1] + 1, nil
}

func renderCheckpoint(sequence int, input CheckpointInput, view recoveryView) string {
	lines := []string{
		fmt.Sprintf("# Recovery Checkpoint %04d", sequence),
		"",
		fmt.Sprintf("- checkpoint_seq: %d", sequence),
		fmt.Sprintf("- kind: %s", input.Kind),
		fmt.Sprintf("- session_id: %s", view.SessionID),
		fmt.Sprintf("- workflow_status: %s", view.WorkflowStatus),
		fmt.Sprintf("- current_stage_id: %s", view.CurrentStageID),
		fmt.Sprintf("- current_role_id: %s", view.CurrentRoleID),
		fmt.Sprintf("- schedule_version: %d", view.ScheduleVersion),
		fmt.Sprintf("- last_event: %s", view.LastEvent),
		fmt.Sprintf("- source: %s", defaultString(input.Source, "recovery")),
		fmt.Sprintf("- created_at: %s", utcNow()),
	}
	return strings.Join(lines, "\n") + "\n"
}

func (s *Service) updateSessionCheckpoint(sequence int, lastEvent string) error {
	state, _, err := readSessionState(s.paths.sessionState)
	if err != nil {
		return err
	}
	state.CheckpointSeq = sequence
	state.LastEvent = defaultString(lastEvent, state.LastEvent)
	state.UpdatedAt = utcNow()
	return atomicWrite(s.paths.sessionState, []byte(renderSessionState(state)))
}
