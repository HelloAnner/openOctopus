package eventbus

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func (s *Store) ReadOffsets() ([]OffsetEntry, error) {
	content, err := readMarkdown(s.paths.offsets)
	if err != nil {
		return nil, err
	}
	if isPlaceholderDocument(content) {
		return nil, ErrBusNotInitialized
	}
	return parseOffsets(content)
}

func (s *Store) CommitOffset(lease Lease, commit OffsetCommit) error {
	if err := s.requireActiveLease(lease); err != nil {
		return err
	}
	offsets, err := s.ReadOffsets()
	if err != nil {
		return err
	}
	updated := false
	for index, item := range offsets {
		if item.ConsumerID != commit.ConsumerID {
			continue
		}
		if commit.LastSequence < item.LastSequence {
			return ErrOffsetRegression
		}
		offsets[index] = OffsetEntry{
			ConsumerID:   commit.ConsumerID,
			LastEventID:  commit.LastEventID,
			LastSequence: commit.LastSequence,
			UpdatedAt:    utcNow(),
			Note:         commit.Note,
		}
		updated = true
		break
	}
	if !updated {
		offsets = append(offsets, OffsetEntry{
			ConsumerID:   commit.ConsumerID,
			LastEventID:  commit.LastEventID,
			LastSequence: commit.LastSequence,
			UpdatedAt:    utcNow(),
			Note:         commit.Note,
		})
	}
	sort.Slice(offsets, func(left int, right int) bool {
		return offsets[left].ConsumerID < offsets[right].ConsumerID
	})
	return atomicWrite(s.paths.offsets, renderOffsets(offsets))
}

func parseOffsets(content string) ([]OffsetEntry, error) {
	blocks := splitMarkdownBlocks(content, "## consumer: ")
	offsets := make([]OffsetEntry, 0, len(blocks))
	for _, block := range blocks {
		values := blockValues(block)
		lastSequence, err := strconv.ParseInt(defaultString(values["last_sequence"], "0"), 10, 64)
		if err != nil {
			return nil, ErrBusNotInitialized
		}
		offsets = append(offsets, OffsetEntry{
			ConsumerID:   values["consumer_id"],
			LastEventID:  values["last_event_id"],
			LastSequence: lastSequence,
			UpdatedAt:    values["updated_at"],
			Note:         values["note"],
		})
	}
	return offsets, nil
}

func renderOffsets(offsets []OffsetEntry) []byte {
	lines := []string{"# Bus Offsets", ""}
	for index, item := range offsets {
		if index > 0 {
			lines = append(lines, "")
		}
		lines = append(lines,
			fmt.Sprintf("## consumer: %s", item.ConsumerID),
			fmt.Sprintf("- consumer_id: %s", item.ConsumerID),
			fmt.Sprintf("- last_event_id: %s", item.LastEventID),
			fmt.Sprintf("- last_sequence: %d", item.LastSequence),
			fmt.Sprintf("- updated_at: %s", item.UpdatedAt),
			fmt.Sprintf("- note: %s", item.Note),
		)
	}
	return []byte(fmt.Sprintf("%s\n", strings.Join(lines, "\n")))
}
