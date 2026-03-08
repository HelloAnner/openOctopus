package orchestrator

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func atomicWrite(path string, content []byte) error {
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func appendMarkdownBlock(path string, header string, lines []string) error {
	existing, err := readFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		existing = header + "\n"
	}
	trimmed := strings.TrimRight(existing, "\n")
	block := strings.Join(lines, "\n")
	content := fmt.Sprintf("%s\n\n%s\n", trimmed, block)
	return atomicWrite(path, []byte(content))
}

func utcNow() string {
	return nowFunc().UTC().Format(timeFormat())
}

func timeFormat() string {
	return "2006-01-02T15:04:05Z07:00"
}

func isPlaceholder(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == "" || strings.Contains(content, "Initialized by session 001.")
}

func splitBlocks(content string, prefix string) [][]string {
	lines := strings.Split(content, "\n")
	blocks := make([][]string, 0)
	current := make([]string, 0)
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

func leadingValues(content string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			break
		}
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(trimmed, "- "), ":", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return values
}

func atoi(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

func sortedKeys(values map[string]StageNode) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
