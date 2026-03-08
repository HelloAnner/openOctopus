package roleruntime

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
	content := fmt.Sprintf("%s\n\n%s\n", trimmed, strings.Join(lines, "\n"))
	return atomicWrite(path, []byte(content))
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

func sectionText(content string, heading string) string {
	capture := false
	lines := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			capture = trimmed == heading
			continue
		}
		if capture {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func atoi(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func turnFileName(turnSeq int, suffix string) string {
	return fmt.Sprintf("%04d-%s.md", turnSeq, suffix)
}

func sanitizeRoleEnvKey(roleID string) string {
	upper := strings.ToUpper(roleID)
	replacer := strings.NewReplacer("-", "_", ".", "_", "/", "_")
	return replacer.Replace(upper)
}

func readMarkdown(path string) string {
	content, err := readFile(path)
	if err != nil {
		return ""
	}
	return content
}
