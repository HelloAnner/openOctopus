package eventbus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var eventTypePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

type eventHashPayload struct {
	EventID          string `json:"event_id"`
	Sequence         int64  `json:"sequence"`
	Timestamp        string `json:"ts"`
	EventType        string `json:"event_type"`
	Producer         string `json:"producer"`
	SessionID        string `json:"session_id"`
	RoleID           string `json:"role_id"`
	CorrelationID    string `json:"correlation_id"`
	CausationEventID string `json:"causation_event_id"`
	PayloadRef       string `json:"payload_ref"`
	Summary          string `json:"summary"`
	PrevEventHash    string `json:"prev_event_hash"`
}

func (s *Store) List() ([]Event, error) {
	content, err := readMarkdown(s.paths.events)
	if err != nil {
		return nil, err
	}
	if isPlaceholderDocument(content) {
		return nil, ErrBusNotInitialized
	}
	return parseEvents(content)
}

func (s *Store) ListAfter(afterEventID string) ([]Event, error) {
	events, err := s.List()
	if err != nil {
		return nil, err
	}
	for index, event := range events {
		if event.EventID == afterEventID {
			return events[index+1:], nil
		}
	}
	return nil, fmt.Errorf("after event %s not found", afterEventID)
}

func (s *Store) Tail() (Event, error) {
	events, err := s.List()
	if err != nil {
		return Event{}, err
	}
	if len(events) == 0 {
		return Event{}, ErrBusNotInitialized
	}
	return events[len(events)-1], nil
}

func (s *Store) Append(lease Lease, appendEvent AppendEvent) (Event, error) {
	if !eventTypePattern.MatchString(appendEvent.EventType) {
		return Event{}, ErrInvalidEventType
	}
	if err := s.requireActiveLease(lease); err != nil {
		return Event{}, err
	}
	events, err := s.List()
	if err != nil {
		return Event{}, err
	}
	nextSequence := int64(len(events) + 1)
	prevHash := ""
	if len(events) != 0 {
		prevHash = events[len(events)-1].EventHash
	}
	event := Event{
		EventID:          fmt.Sprintf("event-%06d", nextSequence),
		Sequence:         nextSequence,
		Timestamp:        utcNow(),
		EventType:        appendEvent.EventType,
		Producer:         appendEvent.Producer,
		SessionID:        appendEvent.SessionID,
		RoleID:           appendEvent.RoleID,
		CorrelationID:    appendEvent.CorrelationID,
		CausationEventID: appendEvent.CausationEventID,
		PayloadRef:       appendEvent.PayloadRef,
		Summary:          appendEvent.Summary,
		PrevEventHash:    prevHash,
	}
	event.EventHash = hashEvent(event)
	events = append(events, event)
	if err := atomicWrite(s.paths.events, renderEvents(events)); err != nil {
		return Event{}, err
	}
	return event, nil
}

func parseEvents(content string) ([]Event, error) {
	blocks := splitMarkdownBlocks(content, "### ")
	events := make([]Event, 0, len(blocks))
	for _, block := range blocks {
		event, err := parseEventBlock(block)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := validateEventChain(events); err != nil {
		return nil, err
	}
	return events, nil
}

func parseEventBlock(block []string) (Event, error) {
	values := blockValues(block)
	sequence, err := strconv.ParseInt(values["sequence"], 10, 64)
	if err != nil {
		return Event{}, ErrEventChainBroken
	}
	event := Event{
		EventID:          values["event_id"],
		Sequence:         sequence,
		Timestamp:        values["ts"],
		EventType:        values["event_type"],
		Producer:         values["producer"],
		SessionID:        values["session_id"],
		RoleID:           values["role_id"],
		CorrelationID:    values["correlation_id"],
		CausationEventID: values["causation_event_id"],
		PayloadRef:       values["payload_ref"],
		Summary:          values["summary"],
		PrevEventHash:    values["prev_event_hash"],
		EventHash:        values["event_hash"],
	}
	return event, nil
}

func validateEventChain(events []Event) error {
	for index, event := range events {
		expectedSequence := int64(index + 1)
		if event.Sequence != expectedSequence || event.EventID != fmt.Sprintf("event-%06d", expectedSequence) {
			return ErrEventChainBroken
		}
		expectedPrevHash := ""
		if index > 0 {
			expectedPrevHash = events[index-1].EventHash
		}
		if event.PrevEventHash != expectedPrevHash {
			return ErrEventChainBroken
		}
		if hashEvent(event) != event.EventHash {
			return ErrEventChainBroken
		}
	}
	return nil
}

func hashEvent(event Event) string {
	payload := eventHashPayload{
		EventID:          event.EventID,
		Sequence:         event.Sequence,
		Timestamp:        event.Timestamp,
		EventType:        event.EventType,
		Producer:         event.Producer,
		SessionID:        event.SessionID,
		RoleID:           event.RoleID,
		CorrelationID:    event.CorrelationID,
		CausationEventID: event.CausationEventID,
		PayloadRef:       event.PayloadRef,
		Summary:          event.Summary,
		PrevEventHash:    event.PrevEventHash,
	}
	content, _ := json.Marshal(payload)
	hash := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(hash[:]))
}

func renderEventLines(event Event) []string {
	return []string{
		fmt.Sprintf("### %s", event.EventID),
		"",
		fmt.Sprintf("- event_id: %s", event.EventID),
		fmt.Sprintf("- sequence: %d", event.Sequence),
		fmt.Sprintf("- ts: %s", event.Timestamp),
		fmt.Sprintf("- event_type: %s", event.EventType),
		fmt.Sprintf("- producer: %s", event.Producer),
		fmt.Sprintf("- session_id: %s", event.SessionID),
		fmt.Sprintf("- role_id: %s", event.RoleID),
		fmt.Sprintf("- correlation_id: %s", event.CorrelationID),
		fmt.Sprintf("- causation_event_id: %s", event.CausationEventID),
		fmt.Sprintf("- payload_ref: %s", event.PayloadRef),
		fmt.Sprintf("- summary: %s", event.Summary),
		fmt.Sprintf("- prev_event_hash: %s", event.PrevEventHash),
		fmt.Sprintf("- event_hash: %s", event.EventHash),
	}
}

func splitMarkdownBlocks(content string, prefix string) [][]string {
	lines := strings.Split(content, "\n")
	blocks := make([][]string, 0)
	var current []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			if len(current) != 0 {
				blocks = append(blocks, current)
			}
			current = []string{trimmed}
			continue
		}
		if len(current) != 0 {
			current = append(current, trimmed)
		}
	}
	if len(current) != 0 {
		blocks = append(blocks, current)
	}
	return blocks
}

func blockValues(lines []string) map[string]string {
	values := make(map[string]string)
	for _, line := range lines {
		if !strings.HasPrefix(line, "- ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(line, "- "), ":", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return values
}
