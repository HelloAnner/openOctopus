/*
Package recovery validate 负责恢复前的布局与事件链校验。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"os"
)

func (s *Service) validateSessionLayout() (sessionValidation, error) {
	metadata, err := readMetadata(s.paths.metadata)
	if err != nil {
		return sessionValidation{}, err
	}
	validation := sessionValidation{SessionID: metadata.SessionID, RepairableFiles: make([]string, 0), CheckedFiles: []string{"metadata.md"}}
	missing := make([]string, 0)
	mandatory := map[string]string{
		"state/effective_config.yaml": s.paths.effectiveConfig,
		"bus/events.md":               s.paths.events,
		"planner/master_schedule.md":  s.paths.schedule,
	}
	for ref, path := range mandatory {
		if _, statErr := os.Stat(path); statErr != nil {
			if os.IsNotExist(statErr) {
				missing = append(missing, ref)
				continue
			}
			return sessionValidation{}, statErr
		}
		validation.CheckedFiles = append(validation.CheckedFiles, ref)
	}
	if len(missing) != 0 {
		return sessionValidation{}, layoutInvalidError{Missing: missing}
	}
	if _, statErr := os.Stat(s.paths.sessionState); statErr != nil {
		if os.IsNotExist(statErr) {
			validation.RepairableFiles = append(validation.RepairableFiles, "session.state.md")
		} else {
			return sessionValidation{}, statErr
		}
	} else {
		validation.CheckedFiles = append(validation.CheckedFiles, "session.state.md")
	}
	if _, err := s.bus.List(); err != nil {
		return sessionValidation{}, err
	}
	return validation, nil
}
