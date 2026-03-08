package artifact

import (
	"fmt"
	"strings"
)

func (s *Store) appendLineage(version ArtifactVersion, request PublishRequest) error {
	content, err := readFile(s.lineagePath)
	if err != nil {
		return err
	}
	blocks := splitBlocks(content, "## record: ")
	recordCount := len(blocks) + 1
	lines := []string{
		"# Artifact Lineage",
		"",
		fmt.Sprintf("- record_count: %d", recordCount),
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}
	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed != "" {
			lines = append(lines, "", trimmed)
		}
	}
	lines = append(lines,
		"",
		fmt.Sprintf("## record: %s@%s", version.ArtifactName, formatVersion(version.Version)),
		fmt.Sprintf("- artifact_name: %s", version.ArtifactName),
		fmt.Sprintf("- version: %d", version.Version),
		fmt.Sprintf("- stage_id: %s", version.StageID),
		fmt.Sprintf("- role_id: %s", version.RoleID),
		fmt.Sprintf("- task_id: %s", version.TaskID),
		fmt.Sprintf("- source_ref: %s", version.SourceRef),
		fmt.Sprintf("- content_ref: %s", version.ContentRef),
		fmt.Sprintf("- manifest_ref: %s", version.ManifestRef),
		fmt.Sprintf("- conclusion_ref: %s", request.ConclusionRef),
		fmt.Sprintf("- turn_output_ref: %s", request.TurnOutputRef),
		fmt.Sprintf("- created_at: %s", version.CreatedAt),
	)
	return atomicWrite(s.lineagePath, []byte(strings.Join(lines, "\n")+"\n"))
}
