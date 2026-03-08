package artifact

import (
    "errors"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/anner/openoctopus/internal/config/model"
    "github.com/anner/openoctopus/internal/session"
)

func TestBootstrapPromotesPlaceholderDocuments(t *testing.T) {
    result := createArtifactTestSession(t)

    store := NewStore(result.SessionDir)
    if err := store.Bootstrap(); err != nil {
        t.Fatalf("bootstrap: %v", err)
    }

    indexContent := readTestFile(t, filepath.Join(result.SessionDir, "artifacts", "index.md"))
    if strings.Contains(indexContent, "Initialized by session 001.") {
        t.Fatalf("expected index placeholder upgraded, got %q", indexContent)
    }
    if !strings.Contains(indexContent, "# Artifact Index") {
        t.Fatalf("expected artifact index header, got %q", indexContent)
    }

    lineageContent := readTestFile(t, filepath.Join(result.SessionDir, "audit", "lineage.md"))
    if strings.Contains(lineageContent, "Initialized by session 001.") {
        t.Fatalf("expected lineage placeholder upgraded, got %q", lineageContent)
    }
    if !strings.Contains(lineageContent, "# Artifact Lineage") {
        t.Fatalf("expected lineage header, got %q", lineageContent)
    }
}

func TestPublishFileArtifactAndResolveLatest(t *testing.T) {
    result := createArtifactTestSession(t)
    store := NewStore(result.SessionDir)
    if err := store.Bootstrap(); err != nil {
        t.Fatalf("bootstrap: %v", err)
    }

    sourceRef := filepath.ToSlash(filepath.Join("artifacts", "_staging", "stage_a", "artifact_a.md"))
    writeTestFile(t, filepath.Join(result.SessionDir, filepath.FromSlash(sourceRef)), "# draft\n\nhello artifact\n")

    published, err := store.Publish(PublishRequest{
        ArtifactName:  "artifact_a",
        StageID:       "stage_a",
        RoleID:        "agent_a",
        TaskID:        "task-stage_a-01",
        SourceRef:     sourceRef,
        ConclusionRef: "roles/agent_a/conclusion.md",
        TurnOutputRef: "roles/agent_a/turns/0001-output.md",
    })
    if err != nil {
        t.Fatalf("publish: %v", err)
    }
    if published.Version != 1 {
        t.Fatalf("expected version 1, got %+v", published)
    }
    if !strings.HasSuffix(published.ContentRef, "/content.md") {
        t.Fatalf("expected markdown content ref, got %+v", published)
    }

    content := readTestFile(t, filepath.Join(result.SessionDir, filepath.FromSlash(published.ContentRef)))
    if !strings.Contains(content, "hello artifact") {
        t.Fatalf("expected copied content, got %q", content)
    }

    manifest := readTestFile(t, filepath.Join(result.SessionDir, filepath.FromSlash(published.ManifestRef)))
    if !strings.Contains(manifest, "source_kind: file") {
        t.Fatalf("expected file manifest, got %q", manifest)
    }

    latest, err := store.ResolveLatest("artifact_a")
    if err != nil {
        t.Fatalf("resolve latest: %v", err)
    }
    if latest.Version != 1 || latest.ContentRef != published.ContentRef {
        t.Fatalf("expected latest to match publish, got %+v want %+v", latest, published)
    }
}

func TestPublishDirectoryArtifactCreatesVersionedSnapshot(t *testing.T) {
    result := createArtifactTestSession(t)
    store := NewStore(result.SessionDir)
    if err := store.Bootstrap(); err != nil {
        t.Fatalf("bootstrap: %v", err)
    }

    sourceDir := filepath.Join(result.SessionDir, "artifacts", "_staging", "stage_a", "artifact_tree")
    if err := os.MkdirAll(sourceDir, 0o755); err != nil {
        t.Fatalf("mkdir staging dir: %v", err)
    }
    writeTestFile(t, filepath.Join(sourceDir, "README.md"), "# tree\n")
    writeTestFile(t, filepath.Join(sourceDir, "src", "main.go"), "package main\n")

    published, err := store.Publish(PublishRequest{
        ArtifactName:  "artifact_tree",
        StageID:       "stage_a",
        RoleID:        "agent_a",
        TaskID:        "task-stage_a-01",
        SourceRef:     filepath.ToSlash(filepath.Join("artifacts", "_staging", "stage_a", "artifact_tree")),
        ConclusionRef: "roles/agent_a/conclusion.md",
        TurnOutputRef: "roles/agent_a/turns/0001-output.md",
    })
    if err != nil {
        t.Fatalf("publish dir: %v", err)
    }

    if _, statErr := os.Stat(filepath.Join(result.SessionDir, filepath.FromSlash(published.ContentRef), "README.md")); statErr != nil {
        t.Fatalf("expected directory snapshot to exist: %v", statErr)
    }
    manifest := readTestFile(t, filepath.Join(result.SessionDir, filepath.FromSlash(published.ManifestRef)))
    if !strings.Contains(manifest, "source_kind: directory") {
        t.Fatalf("expected directory manifest, got %q", manifest)
    }
}

