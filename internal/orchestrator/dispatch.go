package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/eventbus"
)

func selectReadyStages(schedule Schedule, maxParallel int) []StageSchedule {
	selected := make([]StageSchedule, 0)
	activeRoles := make(map[string]struct{})
	for _, item := range schedule.Stages {
		if item.Status == stageStatusDispatched {
			activeRoles[item.RoleID] = struct{}{}
		}
	}
	for _, item := range schedule.Stages {
		if item.Status != stageStatusReady && item.Status != stageStatusRetryPending {
			continue
		}
		if _, exists := activeRoles[item.RoleID]; exists {
			continue
		}
		selected = append(selected, item)
		activeRoles[item.RoleID] = struct{}{}
		if len(selected) >= maxParallel {
			break
		}
	}
	return selected
}

func (e *Engine) dispatchStage(config model.RuntimeConfig, schedule *Schedule, stage *StageSchedule, lease eventbus.Lease) error {
	roleConfigs := stageRoleMap(config)
	roleConfig := roleConfigs[stage.RoleID]
	if roleConfig.ID == "" {
		return ErrDispatchConflict
	}
	attempt := stage.Attempt
	if attempt == 0 {
		attempt = 1
	}
	taskID := fmt.Sprintf("task-%s-%02d", stage.StageID, attempt)
	roleDir := filepath.Join(e.paths.rolesDir, stage.RoleID)
	if err := os.MkdirAll(roleDir, 0o755); err != nil {
		return err
	}
	contextPath := filepath.Join(roleDir, "context.md")
	inboxPath := filepath.Join(roleDir, "inbox.md")
	contextVersion := stage.Attempt + 1
	inboxVersion := stage.Attempt + 1
	context := renderContext(*stage, taskID, contextVersion, roleConfig)
	inboxEvent, err := e.bus.Append(lease, eventbus.AppendEvent{
		EventType:  "TASK_DISPATCHED",
		Producer:   "orchestrator",
		SessionID:  readStateSessionIDOrEmpty(e.paths.sessionState),
		RoleID:     stage.RoleID,
		PayloadRef: filepath.ToSlash(filepath.Join("roles", stage.RoleID, "inbox.md")),
		Summary:    fmt.Sprintf("dispatched %s to %s", stage.StageID, stage.RoleID),
	})
	if err != nil {
		return err
	}
	if err := atomicWrite(contextPath, []byte(context)); err != nil {
		return err
	}
	if err := atomicWrite(inboxPath, []byte(renderInbox(*stage, taskID, inboxVersion, contextVersion, inboxEvent.EventID))); err != nil {
		return err
	}
	stage.Status = stageStatusDispatched
	stage.Attempt = attempt
	stage.LastTaskID = taskID
	stage.UpdatedAt = utcNow()
	if err := appendDispatchLog(e.paths.dispatchLog, *stage, taskID, inboxEvent.EventID); err != nil {
		return err
	}
	return nil
}

func renderContext(stage StageSchedule, taskID string, contextVersion int, roleConfig model.RoleConfig) string {
	lines := []string{
		"# Role Context",
		"",
		fmt.Sprintf("- context_version: %d", contextVersion),
		fmt.Sprintf("- task_id: %s", taskID),
		fmt.Sprintf("- stage_id: %s", stage.StageID),
		fmt.Sprintf("- role_id: %s", stage.RoleID),
		fmt.Sprintf("- requirement_snapshot_ref: %s", filepath.ToSlash(filepath.Join("planner", "requirement.snapshot.md"))),
		fmt.Sprintf("- master_schedule_ref: %s", filepath.ToSlash(filepath.Join("planner", "master_schedule.md"))),
		fmt.Sprintf("- attempt: %d", stage.Attempt+1),
		fmt.Sprintf("- updated_at: %s", utcNow()),
		"",
		"## stage_goal",
		stage.StageName,
		"",
		"## system_prompt",
		roleConfig.SystemPrompt,
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderInbox(stage StageSchedule, taskID string, inboxVersion int, contextVersion int, dispatchEventID string) string {
	lines := []string{
		"# Role Inbox",
		"",
		fmt.Sprintf("- inbox_version: %d", inboxVersion),
		fmt.Sprintf("- task_id: %s", taskID),
		fmt.Sprintf("- stage_id: %s", stage.StageID),
		fmt.Sprintf("- role_id: %s", stage.RoleID),
		"- status: DISPATCHED",
		fmt.Sprintf("- dispatch_event_id: %s", dispatchEventID),
		fmt.Sprintf("- context_version: %d", contextVersion),
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}
	return strings.Join(lines, "\n") + "\n"
}

func appendDispatchLog(path string, stage StageSchedule, taskID string, dispatchEventID string) error {
	lines := []string{
		fmt.Sprintf("## dispatch: %s", taskID),
		fmt.Sprintf("- task_id: %s", taskID),
		fmt.Sprintf("- stage_id: %s", stage.StageID),
		fmt.Sprintf("- role_id: %s", stage.RoleID),
		fmt.Sprintf("- dispatch_event_id: %s", dispatchEventID),
		fmt.Sprintf("- at: %s", utcNow()),
	}
	return appendMarkdownBlock(path, "# Dispatch Log", lines)
}

func readStateSessionIDOrEmpty(path string) string {
	state, err := readSessionState(path)
	if err != nil {
		return ""
	}
	return state.SessionID
}
