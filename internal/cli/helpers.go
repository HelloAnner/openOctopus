/*
Package cli helpers 提供 CLI 只读解析所需的轻量工具。
Author: Anner
Created on 2026/3/8
*/
package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readTextFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func leadingValues(content string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		parts := strings.SplitN(strings.TrimPrefix(trimmed, "- "), ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		values[key] = strings.TrimSpace(parts[1])
	}
	return values
}

func placeholderDocument(content string) bool {
	return strings.Contains(content, "Initialized by session 001.")
}

func atoi(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func resolveCandidates(sessionRef string, workingDir string) []string {
	candidates := []string{sessionRef}
	if !filepath.IsAbs(sessionRef) {
		candidates = append(candidates, filepath.Join(workingDir, sessionRef))
	}
	if !strings.Contains(sessionRef, string(os.PathSeparator)) {
		candidates = append(candidates, filepath.Join(workingDir, ".octopus", "sessions", sessionRef))
	}
	return candidates
}

func defaultBlockerSummary(content string) string {
	if content == "" || placeholderDocument(content) {
		return "clear"
	}
	return leadingValues(content)["summary"]
}
