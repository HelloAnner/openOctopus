package roleruntime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/eventbus"
)

func (e *Engine) TickAll() (TickAllResult, error) {
	config, err := e.loadConfig()
	if err != nil {
		return TickAllResult{}, err
	}
	result := TickAllResult{}
	for _, role := range config.Roles {
		item, tickErr := e.TickRole(role.ID)
		if errors.Is(tickErr, ErrUnsupportedExecutor) {
			continue
		}
		if tickErr != nil {
			return TickAllResult{}, tickErr
		}
		if item.Progressed {
			result.Progressed = true
			result.ExecutedRoles++
			result.TurnCount += item.TurnSeq
		}
	}
	return result, nil
}

func (e *Engine) TickRole(roleID string) (RoleTickResult, error) {
	config, err := e.loadConfig()
	if err != nil {
		return RoleTickResult{}, err
	}
	role, profile, ok := findRole(config, roleID)
	if !ok {
		return RoleTickResult{}, fmt.Errorf("role not found: %s", roleID)
	}
	if err := ensureRoleFiles(e.paths.rolesDir, roleID); err != nil {
		return RoleTickResult{}, err
	}
	state, err := readRoleState(e.paths.rolesDir, roleID)
	if err != nil {
		return RoleTickResult{}, err
	}
	context, err := readRoleContext(e.paths.rolesDir, roleID)
	if err != nil {
		if os.IsNotExist(err) {
			return RoleTickResult{RoleID: roleID}, nil
		}
		return RoleTickResult{}, err
	}
	inbox, err := readRoleInbox(e.paths.rolesDir, roleID)
	if err != nil {
		if os.IsNotExist(err) {
			return RoleTickResult{RoleID: roleID}, nil
		}
		return RoleTickResult{}, err
	}
	reset, err := readRoleReset(e.paths.rolesDir, roleID)
	if err != nil {
		return RoleTickResult{}, err
	}
	if reset.Status == resetStatusRequested {
		return e.applyReset(roleID, state, reset)
	}
	if interrupted, interruptErr := e.handleInterrupt(roleID, state); interruptErr != nil || interrupted.Progressed {
		return interrupted, interruptErr
	}
	if !shouldExecute(state, context, inbox) {
		return RoleTickResult{RoleID: roleID}, nil
	}
	resolved, resolveErr := resolveExecutor(profile)
	if resolveErr != nil {
		return RoleTickResult{}, resolveErr
	}
	turnSeq := state.CurrentTurnSeq + 1
	request := buildExecuteRequest(e.sessionDir, config, role, profile, context, inbox, turnSeq)
	state = buildRunningState(state, profile.Provider, context, inbox, turnSeq)
	if err := writeRoleState(e.paths.rolesDir, state); err != nil {
		return RoleTickResult{}, err
	}
	if err := writeHeartbeat(e.paths.rolesDir, buildHeartbeat(roleID, state, heartbeatTimeout(config))); err != nil {
		return RoleTickResult{}, err
	}
	inputPath, err := writeTurnInput(e.paths.rolesDir, roleID, request, config)
	if err != nil {
		return RoleTickResult{}, err
	}
	_ = inputPath
	result, execErr := resolved.Execute(request)
	if execErr != nil {
		result = ExecuteResult{Provider: profile.Provider, Command: profile.CLIPath, ExitCode: 1, DurationMS: 0, Stderr: execErr.Error(), RawOutput: "## role_result\n- status: FAILED\n- summary: " + execErr.Error() + "\n- output_refs: "}
	}
	outputPath, err := writeTurnOutput(e.paths.rolesDir, roleID, turnSeq, result)
	if err != nil {
		return RoleTickResult{}, err
	}
	parsed := parseRoleResult(result.RawOutput)
	if err := writeConclusion(e.paths.rolesDir, roleID, inbox, parsed); err != nil {
		return RoleTickResult{}, err
	}
	if err := writeOutbox(e.paths.rolesDir, roleOutbox{
		OutboxVersion: turnSeq,
		RoleID:        roleID,
		StageID:       inbox.StageID,
		TaskID:        inbox.TaskID,
		TurnSeq:       turnSeq,
		Status:        parsed.Status,
		ConclusionRef: filepath.ToSlash(filepath.Join("roles", roleID, "conclusion.md")),
		TurnOutputRef: filepath.ToSlash(filepath.Join("roles", roleID, "turns", turnFileName(turnSeq, "output"))),
	}); err != nil {
		return RoleTickResult{}, err
	}
	state.Status = mapConclusionToRoleStatus(parsed.Status)
	state.CurrentTaskID = inbox.TaskID
	state.CurrentStageID = inbox.StageID
	state.CurrentTurnSeq = turnSeq
	state.ContextVersion = context.ContextVersion
	state.InboxVersion = inbox.InboxVersion
	state.LastConclusionStatus = parsed.Status
	if err := writeRoleState(e.paths.rolesDir, state); err != nil {
		return RoleTickResult{}, err
	}
	if err := writeHeartbeat(e.paths.rolesDir, buildHeartbeat(roleID, state, heartbeatTimeout(config))); err != nil {
		return RoleTickResult{}, err
	}
	if err := appendRoleEvent(e.paths.rolesDir, roleID, turnSeq, parsed.Status, outputPath); err != nil {
		return RoleTickResult{}, err
	}
	if err := e.commitRoleOffset(roleID); err != nil {
		return RoleTickResult{}, err
	}
	return RoleTickResult{RoleID: roleID, Progressed: true, TurnSeq: turnSeq, Status: parsed.Status}, nil
}

