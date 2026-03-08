package artifact

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func (s *Store) ResolveLatest(name string) (ArtifactVersion, error) {
	versions, err := s.readVersions()
	if err != nil {
		return ArtifactVersion{}, err
	}
	latest := ArtifactVersion{}
	found := false
	for _, item := range versions {
		if item.ArtifactName != name {
			continue
		}
		if !found || item.Version > latest.Version {
			latest = item
			found = true
		}
	}
	if !found {
		return ArtifactVersion{}, ErrArtifactNotFound
	}
	return latest, nil
}

func (s *Store) readVersions() ([]ArtifactVersion, error) {
	if err := s.Bootstrap(); err != nil {
		return nil, err
	}
	content, err := readFile(s.indexPath)
	if err != nil {
		return nil, err
	}
	versions := make([]ArtifactVersion, 0)
	for _, block := range splitBlocks(content, "## artifact: ") {
		values := blockValues(block)
		versions = append(versions, ArtifactVersion{
			ArtifactName:   values["artifact_name"],
			Version:        atoi(values["version"]),
			StageID:        values["stage_id"],
			RoleID:         values["role_id"],
			TaskID:         values["task_id"],
			SourceRef:      values["source_ref"],
			ContentRef:     values["content_ref"],
			ManifestRef:    values["manifest_ref"],
			DiffRef:        values["diff_ref"],
			ContentHash:    values["content_hash"],
			SourceKind:     values["source_kind"],
			CreatedAt:      values["created_at"],
			PreviousVersion: atoi(values["previous_version"]),
			LineCount:      atoi(values["line_count"]),
			FileCount:      atoi(values["file_count"]),
		})
	}
	return versions, nil
}

func (s *Store) writeVersions(versions []ArtifactVersion) error {
	sort.Slice(versions, func(left int, right int) bool {
		if versions[left].ArtifactName == versions[right].ArtifactName {
			return versions[left].Version < versions[right].Version
		}
		return versions[left].ArtifactName < versions[right].ArtifactName
	})
	artifactNames := make(map[string]struct{})
	for _, item := range versions {
		artifactNames[item.ArtifactName] = struct{}{}
	}
	lines := []string{
		"# Artifact Index",
		"",
		fmt.Sprintf("- artifact_count: %d", len(artifactNames)),
		fmt.Sprintf("- version_count: %d", len(versions)),
		fmt.Sprintf("- updated_at: %s", utcNow()),
	}
	for _, item := range versions {
		lines = append(lines,
			"",
			fmt.Sprintf("## artifact: %s@%s", item.ArtifactName, formatVersion(item.Version)),
			fmt.Sprintf("- artifact_name: %s", item.ArtifactName),
			fmt.Sprintf("- version: %d", item.Version),
			fmt.Sprintf("- stage_id: %s", item.StageID),
			fmt.Sprintf("- role_id: %s", item.RoleID),
			fmt.Sprintf("- task_id: %s", item.TaskID),
			fmt.Sprintf("- source_ref: %s", item.SourceRef),
			fmt.Sprintf("- source_kind: %s", item.SourceKind),
			fmt.Sprintf("- content_ref: %s", item.ContentRef),
			fmt.Sprintf("- manifest_ref: %s", item.ManifestRef),
			fmt.Sprintf("- diff_ref: %s", item.DiffRef),
			fmt.Sprintf("- content_hash: %s", item.ContentHash),
			fmt.Sprintf("- previous_version: %d", item.PreviousVersion),
			fmt.Sprintf("- line_count: %d", item.LineCount),
			fmt.Sprintf("- file_count: %d", item.FileCount),
			fmt.Sprintf("- created_at: %s", item.CreatedAt),
		)
	}
	return atomicWrite(s.indexPath, []byte(strings.Join(lines, "\n")+"\n"))
}

func atoi(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}
