package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) Publish(request PublishRequest) (ArtifactVersion, error) {
	if err := s.Bootstrap(); err != nil {
		return ArtifactVersion{}, err
	}
	sourcePath, err := resolveSourcePath(s.sessionDir, request.SourceRef)
	if err != nil {
		return ArtifactVersion{}, err
	}
	versions, err := s.readVersions()
	if err != nil {
		return ArtifactVersion{}, err
	}
	previous, found := latestVersion(versions, request.ArtifactName)
	version := 1
	if found {
		version = previous.Version + 1
	}
	published, err := s.snapshotVersion(request, sourcePath, version, previous)
	if err != nil {
		return ArtifactVersion{}, err
	}
	versions = append(versions, published)
	if err := s.writeVersions(versions); err != nil {
		return ArtifactVersion{}, err
	}
	if err := s.appendLineage(published, request); err != nil {
		return ArtifactVersion{}, err
	}
	return published, nil
}

func latestVersion(versions []ArtifactVersion, name string) (ArtifactVersion, bool) {
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
	return latest, found
}

func (s *Store) snapshotVersion(request PublishRequest, sourcePath string, version int, previous ArtifactVersion) (ArtifactVersion, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return ArtifactVersion{}, err
	}
	versionDir := filepath.Join(s.artifactsDir, request.ArtifactName, formatVersion(version))
	createdAt := utcNow()
	published := ArtifactVersion{
		ArtifactName:  request.ArtifactName,
		Version:       version,
		StageID:       request.StageID,
		RoleID:        request.RoleID,
		TaskID:        request.TaskID,
		SourceRef:     sessionRelativePath(s.sessionDir, sourcePath),
		ManifestRef:   sessionRelativePath(s.sessionDir, filepath.Join(versionDir, "manifest.md")),
		DiffRef:       sessionRelativePath(s.sessionDir, filepath.Join(versionDir, "diff.md")),
		CreatedAt:     createdAt,
		PreviousVersion: previous.Version,
	}
	if info.IsDir() {
		published.SourceKind = "directory"
		contentDir := filepath.Join(versionDir, "content")
		if err := copyDirectory(sourcePath, contentDir); err != nil {
			return ArtifactVersion{}, err
		}
		hash, fileCount, err := hashDirectory(contentDir)
		if err != nil {
			return ArtifactVersion{}, err
		}
		published.ContentHash = hash
		published.ContentRef = sessionRelativePath(s.sessionDir, contentDir)
		published.FileCount = fileCount
	} else {
		published.SourceKind = "file"
		ext := filepath.Ext(sourcePath)
		if ext == "" {
			ext = ".txt"
		}
		contentPath := filepath.Join(versionDir, "content"+ext)
		if err := copyFile(sourcePath, contentPath); err != nil {
			return ArtifactVersion{}, err
		}
		hash, sizeBytes, lineCount, err := hashFile(contentPath)
		if err != nil {
			return ArtifactVersion{}, err
		}
		published.ContentHash = hash
		published.ContentRef = sessionRelativePath(s.sessionDir, contentPath)
		published.SizeBytes = sizeBytes
		published.LineCount = lineCount
	}
	if err := atomicWrite(filepath.Join(versionDir, "manifest.md"), []byte(renderManifest(published, request))); err != nil {
		return ArtifactVersion{}, err
	}
	if err := atomicWrite(filepath.Join(versionDir, "diff.md"), []byte(renderDiff(published, previous))); err != nil {
		return ArtifactVersion{}, err
	}
	return published, nil
}

func renderManifest(version ArtifactVersion, request PublishRequest) string {
	lines := []string{
		"# Artifact Manifest",
		"",
		fmt.Sprintf("- artifact_name: %s", version.ArtifactName),
		fmt.Sprintf("- version: %d", version.Version),
		fmt.Sprintf("- source_ref: %s", version.SourceRef),
		fmt.Sprintf("- source_kind: %s", version.SourceKind),
		fmt.Sprintf("- stage_id: %s", version.StageID),
		fmt.Sprintf("- role_id: %s", version.RoleID),
		fmt.Sprintf("- task_id: %s", version.TaskID),
		fmt.Sprintf("- conclusion_ref: %s", request.ConclusionRef),
		fmt.Sprintf("- turn_output_ref: %s", request.TurnOutputRef),
		fmt.Sprintf("- content_ref: %s", version.ContentRef),
		fmt.Sprintf("- content_hash: %s", version.ContentHash),
		fmt.Sprintf("- previous_version: %d", version.PreviousVersion),
		fmt.Sprintf("- created_at: %s", version.CreatedAt),
	}
	if version.SourceKind == "file" {
		lines = append(lines,
			fmt.Sprintf("- size_bytes: %d", version.SizeBytes),
			fmt.Sprintf("- line_count: %d", version.LineCount),
		)
	} else {
		lines = append(lines, fmt.Sprintf("- file_count: %d", version.FileCount))
	}
	return strings.Join(lines, "\n") + "\n"
}

func renderDiff(current ArtifactVersion, previous ArtifactVersion) string {
	if previous.Version == 0 {
		return "# Artifact Diff\n\n- summary: initial version\n"
	}
	lines := []string{
		"# Artifact Diff",
		"",
		fmt.Sprintf("- artifact_name: %s", current.ArtifactName),
		fmt.Sprintf("- previous_version: %d", previous.Version),
		fmt.Sprintf("- current_version: %d", current.Version),
		fmt.Sprintf("- previous_hash: %s", previous.ContentHash),
		fmt.Sprintf("- current_hash: %s", current.ContentHash),
		fmt.Sprintf("- changed: %t", previous.ContentHash != current.ContentHash),
		fmt.Sprintf("- previous_content_ref: %s", previous.ContentRef),
		fmt.Sprintf("- current_content_ref: %s", current.ContentRef),
	}
	if current.SourceKind == "directory" || previous.SourceKind == "directory" {
		lines = append(lines,
			fmt.Sprintf("- previous_file_count: %d", previous.FileCount),
			fmt.Sprintf("- current_file_count: %d", current.FileCount),
		)
	}
	return strings.Join(lines, "\n") + "\n"
}