func buildExecuteRequest(sessionDir string, config model.RuntimeConfig, role model.RoleConfig, profile model.LLMProfile, context roleContext, inbox roleInbox, turnSeq int) ExecuteRequest {
	paths := []string{
		filepath.ToSlash(filepath.Join("planner", "requirement.snapshot.md")),
		filepath.ToSlash(filepath.Join("roles", role.ID, "context.md")),
		filepath.ToSlash(filepath.Join("roles", role.ID, "inbox.md")),
	}
	lines := []string{
		fmt.Sprintf("You are role %s.", role.ID),
		"Work only inside the current session directory.",
		"Do not browse the web, do not inspect unrelated files, and do not delete or rename any existing path.",
		"Read only these files before doing anything:",
	}
	for _, path := range paths {
		lines = append(lines, "- "+path)
	}
	for _, ref := range suggestedOutputRefs(config, inbox.StageID) {
		lines = append(lines, "- suggested artifact output: "+ref)
	}
	lines = append(lines,
		"If a suggested artifact output is listed, create or update only that path.",
		"After writing the artifact, return only one markdown block and no extra prose:",
		"## role_result",
		"- status: SUCCESS | NEEDS_RETRY | BLOCKED | FAILED",
		"- summary: short summary",
		"- output_refs: comma-separated refs or blank",
		fmt.Sprintf("Task: %s", inbox.TaskID),
		fmt.Sprintf("Stage: %s", inbox.StageID),
	)
	prompt := strings.Join(lines, "\n")
	return ExecuteRequest{SessionDir: sessionDir, Role: role, Profile: profile, Context: context, Inbox: inbox, TurnSeq: turnSeq, Prompt: prompt}
}


func suggestedOutputRefs(config model.RuntimeConfig, stageID string) []string {
	refs := make([]string, 0)
	for _, stage := range config.Stages {
		if stage.ID != stageID {
			continue
		}
		for _, output := range stage.Output {
			if output.Type != "artifact" || strings.TrimSpace(output.Name) == "" {
				continue
			}
			refs = append(refs, filepath.ToSlash(filepath.Join("artifacts", "_staging", stageID, output.Name+".md")))
		}
	}
	return refs
}

func buildRunningState(state roleState, provider string, context roleContext, inbox roleInbox, turnSeq int) roleState {
	state.Status = statusRunning
	state.CurrentStageID = inbox.StageID
	state.CurrentTaskID = inbox.TaskID
	state.CurrentTurnSeq = turnSeq
	state.ContextVersion = context.ContextVersion
	state.InboxVersion = inbox.InboxVersion
	state.ExecutorProvider = provider
	return state
}

func heartbeatTimeout(config model.RuntimeConfig) int {
	if config.Policies.Timeout.RoleHeartbeatTimeoutSeconds > 0 {
		return config.Policies.Timeout.RoleHeartbeatTimeoutSeconds
	}
	return 120
}

