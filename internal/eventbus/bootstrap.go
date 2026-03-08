package eventbus

import (
	"fmt"
)

func (s *Store) Bootstrap(options BootstrapOptions) error {
	if err := s.ensureBusDirectory(); err != nil {
		return err
	}
	if err := s.bootstrapEvents(options); err != nil {
		return err
	}
	if err := s.bootstrapLock(); err != nil {
		return err
	}
	if err := s.bootstrapOffsets(); err != nil {
		return err
	}
	return s.bootstrapInterrupts()
}

func (s *Store) bootstrapEvents(options BootstrapOptions) error {
	content, err := readMarkdown(s.paths.events)
	if err == nil && !isPlaceholderDocument(content) {
		events, parseErr := s.List()
		if parseErr != nil {
			return parseErr
		}
		if len(events) != 0 {
			return nil
		}
	}
	initial := Event{
		EventID:       "event-000001",
		Sequence:      1,
		Timestamp:     utcNow(),
		EventType:     "SESSION_CREATED",
		Producer:      "session",
		SessionID:     options.SessionID,
		CorrelationID: "bootstrap",
		PayloadRef:    options.MetadataRef,
		Summary:       "session skeleton promoted to event bus",
	}
	initial.EventHash = hashEvent(initial)
	return atomicWrite(s.paths.events, renderEvents([]Event{initial}))
}

func (s *Store) bootstrapLock() error {
	content, err := readMarkdown(s.paths.lock)
	if err == nil && !isPlaceholderDocument(content) {
		_, parseErr := s.ReadLock()
		return parseErr
	}
	return atomicWrite(s.paths.lock, renderLock(Lease{Status: LockStatusFree, LeaseVersion: 0, LastOperation: "bootstrap"}))
}

func (s *Store) bootstrapOffsets() error {
	content, err := readMarkdown(s.paths.offsets)
	if err == nil && !isPlaceholderDocument(content) {
		_, parseErr := s.ReadOffsets()
		return parseErr
	}
	return atomicWrite(s.paths.offsets, renderOffsets(nil))
}

func (s *Store) bootstrapInterrupts() error {
	content, err := readMarkdown(s.paths.interrupts)
	if err == nil && !isPlaceholderDocument(content) {
		_, parseErr := s.ReadInterrupts()
		return parseErr
	}
	return atomicWrite(s.paths.interrupts, renderInterrupts(nil))
}

func renderEvents(events []Event) []byte {
	lines := []string{"# Bus Events", ""}
	for index, event := range events {
		if index > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, renderEventLines(event)...)
	}
	return []byte(fmt.Sprintf("%s\n", joinLines(lines)))
}

func joinLines(lines []string) string {
	result := ""
	for index, line := range lines {
		if index > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}
