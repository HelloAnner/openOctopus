package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) Bootstrap() error {
	if err := os.MkdirAll(filepath.Join(s.artifactsDir, "_staging"), 0o755); err != nil {
		return err
	}
	if err := s.bootstrapIndex(); err != nil {
		return err
	}
	return s.bootstrapLineage()
}

func (s *Store) bootstrapIndex() error {
	content, err := readFile(s.indexPath)
	if err == nil && !isPlaceholder(content) && strings.Contains(content, "# Artifact Index") {
		return nil
	}
	lines := []string{
		"# Artifact Index",
		"",
		"- artifact_count: 0",
		"- version_count: 0",
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}
	return atomicWrite(s.indexPath, []byte(strings.Join(lines, "\n")+"\n"))
}

func (s *Store) bootstrapLineage() error {
	content, err := readFile(s.lineagePath)
	if err == nil && !isPlaceholder(content) && strings.Contains(content, "# Artifact Lineage") {
		return nil
	}
	lines := []string{
		"# Artifact Lineage",
		"",
		"- record_count: 0",
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}
	return atomicWrite(s.lineagePath, []byte(strings.Join(lines, "\n")+"\n"))
}
