/*
Package humangate helpers 提供人工 gate 读写 Markdown 所需的基础工具。
Author: Anner
Created on 2026/3/8
*/
package humangate

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

func appendBlock(path string, header string, lines []string) error {
	existing, err := readFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		existing = header + "\n"
	}
	trimmed := strings.TrimRight(existing, "\n")
	return atomicWrite(path, []byte(fmt.Sprintf("%s\n\n%s\n", trimmed, strings.Join(lines, "\n"))))
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

func atoi(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func placeholderDocument(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == "" || strings.Contains(trimmed, "Initialized by session 001.")
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
