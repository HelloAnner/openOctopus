package orchestrator

import (
	"fmt"
	"strings"

	"github.com/anner/openoctopus/internal/config/model"
)

func initialSchedule(graph Graph) Schedule {
	now := utcNow()
	entrySet := make(map[string]struct{}, len(graph.EntryStageIDs))
	for _, item := range graph.EntryStageIDs {
		entrySet[item] = struct{}{}
	}
	stages := make([]StageSchedule, 0, len(graph.Order))
	for _, stageID := range graph.Order {
		node := graph.Stages[stageID]
		status := stageStatusPending
		if _, exists := entrySet[stageID]; exists {
			status = stageStatusReady
		}
		stages = append(stages, StageSchedule{
			StageID:     stageID,
			StageName:   node.Config.Name,
			RoleID:      node.Config.Role,
			Status:      status,
			Attempt:     0,
			NextStageID: node.NextStageID,
			UpdatedAt:   now,
		})
	}
	return Schedule{ScheduleVersion: 1, WorkflowStatus: workflowStatusReady, ActiveDispatchCount: 0, UpdatedAt: now, Stages: stages}
}

func readSchedule(path string) (Schedule, error) {
	content, err := readFile(path)
	if err != nil {
		return Schedule{}, err
	}
	if isPlaceholder(content) {
		return Schedule{}, ErrPlannerNotInitialized
	}
	head := leadingValues(content)
	schedule := Schedule{
		ScheduleVersion:     atoi(head["schedule_version"]),
		WorkflowStatus:      head["workflow_status"],
		ActiveDispatchCount: atoi(head["active_dispatch_count"]),
		UpdatedAt:           head["updated_at"],
		Stages:              make([]StageSchedule, 0),
	}
	for _, block := range splitBlocks(content, "## stage: ") {
		values := blockValues(block)
		schedule.Stages = append(schedule.Stages, StageSchedule{
			StageID:           values["stage_id"],
			StageName:         values["stage_name"],
			RoleID:            values["role_id"],
			Status:            values["status"],
			Attempt:           atoi(values["attempt"]),
			LastTaskID:        values["last_task_id"],
			LastConclusionRef: values["last_conclusion_ref"],
			NextStageID:       values["next_stage_id"],
			UpdatedAt:         values["updated_at"],
		})
	}
	return schedule, nil
}

func writeSchedule(path string, schedule Schedule) error {
	return atomicWrite(path, renderSchedule(schedule))
}

func renderSchedule(schedule Schedule) []byte {
	lines := []string{
		"# Master Schedule",
		"",
		fmt.Sprintf("- schedule_version: %d", schedule.ScheduleVersion),
		fmt.Sprintf("- workflow_status: %s", schedule.WorkflowStatus),
		fmt.Sprintf("- active_dispatch_count: %d", schedule.ActiveDispatchCount),
		fmt.Sprintf("- updated_at: %s", schedule.UpdatedAt),
	}
	for _, item := range schedule.Stages {
		lines = append(lines,
			"",
			fmt.Sprintf("## stage: %s", item.StageID),
			fmt.Sprintf("- stage_id: %s", item.StageID),
			fmt.Sprintf("- stage_name: %s", item.StageName),
			fmt.Sprintf("- role_id: %s", item.RoleID),
			fmt.Sprintf("- status: %s", item.Status),
			fmt.Sprintf("- attempt: %d", item.Attempt),
			fmt.Sprintf("- last_task_id: %s", item.LastTaskID),
			fmt.Sprintf("- last_conclusion_ref: %s", item.LastConclusionRef),
			fmt.Sprintf("- next_stage_id: %s", item.NextStageID),
			fmt.Sprintf("- updated_at: %s", item.UpdatedAt),
		)
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func findStage(schedule *Schedule, stageID string) *StageSchedule {
	for index := range schedule.Stages {
		if schedule.Stages[index].StageID == stageID {
			return &schedule.Stages[index]
		}
	}
	return nil
}

func updateWorkflowStatus(schedule *Schedule) {
	active := 0
	allCompleted := true
	for _, item := range schedule.Stages {
		if item.Status == stageStatusDispatched {
			active++
		}
		if item.Status != stageStatusCompleted {
			allCompleted = false
		}
		if item.Status == stageStatusFailed {
			schedule.WorkflowStatus = workflowStatusFailed
			schedule.ActiveDispatchCount = active
			schedule.UpdatedAt = utcNow()
			return
		}
		if item.Status == stageStatusBlocked {
			schedule.WorkflowStatus = workflowStatusWaitingHuman
			schedule.ActiveDispatchCount = active
			schedule.UpdatedAt = utcNow()
			return
		}
	}
	if allCompleted && len(schedule.Stages) != 0 {
		schedule.WorkflowStatus = workflowStatusCompleted
	} else if active > 0 {
		schedule.WorkflowStatus = workflowStatusRunning
	} else {
		schedule.WorkflowStatus = workflowStatusReady
	}
	schedule.ActiveDispatchCount = active
	schedule.UpdatedAt = utcNow()
}

func stageRoleMap(config model.RuntimeConfig) map[string]model.RoleConfig {
	values := make(map[string]model.RoleConfig, len(config.Roles))
	for _, item := range config.Roles {
		values[item.ID] = item
	}
	return values
}
