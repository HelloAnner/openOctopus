/*
Package recovery helpers 提供 recovery 读写与 Markdown 解析工具。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func atomicWrite(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
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

func splitBlocks(content string, prefix string) [][]string {
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

func atoi(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func relativeSessionPath(sessionDir string, target string) string {
	relative, err := filepath.Rel(sessionDir, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(relative)
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func formatCheckpointFileName(sequence int, kind string) string {
	return fmt.Sprintf("%04d-%s.md", sequence, kind)
}
