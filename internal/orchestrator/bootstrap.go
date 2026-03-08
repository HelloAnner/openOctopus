package orchestrator

import (
	"fmt"
	"os"

	"github.com/anner/openoctopus/internal/config/model"
)

func (e *Engine) Bootstrap() error {
	config, err := e.loadConfig()
	if err != nil {
		return err
	}
	graph, err := buildGraph(config)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(e.paths.rolesDir, 0o755); err != nil {
		return err
	}
	if err := e.ensureSnapshot(config); err != nil {
		return err
	}
	schedule := initialSchedule(graph)
	if err := e.ensureFile(e.paths.schedule, renderSchedule(schedule)); err != nil {
		return err
	}
	if err := e.ensureFile(e.paths.taskBoard, renderTaskBoard(schedule)); err != nil {
		return err
	}
	if err := e.ensureFile(e.paths.taskGraph, renderTaskGraph(graph)); err != nil {
		return err
	}
	if err := e.ensureFile(e.paths.globalProgress, renderGlobalProgress(schedule)); err != nil {
		return err
	}
	if err := e.ensureFile(e.paths.blockers, renderBlockers("clear")); err != nil {
		return err
	}
	if err := e.ensureFile(e.paths.dispatchLog, []byte("# Dispatch Log\n")); err != nil {
		return err
	}
	return e.ensureFile(e.paths.decisionLog, []byte("# Decision Log\n"))
}

func (e *Engine) ensureFile(path string, content []byte) error {
	existing, err := readFile(path)
	if err == nil && !isPlaceholder(existing) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return atomicWrite(path, content)
}

func (e *Engine) ensureSnapshot(config model.RuntimeConfig) error {
	existing, err := readFile(e.paths.requirement)
	if err == nil && !isPlaceholder(existing) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	snapshot := RequirementSnapshot{
		SnapshotVersion:    1,
		WorkflowID:         config.Meta.WorkflowID,
		WorkflowName:       config.Meta.Name,
		HumanMessageCursor: "",
		SourceMessageCount: 0,
		PlannerStatus:      workflowStatusReady,
		UpdatedAt:          utcNow(),
		WorkflowSummary:    buildWorkflowSummary(config),
		LatestMessages:     "none",
		DispatchBrief:      "not dispatched yet",
	}
	return writeRequirementSnapshot(e.paths.requirement, snapshot)
}

func buildWorkflowSummary(config model.RuntimeConfig) string {
	return fmt.Sprintf("workflow_id=%s, roles=%d, stages=%d", config.Meta.WorkflowID, len(config.Roles), len(config.Stages))
}