func shouldExecute(state roleState, context roleContext, inbox roleInbox) bool {
	if inbox.TaskID == "" || context.TaskID == "" {
		return false
	}
	if inbox.Status != "DISPATCHED" {
		return false
	}
	if state.CurrentTaskID != inbox.TaskID {
		return true
	}
	if state.ContextVersion != context.ContextVersion {
		return true
	}
	if state.InboxVersion != inbox.InboxVersion {
		return true
	}
	return state.LastConclusionStatus == ""
}

func appendRoleEvent(root string, roleID string, turnSeq int, status string, outputPath string) error {
	return appendBlock(eventsPath(root, roleID), "# Role Events", []string{
		fmt.Sprintf("## event: turn-%04d", turnSeq),
		fmt.Sprintf("- event_type: ROLE_TURN_COMPLETED"),
		fmt.Sprintf("- turn_seq: %d", turnSeq),
		fmt.Sprintf("- status: %s", status),
		fmt.Sprintf("- output_ref: %s", filepath.ToSlash(filepath.Join("roles", roleID, "turns", filepath.Base(outputPath)))),
		fmt.Sprintf("- updated_at: %s", utcNow()),
	})
}

func (e *Engine) commitRoleOffset(roleID string) error {
	tail, err := e.bus.Tail()
	if err != nil {
		return nil
	}
	lease, err := e.bus.AcquireLock("role-runtime/"+roleID, 30*time.Second)
	if err != nil {
		return nil
	}
	defer func() {
		_ = e.bus.ReleaseLock(lease)
	}()
	_, _ = e.bus.Append(lease, eventbus.AppendEvent{EventType: "ROLE_TURN_COMPLETED", Producer: "role-runtime", SessionID: tail.SessionID, RoleID: roleID, PayloadRef: filepath.ToSlash(filepath.Join("roles", roleID, "conclusion.md")), Summary: "role turn completed"})
	updatedTail, updatedErr := e.bus.Tail()
	if updatedErr != nil {
		return nil
	}
	return e.bus.CommitOffset(lease, eventbus.OffsetCommit{ConsumerID: "role-runtime/" + roleID, LastEventID: updatedTail.EventID, LastSequence: updatedTail.Sequence, Note: "role runtime tick applied"})
}

func (e *Engine) handleInterrupt(roleID string, state roleState) (RoleTickResult, error) {
	records, err := e.bus.ReadInterrupts()
	if err != nil {
		return RoleTickResult{}, nil
	}
	for _, record := range records {
		if record.TargetRoleID != roleID || record.Status != eventbus.InterruptStatusRequested {
			continue
		}
		state.Status = statusInterrupted
		if err := writeRoleState(e.paths.rolesDir, state); err != nil {
			return RoleTickResult{}, err
		}
		if err := writeHeartbeat(e.paths.rolesDir, buildHeartbeat(roleID, state, 120)); err != nil {
			return RoleTickResult{}, err
		}
		lease, acquireErr := e.bus.AcquireLock("role-runtime/"+roleID, 30*time.Second)
		if acquireErr == nil {
			_, _ = e.bus.AcknowledgeInterrupt(lease, record.InterruptID)
			_ = e.bus.ReleaseLock(lease)
		}
		return RoleTickResult{RoleID: roleID, Progressed: true, Status: statusInterrupted}, nil
	}
	return RoleTickResult{RoleID: roleID}, nil
}

func (e *Engine) applyReset(roleID string, state roleState, reset roleReset) (RoleTickResult, error) {
	state.SessionGeneration = state.SessionGeneration + 1
	state.Status = statusIdle
	state.CurrentTaskID = ""
	state.CurrentStageID = ""
	state.CurrentTurnSeq = 0
	state.ContextVersion = 0
	state.InboxVersion = 0
	state.LastConclusionStatus = ""
	if err := writeRoleState(e.paths.rolesDir, state); err != nil {
		return RoleTickResult{}, err
	}
	reset.SessionGeneration = state.SessionGeneration
	reset.Status = resetStatusApplied
	reset.AppliedAt = utcNow()
	reset.LastClearedTaskID = state.CurrentTaskID
	if err := writeRoleReset(e.paths.rolesDir, roleID, reset); err != nil {
		return RoleTickResult{}, err
	}
	if err := writeHeartbeat(e.paths.rolesDir, buildHeartbeat(roleID, state, 120)); err != nil {
		return RoleTickResult{}, err
	}
	return RoleTickResult{RoleID: roleID, Progressed: true, Status: resetStatusApplied}, nil
}
