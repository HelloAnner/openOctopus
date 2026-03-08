/*
Package recovery repair 负责修复当前态文件与 replay 报告。
Author: Anner
Created on 2026/3/8
*/
package recovery

import "strings"

func (s *Service) repairSessionState(view recoveryView) (bool, error) {
	state, _, err := readSessionState(s.paths.sessionState)
	if err != nil {
		return false, err
	}
	state.SessionID = view.SessionID
	state.Status = view.WorkflowStatus
	state.CurrentStageID = view.CurrentStageID
	state.CurrentRoleID = view.CurrentRoleID
	state.LastEvent = view.LastEvent
	state.CreatedAt = defaultString(state.CreatedAt, view.CreatedAt)
	state.UpdatedAt = utcNow()
	state.HumanMessageCursor = view.HumanMessageCursor
	rendered := renderSessionState(state)
	existing, _ := readFile(s.paths.sessionState)
	if existing == rendered {
		return false, nil
	}
	return true, atomicWrite(s.paths.sessionState, []byte(rendered))
}

func (s *Service) repairBlockers(view recoveryView) (bool, error) {
	rendered := renderBlockers(view.BlockerSummary)
	existing, err := readFile(s.paths.blockers)
	if err == nil && existing == rendered {
		return false, nil
	}
	if err := atomicWrite(s.paths.blockers, []byte(rendered)); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) writeReplayReport(result RecoverResult) error {
	lines := []string{
		"# Recovery Replay",
		"",
		"- session_id: " + result.SessionID,
		"- recovered_status: " + result.RecoveredStatus,
		"- continued: " + boolString(result.Continued),
		"- reason: " + result.Reason,
		"- checkpoint_ref: " + result.CheckpointRef,
		"- replay_ref: " + result.ReplayRef,
		"- repaired_files: " + strings.Join(result.RepairedFiles, ", "),
		"- checked_files: " + strings.Join(result.CheckedFiles, ", "),
		"- generated_at: " + utcNow(),
	}
	return atomicWrite(s.paths.replay, []byte(strings.Join(lines, "\n")+"\n"))
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