func TestPublishSecondVersionWritesDiffSummary(t *testing.T) {
    result := createArtifactTestSession(t)
    store := NewStore(result.SessionDir)
    if err := store.Bootstrap(); err != nil {
        t.Fatalf("bootstrap: %v", err)
    }

    firstRef := filepath.ToSlash(filepath.Join("artifacts", "_staging", "stage_a", "artifact_a.md"))
    secondRef := filepath.ToSlash(filepath.Join("artifacts", "_staging", "stage_b", "artifact_a.md"))
    writeTestFile(t, filepath.Join(result.SessionDir, filepath.FromSlash(firstRef)), "# v1\n")
    if _, err := store.Publish(PublishRequest{ArtifactName: "artifact_a", StageID: "stage_a", RoleID: "agent_a", TaskID: "task-stage_a-01", SourceRef: firstRef, ConclusionRef: "roles/agent_a/conclusion.md", TurnOutputRef: "roles/agent_a/turns/0001-output.md"}); err != nil {
        t.Fatalf("publish v1: %v", err)
    }

    writeTestFile(t, filepath.Join(result.SessionDir, filepath.FromSlash(secondRef)), "# v2\n\nchanged\n")
    published, err := store.Publish(PublishRequest{ArtifactName: "artifact_a", StageID: "stage_b", RoleID: "agent_b", TaskID: "task-stage_b-01", SourceRef: secondRef, ConclusionRef: "roles/agent_b/conclusion.md", TurnOutputRef: "roles/agent_b/turns/0001-output.md"})
    if err != nil {
        t.Fatalf("publish v2: %v", err)
    }
    if published.Version != 2 {
        t.Fatalf("expected version 2, got %+v", published)
    }

    diffContent := readTestFile(t, filepath.Join(result.SessionDir, filepath.FromSlash(published.DiffRef)))
    if !strings.Contains(diffContent, "previous_hash:") || !strings.Contains(diffContent, "current_hash:") {
        t.Fatalf("expected diff summary, got %q", diffContent)
    }

    indexContent := readTestFile(t, filepath.Join(result.SessionDir, "artifacts", "index.md"))
    if !strings.Contains(indexContent, "version_count: 2") {
        t.Fatalf("expected index version count updated, got %q", indexContent)
    }
}

func TestResolveLatestReturnsNotFound(t *testing.T) {
    result := createArtifactTestSession(t)
    store := NewStore(result.SessionDir)
    if err := store.Bootstrap(); err != nil {
        t.Fatalf("bootstrap: %v", err)
    }

    _, err := store.ResolveLatest("missing_artifact")
    if !errors.Is(err, ErrArtifactNotFound) {
        t.Fatalf("expected ErrArtifactNotFound, got %v", err)
    }
}

func createArtifactTestSession(t *testing.T) session.CreateResult {
    t.Helper()
    tempDir := t.TempDir()
    configPath := filepath.Join(tempDir, "octopus.yaml")
    config := model.RuntimeConfig{
        Version: model.SupportedConfigVersion,
        Meta: model.MetaConfig{WorkflowID: "artifact-workflow", Name: "Artifact Workflow"},
        Runtime: model.RuntimeSection{Workspace: model.WorkspaceConfig{Root: ".octopus", SessionsDir: ".octopus/sessions"}},
    }
    result, err := session.Create(session.CreateOptions{Config: config, ConfigPath: configPath})
    if err != nil {
        t.Fatalf("create session: %v", err)
    }
    return result
}

func readTestFile(t *testing.T, path string) string {
    t.Helper()
    content, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("read %s: %v", path, err)
    }
    return string(content)
}

func writeTestFile(t *testing.T, path string, content string) {
    t.Helper()
    if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
        t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
    }
    if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
        t.Fatalf("write %s: %v", path, err)
    }
}
