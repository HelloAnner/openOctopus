package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/eventbus"
)

func (e *Engine) Tick() (TickResult, error) {
	if err := e.Bootstrap(); err != nil {
		return TickResult{}, err
	}
	lease, err := e.bus.AcquireLock(orchestratorHolder, 30*time.Second)
	if err != nil {
		return TickResult{}, err
	}
	defer func() {
		_ = e.bus.ReleaseLock(lease)
	}()

	config, err := e.loadConfig()
	if err != nil {
		return TickResult{}, err
	}
	graph, err := buildGraph(config)
	if err != nil {
		return TickResult{}, err
	}
	schedule, err := readSchedule(e.paths.schedule)
	if err != nil {
		return TickResult{}, err
	}
	state, err := readSessionState(e.paths.sessionState)
	if err != nil {
		return TickResult{}, err
	}
	snapshot, err := readRequirementSnapshot(e.paths.requirement)
	if err != nil {
		return TickResult{}, err
	}
	result := TickResult{WorkflowStatus: schedule.WorkflowStatus}
	latestSummary := "none"

	updatedSnapshot, snapshotChanged, err := e.refreshSnapshot(config, snapshot, lease)
	if err != nil {
		return TickResult{}, err
	}
	if snapshotChanged {
		snapshot = updatedSnapshot
		result.SnapshotUpdated = true
		latestSummary = snapshot.LatestMessages
	}
	decisionLines := make([][]string, 0)
	blockerSummary := "clear"
	if err := e.applyConclusions(config, &schedule, &state, lease, &decisionLines, &blockerSummary); err != nil {
		return TickResult{}, err
	}
	if schedule.WorkflowStatus != workflowStatusWaitingHuman && schedule.WorkflowStatus != workflowStatusFailed && schedule.WorkflowStatus != workflowStatusCompleted {
		selected := selectReadyStages(schedule, effectiveMaxParallelRoles(config))
		for _, ready := range selected {
			stage := findStage(&schedule, ready.StageID)
			if stage == nil {
				continue
			}
			if err := e.dispatchStage(config, &schedule, stage, lease); err != nil {
				return TickResult{}, err
			}
			state.CurrentStageID = stage.StageID
			state.CurrentRoleID = stage.RoleID
			result.DispatchedCount++
			decisionLines = append(decisionLines, []string{
				fmt.Sprintf("## decision: dispatch-%s", stage.LastTaskID),
				fmt.Sprintf("- type: dispatch"),
				fmt.Sprintf("- stage_id: %s", stage.StageID),
				fmt.Sprintf("- role_id: %s", stage.RoleID),
				fmt.Sprintf("- task_id: %s", stage.LastTaskID),
				fmt.Sprintf("- at: %s", utcNow()),
			})
		}
	}
	updateWorkflowStatus(&schedule)
	if schedule.WorkflowStatus == workflowStatusCompleted {
		state.Status = workflowStatusCompleted
	} else if schedule.WorkflowStatus == workflowStatusFailed {
		state.Status = workflowStatusFailed
	} else if schedule.WorkflowStatus == workflowStatusWaitingHuman {
		state.Status = workflowStatusWaitingHuman
	} else if schedule.ActiveDispatchCount > 0 {
		state.Status = workflowStatusRunning
	} else {
		state.Status = workflowStatusReady
	}
	state.HumanMessageCursor = snapshot.HumanMessageCursor
	state.UpdatedAt = utcNow()
	if state.LastEvent == "" {
		state.LastEvent = "SESSION_CREATED"
	}

	if err := writeSchedule(e.paths.schedule, schedule); err != nil {
		return TickResult{}, err
	}
	if err := atomicWrite(e.paths.taskBoard, renderTaskBoard(schedule)); err != nil {
		return TickResult{}, err
	}
	if err := atomicWrite(e.paths.taskGraph, renderTaskGraph(graph)); err != nil {
		return TickResult{}, err
	}
	if err := atomicWrite(e.paths.globalProgress, renderGlobalProgress(schedule)); err != nil {
		return TickResult{}, err
	}
	if err := atomicWrite(e.paths.blockers, renderBlockers(blockerSummary)); err != nil {
		return TickResult{}, err
	}
	if err := writeSessionState(e.paths.sessionState, state); err != nil {
		return TickResult{}, err
	}
	for _, lines := range decisionLines {
		if err := appendMarkdownBlock(e.paths.decisionLog, "# Decision Log", lines); err != nil {
			return TickResult{}, err
		}
	}
	if result.DispatchedCount > 0 || snapshotChanged || len(decisionLines) > 0 {
		tail, err := e.bus.Tail()
		if err == nil {
			state.LastEvent = tail.EventType
			if commitErr := e.bus.CommitOffset(lease, eventbus.OffsetCommit{ConsumerID: orchestratorConsumerID, LastEventID: tail.EventID, LastSequence: tail.Sequence, Note: "orchestrator tick applied"}); commitErr != nil {
				return TickResult{}, commitErr
			}
		}
	}
	result.WorkflowStatus = schedule.WorkflowStatus
	_ = latestSummary
	return result, nil
}

