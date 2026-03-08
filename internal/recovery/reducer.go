/*
Package recovery reducer 负责从当前文档归约恢复视图。
Author: Anner
Created on 2026/3/8
*/
package recovery

func (s *Service) buildRecoveryView(sessionID string) (recoveryView, error) {
	metadata, err := readMetadata(s.paths.metadata)
	if err != nil {
		return recoveryView{}, err
	}
	state, _, err := readSessionState(s.paths.sessionState)
	if err != nil {
		return recoveryView{}, err
	}
	schedule, err := readSchedule(s.paths.schedule)
	if err != nil {
		return recoveryView{}, err
	}
	blockerSummary, err := readBlockerSummary(s.paths.blockers)
	if err != nil {
		return recoveryView{}, err
	}
	status := reduceWorkflowStatus(schedule)
	currentStage, currentRole := reduceCurrentPointers(schedule, state)
	if status == workflowStatusWaitingHuman && blockerSummary == "clear" {
		blockerSummary = blockedStageSummary(schedule, s.paths.rolesDir)
	}
	if status != workflowStatusWaitingHuman {
		blockerSummary = "clear"
	}
	tail, err := s.bus.Tail()
	lastEvent := state.LastEvent
	if err == nil {
		lastEvent = tail.EventType
	}
	return recoveryView{
		SessionID:          defaultString(sessionID, metadata.SessionID),
		WorkflowStatus:     status,
		CurrentStageID:     currentStage,
		CurrentRoleID:      currentRole,
		BlockerSummary:     blockerSummary,
		LastEvent:          defaultString(lastEvent, "SESSION_CREATED"),
		CreatedAt:          defaultString(state.CreatedAt, metadata.CreatedAt),
		HumanMessageCursor: state.HumanMessageCursor,
		ScheduleVersion:    schedule.ScheduleVersion,
		Reason:             reasonForStatus(status),
	}, nil
}

func reduceWorkflowStatus(schedule scheduleDoc) string {
	hasDispatched := false
	allCompleted := len(schedule.Stages) != 0
	for _, stage := range schedule.Stages {
		switch stage.Status {
		case "FAILED":
			return workflowStatusFailed
		case "BLOCKED":
			return workflowStatusWaitingHuman
		case "DISPATCHED":
			hasDispatched = true
		}
		if stage.Status != "COMPLETED" {
			allCompleted = false
		}
	}
	if allCompleted {
		return workflowStatusCompleted
	}
	if hasDispatched {
		return workflowStatusRunning
	}
	return workflowStatusReady
}

func reduceCurrentPointers(schedule scheduleDoc, state sessionStateDoc) (string, string) {
	for _, status := range []string{"DISPATCHED", "BLOCKED", "FAILED", "RETRY_PENDING", "READY"} {
		for _, stage := range schedule.Stages {
			if stage.Status == status {
				return stage.StageID, stage.RoleID
			}
		}
	}
	return state.CurrentStageID, state.CurrentRoleID
}

func blockedStageSummary(schedule scheduleDoc, rolesDir string) string {
	for _, stage := range schedule.Stages {
		if stage.Status != "BLOCKED" {
			continue
		}
		summary := readConclusionSummary(rolesDir, stage.LastConclusionRef)
		if summary != "" {
			return summary
		}
		return "waiting human"
	}
	return "waiting human"
}

func reasonForStatus(status string) string {
	switch status {
	case workflowStatusWaitingHuman:
		return "needs_human_resume"
	case workflowStatusCompleted:
		return "already_completed"
	case workflowStatusFailed:
		return "already_failed"
	default:
		return ""
	}
}
