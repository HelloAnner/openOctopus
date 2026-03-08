package eventbus

import (
	"fmt"
	"strings"
)

func (s *Store) ReadInterrupts() ([]InterruptRecord, error) {
	content, err := readMarkdown(s.paths.interrupts)
	if err != nil {
		return nil, err
	}
	if isPlaceholderDocument(content) {
		return nil, ErrBusNotInitialized
	}
	return parseInterrupts(content), nil
}

func (s *Store) RequestInterrupt(lease Lease, request InterruptRequest) (InterruptRecord, error) {
	if err := s.requireActiveLease(lease); err != nil {
		return InterruptRecord{}, err
	}
	sessionID, err := s.readSessionID()
	if err != nil {
		return InterruptRecord{}, err
	}
	event, err := s.Append(lease, AppendEvent{
		EventType:  "INTERRUPT_REQUESTED",
		Producer:   request.Source,
		SessionID:  sessionID,
		RoleID:     request.TargetRoleID,
		PayloadRef: "bus/interrupts.md",
		Summary:    request.Reason,
	})
	if err != nil {
		return InterruptRecord{}, err
	}
	records, err := s.ReadInterrupts()
	if err != nil {
		return InterruptRecord{}, err
	}
	record := InterruptRecord{
		InterruptID:    event.EventID,
		Scope:          request.Scope,
		TargetRoleID:   request.TargetRoleID,
		Source:         request.Source,
		Reason:         request.Reason,
		Status:         InterruptStatusRequested,
		RequestEventID: event.EventID,
		CreatedAt:      utcNow(),
		UpdatedAt:      utcNow(),
	}
	records = append(records, record)
	if err := atomicWrite(s.paths.interrupts, renderInterrupts(records)); err != nil {
		return InterruptRecord{}, err
	}
	return record, nil
}

func (s *Store) AcknowledgeInterrupt(lease Lease, interruptID string) (InterruptRecord, error) {
	return s.updateInterrupt(lease, interruptID, InterruptStatusRequested, InterruptStatusAcknowledged, "INTERRUPT_ACKNOWLEDGED")
}

func (s *Store) ClearInterrupt(lease Lease, interruptID string) (InterruptRecord, error) {
	return s.updateInterrupt(lease, interruptID, InterruptStatusAcknowledged, InterruptStatusCleared, "INTERRUPT_CLEARED")
}

func (s *Store) updateInterrupt(lease Lease, interruptID string, fromStatus string, toStatus string, eventType string) (InterruptRecord, error) {
	if err := s.requireActiveLease(lease); err != nil {
		return InterruptRecord{}, err
	}
	records, err := s.ReadInterrupts()
	if err != nil {
		return InterruptRecord{}, err
	}
	for index, item := range records {
		if item.InterruptID != interruptID {
			continue
		}
		if item.Status != fromStatus {
			return InterruptRecord{}, fmt.Errorf("invalid interrupt transition")
		}
		sessionID, readErr := s.readSessionID()
		if readErr != nil {
			return InterruptRecord{}, readErr
		}
		event, appendErr := s.Append(lease, AppendEvent{
			EventType:  eventType,
			Producer:   lease.Holder,
			SessionID:  sessionID,
			RoleID:     item.TargetRoleID,
			PayloadRef: "bus/interrupts.md",
			Summary:    fmt.Sprintf("interrupt %s", strings.ToLower(toStatus)),
		})
		if appendErr != nil {
			return InterruptRecord{}, appendErr
		}
		records[index].Status = toStatus
		records[index].UpdatedAt = utcNow()
		if toStatus == InterruptStatusAcknowledged {
			records[index].AckEventID = event.EventID
		}
		if toStatus == InterruptStatusCleared {
			records[index].ClearEventID = event.EventID
		}
		if err := atomicWrite(s.paths.interrupts, renderInterrupts(records)); err != nil {
			return InterruptRecord{}, err
		}
		return records[index], nil
	}
	return InterruptRecord{}, ErrInterruptNotFound
}

func parseInterrupts(content string) []InterruptRecord {
	blocks := splitMarkdownBlocks(content, "## interrupt: ")
	records := make([]InterruptRecord, 0, len(blocks))
	for _, block := range blocks {
		values := blockValues(block)
		records = append(records, InterruptRecord{
			InterruptID:    values["interrupt_id"],
			Scope:          values["scope"],
			TargetRoleID:   values["target_role_id"],
			Source:         values["source"],
			Reason:         values["reason"],
			Status:         values["status"],
			RequestEventID: values["request_event_id"],
			AckEventID:     values["ack_event_id"],
			ClearEventID:   values["clear_event_id"],
			CreatedAt:      values["created_at"],
			UpdatedAt:      values["updated_at"],
		})
	}
	return records
}

func renderInterrupts(records []InterruptRecord) []byte {
	lines := []string{"# Interrupts", ""}
	for index, item := range records {
		if index > 0 {
			lines = append(lines, "")
		}
		lines = append(lines,
			fmt.Sprintf("## interrupt: %s", item.InterruptID),
			fmt.Sprintf("- interrupt_id: %s", item.InterruptID),
			fmt.Sprintf("- scope: %s", item.Scope),
			fmt.Sprintf("- target_role_id: %s", item.TargetRoleID),
			fmt.Sprintf("- source: %s", item.Source),
			fmt.Sprintf("- reason: %s", item.Reason),
			fmt.Sprintf("- status: %s", item.Status),
			fmt.Sprintf("- request_event_id: %s", item.RequestEventID),
			fmt.Sprintf("- ack_event_id: %s", item.AckEventID),
			fmt.Sprintf("- clear_event_id: %s", item.ClearEventID),
			fmt.Sprintf("- created_at: %s", item.CreatedAt),
			fmt.Sprintf("- updated_at: %s", item.UpdatedAt),
		)
	}
	return []byte(fmt.Sprintf("%s\n", strings.Join(lines, "\n")))
}
