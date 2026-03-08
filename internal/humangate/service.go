/*
Package humangate service 负责组合 event-bus、planner 与 session 状态更新。
Author: Anner
Created on 2026/3/8
*/
package humangate

import (
	"fmt"
	"strings"
	"time"

	"github.com/anner/openoctopus/internal/eventbus"
	"github.com/anner/openoctopus/internal/recovery"
)

// Interrupt 对单个角色发起中断请求。
func (s *Service) Interrupt(options InterruptOptions) (eventbus.InterruptRecord, error) {
	if strings.TrimSpace(options.RoleID) == "" {
		return eventbus.InterruptRecord{}, fmt.Errorf("role is required")
	}
	if strings.TrimSpace(options.Reason) == "" {
		return eventbus.InterruptRecord{}, fmt.Errorf("reason is required")
	}
	lease, err := s.bus.AcquireLock(humanGateHolder, 30*time.Second)
	if err != nil {
		return eventbus.InterruptRecord{}, err
	}
	defer func() {
		_ = s.bus.ReleaseLock(lease)
	}()
	return s.bus.RequestInterrupt(lease, eventbus.InterruptRequest{
		Scope:        eventbus.InterruptScopeRole,
		TargetRoleID: options.RoleID,
		Source:       humanGateSource,
		Reason:       options.Reason,
	})
}

// InterruptAll 将当前未完成角色统一拉入等待人工态。
func (s *Service) InterruptAll(options InterruptAllOptions) (InterruptAllResult, error) {
	if strings.TrimSpace(options.Reason) == "" {
		return InterruptAllResult{}, fmt.Errorf("reason is required")
	}
	schedule, err := readSchedule(s.paths.schedule)
	if err != nil {
		return InterruptAllResult{}, err
	}
	state, err := readSessionState(s.paths.sessionState)
	if err != nil {
		return InterruptAllResult{}, err
	}
	roles := interruptTargets(schedule)
	lease, err := s.bus.AcquireLock(humanGateHolder, 30*time.Second)
	if err != nil {
		return InterruptAllResult{}, err
	}
	defer func() {
		_ = s.bus.ReleaseLock(lease)
	}()
	requested, err := s.requestInterrupts(lease, roles, options.Reason)
	if err != nil {
		return InterruptAllResult{}, err
	}
	state.Status = "WAITING_HUMAN"
	state.UpdatedAt = utcNow()
	if err := writeSessionState(s.paths.sessionState, state); err != nil {
		return InterruptAllResult{}, err
	}
	if err := writeBlockers(s.paths.blockers, options.Reason); err != nil {
		return InterruptAllResult{}, err
	}
	if err := appendDecision(s.paths.decisionLog, "interrupt-all", "human_gate", fmt.Sprintf("- requested_roles: %d", requested), fmt.Sprintf("- reason: %s", options.Reason)); err != nil {
		return InterruptAllResult{}, err
	}
	if _, err := recovery.RecordCheckpoint(s.sessionDir, recovery.CheckpointInput{Kind: "session-waiting-human", Source: "human-gate"}); err != nil {
		return InterruptAllResult{}, err
	}
	return InterruptAllResult{RequestedCount: requested}, nil
}

// Resume 清理人工等待态对应的 interrupt，并把阻塞阶段重新入队。
func (s *Service) Resume(options ResumeOptions) (ResumeResult, error) {
	result := ResumeResult{}
	schedule, err := readSchedule(s.paths.schedule)
	if err != nil {
		return result, err
	}
	state, err := readSessionState(s.paths.sessionState)
	if err != nil {
		return result, err
	}
	lease, err := s.bus.AcquireLock(humanGateHolder, 30*time.Second)
	if err != nil {
		return result, err
	}
	defer func() {
		_ = s.bus.ReleaseLock(lease)
	}()
	cleared, err := s.clearInterrupts(lease, options.RoleID)
	if err != nil {
		return result, err
	}
	result.ClearedInterrupts = cleared
	requeued := requeueBlockedStages(&schedule, options.RoleID)
	result.RequeuedStages = requeued
	if requeued > 0 {
		if err := writeSchedule(s.paths.schedule, schedule); err != nil {
			return result, err
		}
	}
	if result.ClearedInterrupts == 0 && result.RequeuedStages == 0 {
		return result, nil
	}
	state.Status = "READY"
	state.UpdatedAt = utcNow()
	if err := writeSessionState(s.paths.sessionState, state); err != nil {
		return result, err
	}
	if err := writeBlockers(s.paths.blockers, "clear"); err != nil {
		return result, err
	}
	if err := appendDecision(s.paths.decisionLog, "resume", "human_gate", fmt.Sprintf("- cleared_interrupts: %d", result.ClearedInterrupts), fmt.Sprintf("- requeued_stages: %d", result.RequeuedStages)); err != nil {
		return result, err
	}
	if _, err := recovery.RecordCheckpoint(s.sessionDir, recovery.CheckpointInput{Kind: "session-resumed", Source: "human-gate"}); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) requestInterrupts(lease eventbus.Lease, roles []string, reason string) (int, error) {
	requested := 0
	for _, roleID := range roles {
		if _, err := s.bus.RequestInterrupt(lease, eventbus.InterruptRequest{
			Scope:        eventbus.InterruptScopeRole,
			TargetRoleID: roleID,
			Source:       humanGateSource,
			Reason:       reason,
		}); err != nil {
			return requested, err
		}
		requested++
	}
	return requested, nil
}

func (s *Service) clearInterrupts(lease eventbus.Lease, roleID string) (int, error) {
	records, err := s.bus.ReadInterrupts()
	if err != nil {
		return 0, nil
	}
	cleared := 0
	for _, record := range records {
		if roleID != "" && record.TargetRoleID != roleID {
			continue
		}
		success, err := s.clearInterruptRecord(lease, record)
		if err != nil {
			return cleared, err
		}
		if success {
			cleared++
		}
	}
	return cleared, nil
}

func (s *Service) clearInterruptRecord(lease eventbus.Lease, record eventbus.InterruptRecord) (bool, error) {
	if record.Status == eventbus.InterruptStatusAcknowledged {
		_, err := s.bus.ClearInterrupt(lease, record.InterruptID)
		return err == nil, err
	}
	if record.Status != eventbus.InterruptStatusRequested {
		return false, nil
	}
	updated, err := s.bus.AcknowledgeInterrupt(lease, record.InterruptID)
	if err != nil {
		return false, err
	}
	_, err = s.bus.ClearInterrupt(lease, updated.InterruptID)
	return err == nil, err
}
