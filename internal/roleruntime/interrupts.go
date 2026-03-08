/*
Package roleruntime interrupts 处理人工中断的暂停闸门。
Author: Anner
Created on 2026/3/8
*/
package roleruntime

import (
	"time"

	"github.com/anner/openoctopus/internal/eventbus"
)

func (e *Engine) handleInterrupt(roleID string, state roleState) (RoleTickResult, error) {
	record, found, err := e.readActiveInterrupt(roleID)
	if err != nil || !found {
		return RoleTickResult{RoleID: roleID}, err
	}
	if record.Status == eventbus.InterruptStatusRequested {
		return e.acknowledgeRequestedInterrupt(roleID, state, record)
	}
	if record.Status == eventbus.InterruptStatusAcknowledged {
		if err := e.writeInterruptedState(roleID, state); err != nil {
			return RoleTickResult{}, err
		}
		return RoleTickResult{RoleID: roleID, Status: statusInterrupted, Skipped: true}, nil
	}
	return RoleTickResult{RoleID: roleID}, nil
}

func (e *Engine) readActiveInterrupt(roleID string) (eventbus.InterruptRecord, bool, error) {
	records, err := e.bus.ReadInterrupts()
	if err != nil {
		return eventbus.InterruptRecord{}, false, nil
	}
	for _, record := range records {
		if record.TargetRoleID != roleID {
			continue
		}
		if record.Status == eventbus.InterruptStatusRequested || record.Status == eventbus.InterruptStatusAcknowledged {
			return record, true, nil
		}
	}
	return eventbus.InterruptRecord{}, false, nil
}

func (e *Engine) acknowledgeRequestedInterrupt(roleID string, state roleState, record eventbus.InterruptRecord) (RoleTickResult, error) {
	if err := e.writeInterruptedState(roleID, state); err != nil {
		return RoleTickResult{}, err
	}
	lease, err := e.bus.AcquireLock("role-runtime/"+roleID, 30*time.Second)
	if err == nil {
		_, _ = e.bus.AcknowledgeInterrupt(lease, record.InterruptID)
		_ = e.bus.ReleaseLock(lease)
	}
	return RoleTickResult{RoleID: roleID, Progressed: true, Status: statusInterrupted, Skipped: true}, nil
}

func (e *Engine) writeInterruptedState(roleID string, state roleState) error {
	state.Status = statusInterrupted
	if err := writeRoleState(e.paths.rolesDir, state); err != nil {
		return err
	}
	return writeHeartbeat(e.paths.rolesDir, buildHeartbeat(roleID, state, 120))
}
