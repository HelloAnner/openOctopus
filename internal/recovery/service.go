/*
Package recovery service 提供 recovery 首版正式恢复入口。
Author: Anner
Created on 2026/3/8
*/
package recovery

// Recover 执行 recovery 001 的最小校验、修复与恢复准备。
func (s *Service) Recover(_ RecoverOptions) (RecoverResult, error) {
	validation, err := s.validateSessionLayout()
	if err != nil {
		return RecoverResult{}, err
	}
	view, err := s.buildRecoveryView(validation.SessionID)
	if err != nil {
		return RecoverResult{}, err
	}
	repairedFiles := make([]string, 0)
	if changed, err := s.repairSessionState(view); err != nil {
		return RecoverResult{}, err
	} else if changed {
		repairedFiles = appendUnique(repairedFiles, "session.state.md")
	}
	if changed, err := s.repairBlockers(view); err != nil {
		return RecoverResult{}, err
	} else if changed {
		repairedFiles = appendUnique(repairedFiles, "planner/blockers.md")
	}
	checkpoint, err := RecordCheckpoint(s.sessionDir, CheckpointInput{Kind: "recover-start", Source: "recovery"})
	if err != nil {
		return RecoverResult{}, err
	}
	result := RecoverResult{
		SessionID:       view.SessionID,
		SessionDir:      s.sessionDir,
		RecoveredStatus: view.WorkflowStatus,
		Continued:       canContinue(view.WorkflowStatus),
		RepairedFiles:   repairedFiles,
		CheckpointRef:   checkpoint.Ref,
		ReplayRef:       relativeSessionPath(s.sessionDir, s.paths.replay),
		Reason:          view.Reason,
		CanContinue:     canContinue(view.WorkflowStatus),
		CheckedFiles:    validation.CheckedFiles,
	}
	result.RepairedFiles = appendUnique(result.RepairedFiles, result.ReplayRef)
	if err := s.writeReplayReport(result); err != nil {
		return RecoverResult{}, err
	}
	return result, nil
}

func canContinue(status string) bool {
	switch status {
	case workflowStatusReady, workflowStatusRunning:
		return true
	default:
		return false
	}
}