func (e *Engine) refreshSnapshot(config model.RuntimeConfig, snapshot RequirementSnapshot, lease eventbus.Lease) (RequirementSnapshot, bool, error) {
	content, err := readFile(e.paths.humanMessages)
	if err != nil {
		return RequirementSnapshot{}, false, err
	}
	messages, err := parseHumanMessages(content)
	if err != nil {
		return RequirementSnapshot{}, false, err
	}
	newMessages := unseenMessages(messages, snapshot.HumanMessageCursor)
	if len(newMessages) == 0 {
		return snapshot, false, nil
	}
	cursor := newMessages[len(newMessages)-1].MessageID
	merged := make([]string, 0, len(newMessages))
	for _, item := range newMessages {
		merged = append(merged, item.Content)
	}
	snapshot.SnapshotVersion++
	snapshot.HumanMessageCursor = cursor
	snapshot.SourceMessageCount = len(messages)
	snapshot.PlannerStatus = workflowStatusRunning
	snapshot.UpdatedAt = utcNow()
	snapshot.WorkflowSummary = buildWorkflowSummary(config)
	snapshot.LatestMessages = strings.Join(merged, "\n")
	snapshot.DispatchBrief = "messages absorbed"
	if err := writeRequirementSnapshot(e.paths.requirement, snapshot); err != nil {
		return RequirementSnapshot{}, false, err
	}
	_, err = e.bus.Append(lease, eventbus.AppendEvent{
		EventType:  "REQUIREMENT_SNAPSHOT_UPDATED",
		Producer:   "orchestrator",
		SessionID:  readStateSessionIDOrEmpty(e.paths.sessionState),
		PayloadRef: filepath.ToSlash(filepath.Join("planner", "requirement.snapshot.md")),
		Summary:    "requirement snapshot updated",
	})
	if err != nil {
		return RequirementSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func unseenMessages(messages []HumanMessage, cursor string) []HumanMessage {
	if cursor == "" {
		return messages
	}
	found := false
	filtered := make([]HumanMessage, 0)
	for _, item := range messages {
		if found {
			filtered = append(filtered, item)
		}
		if item.MessageID == cursor {
			found = true
		}
	}
	if !found {
		return messages
	}
	return filtered
}

func (e *Engine) applyConclusions(config model.RuntimeConfig, schedule *Schedule, state *sessionState, lease eventbus.Lease, decisionLines *[][]string, blockerSummary *string) error {
	for index := range schedule.Stages {
		stage := &schedule.Stages[index]
		if stage.Status != stageStatusDispatched {
			continue
		}
		path := filepath.Join(e.paths.rolesDir, stage.RoleID, "conclusion.md")
		if _, err := os.Stat(path); err != nil {
			continue
		}
		conclusion, err := readConclusion(path)
		if err != nil {
			return err
		}
		if conclusion.TaskID != stage.LastTaskID {
			continue
		}
		stage.LastConclusionRef = conclusionPath(stage.RoleID)
		stage.UpdatedAt = utcNow()
		switch conclusion.Status {
		case conclusionSuccess:
			stageConfig, found := findStageConfig(config, stage.StageID)
			if !found {
				return ErrDispatchConflict
			}
			if err := e.publishArtifacts(stageConfig, *stage, conclusion, lease); err != nil {
				return err
			}
			stage.Status = stageStatusCompleted
			_, err = e.bus.Append(lease, eventbus.AppendEvent{EventType: "STAGE_COMPLETED", Producer: "orchestrator", SessionID: state.SessionID, RoleID: stage.RoleID, PayloadRef: stage.LastConclusionRef, Summary: conclusion.Summary})
			if err != nil {
				return err
			}
			if stage.NextStageID == model.EndStage {
				_, err = e.bus.Append(lease, eventbus.AppendEvent{EventType: "WORKFLOW_COMPLETED", Producer: "orchestrator", SessionID: state.SessionID, PayloadRef: filepath.ToSlash(filepath.Join("planner", "master_schedule.md")), Summary: "workflow completed"})
				if err != nil {
					return err
				}
			} else if nextStage := findStage(schedule, stage.NextStageID); nextStage != nil && nextStage.Status == stageStatusPending {
				nextStage.Status = stageStatusReady
				nextStage.UpdatedAt = utcNow()
				_, err = e.bus.Append(lease, eventbus.AppendEvent{EventType: "STAGE_READY", Producer: "orchestrator", SessionID: state.SessionID, RoleID: nextStage.RoleID, PayloadRef: filepath.ToSlash(filepath.Join("planner", "master_schedule.md")), Summary: fmt.Sprintf("%s ready", nextStage.StageID)})
				if err != nil {
					return err
				}
			}
		case conclusionNeedsRetry:
			stage.Status = stageStatusRetryPending
			stage.Attempt++
			_, err = e.bus.Append(lease, eventbus.AppendEvent{EventType: "STAGE_RETRY_SCHEDULED", Producer: "orchestrator", SessionID: state.SessionID, RoleID: stage.RoleID, PayloadRef: stage.LastConclusionRef, Summary: conclusion.Summary})
			if err != nil {
				return err
			}
		case conclusionBlocked:
			stage.Status = stageStatusBlocked
			*blockerSummary = conclusion.Summary
			_, err = e.bus.Append(lease, eventbus.AppendEvent{EventType: "STAGE_BLOCKED", Producer: "orchestrator", SessionID: state.SessionID, RoleID: stage.RoleID, PayloadRef: stage.LastConclusionRef, Summary: conclusion.Summary})
			if err != nil {
				return err
			}
		case conclusionFailed:
			stage.Status = stageStatusFailed
			_, err = e.bus.Append(lease, eventbus.AppendEvent{EventType: "WORKFLOW_FAILED", Producer: "orchestrator", SessionID: state.SessionID, RoleID: stage.RoleID, PayloadRef: stage.LastConclusionRef, Summary: conclusion.Summary})
			if err != nil {
				return err
			}
		default:
			return ErrInvalidConclusion
		}
		*decisionLines = append(*decisionLines, []string{
			fmt.Sprintf("## decision: %s-%s", stage.StageID, strings.ToLower(conclusion.Status)),
			fmt.Sprintf("- type: conclusion"),
			fmt.Sprintf("- stage_id: %s", stage.StageID),
			fmt.Sprintf("- status: %s", conclusion.Status),
			fmt.Sprintf("- summary: %s", conclusion.Summary),
			fmt.Sprintf("- at: %s", utcNow()),
		})
	}
	return nil
}
