package artifact

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func NewStore(sessionDir string) *Store {
	artifactsDir := filepath.Join(sessionDir, "artifacts")
	return &Store{
		sessionDir:   sessionDir,
		artifactsDir: artifactsDir,
		lineagePath:  filepath.Join(sessionDir, "audit", "lineage.md"),
		indexPath:    filepath.Join(artifactsDir, "index.md"),
	}
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

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

func isPlaceholder(content string) bool {
	return strings.Contains(content, "Initialized by session 001.")
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

func splitBlocks(content string, prefix string) []string {
	blocks := make([]string, 0)
	current := make([]string, 0)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			if len(current) != 0 {
				blocks = append(blocks, strings.Join(current, "\n"))
			}
			current = []string{line}
			continue
		}
		if len(current) != 0 {
			current = append(current, line)
		}
	}
	if len(current) != 0 {
		blocks = append(blocks, strings.Join(current, "\n"))
	}
	return blocks
}

func blockValues(block string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
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

func formatVersion(version int) string {
	return fmt.Sprintf("%04d", version)
}

func sessionRelativePath(sessionDir string, targetPath string) string {
	relativePath, err := filepath.Rel(sessionDir, targetPath)
	if err != nil {
		return filepath.ToSlash(targetPath)
	}
	return filepath.ToSlash(relativePath)
}

func resolveSourcePath(sessionDir string, ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", ErrSourceNotFound
	}
	if filepath.IsAbs(ref) {
		if _, err := os.Stat(ref); err != nil {
			return "", ErrSourceNotFound
		}
		return ref, nil
	}
	candidate := filepath.Join(sessionDir, filepath.FromSlash(ref))
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", ErrSourceNotFound
}

func hashFile(path string) (string, int64, int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", 0, 0, err
	}
	sum := sha256.Sum256(content)
	return fmt.Sprintf("sha256:%x", sum), int64(len(content)), lineCount(string(content)), nil
}

func hashDirectory(path string) (string, int, error) {
	items := make([]string, 0)
	fileCount := 0
	err := filepath.Walk(path, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(path, current)
		if err != nil {
			return err
		}
		hash, _, _, err := hashFile(current)
		if err != nil {
			return err
		}
		items = append(items, filepath.ToSlash(relative)+"="+hash)
		fileCount++
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	sort.Strings(items)
	sum := sha256.Sum256([]byte(strings.Join(items, "\n")))
	return fmt.Sprintf("sha256:%x", sum), fileCount, nil
}

func copyFile(source string, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	return err
}

func copyDirectory(source string, target string) error {
	return filepath.Walk(source, func(current string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, current)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		return copyFile(current, destination)
	})
}

func lineCount(content string) int {
	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return 0
	}
	return strings.Count(trimmed, "\n") + 1
}
