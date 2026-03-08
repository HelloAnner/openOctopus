package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anner/openoctopus/internal/config/defaults"
	"github.com/anner/openoctopus/internal/config/model"
)

func TestResolveSessionsDir(t *testing.T) {
	configDir := filepath.Join(string(filepath.Separator), "tmp", "octopus")
	absoluteDir := filepath.Join(string(filepath.Separator), "data", "sessions")

	if got := resolveSessionsDir(filepath.Join(configDir, "octopus.yaml"), absoluteDir); got != absoluteDir {
		t.Fatalf("expected absolute sessions dir %q, got %q", absoluteDir, got)
	}

	relativeDir := filepath.Join(".octopus", "sessions")
	expected := filepath.Join(configDir, relativeDir)
	if got := resolveSessionsDir(filepath.Join(configDir, "octopus.yaml"), relativeDir); got != expected {
		t.Fatalf("expected relative sessions dir %q, got %q", expected, got)
	}
}

func TestCreateWritesSessionSkeleton(t *testing.T) {
	workingDir := t.TempDir()
	configPath := filepath.Join(workingDir, "octopus.yaml")
	if err := os.WriteFile(configPath, []byte("version: \"2.1\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousNow := nowFunc
	previousID := sessionIDFunc
	t.Cleanup(func() {
		nowFunc = previousNow
		sessionIDFunc = previousID
		writeFileFunc = writeFileAtomically
	})
	nowFunc = func() time.Time {
		return time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	}
	sessionIDFunc = func() string {
		return "sess_fixed"
	}

	result, err := Create(CreateOptions{
		Config:          buildSessionTestConfig(),
		ConfigPath:      configPath,
		AppliedDefaults: []defaults.AppliedDefault{{Path: "runtime.workspace.root", Value: ".octopus", Reason: "test"}},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if result.SessionID != "sess_fixed" {
		t.Fatalf("expected session id sess_fixed, got %q", result.SessionID)
	}

	requiredPaths := []string{
		result.MetadataPath,
		result.StatePath,
		result.TimelinePath,
		result.EffectiveConfigPath,
		result.InitialCheckpoint,
		filepath.Join(result.SessionDir, "planner", "human_messages.md"),
		filepath.Join(result.SessionDir, "bus", "events.md"),
		filepath.Join(result.SessionDir, "artifacts", "index.md"),
		filepath.Join(result.SessionDir, "audit", "lineage.md"),
	}
	for _, item := range requiredPaths {
		if _, statErr := os.Stat(item); statErr != nil {
			t.Fatalf("expected path %q to exist: %v", item, statErr)
		}
	}

	metadata, err := os.ReadFile(result.MetadataPath)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	assertContains(t, string(metadata), "- session_id: sess_fixed")
	assertContains(t, string(metadata), "- workflow_id: test-workflow")
	assertContains(t, string(metadata), "- status: INITIAL")
	assertContains(t, string(metadata), "- applied_defaults_count: 1")

	stateFile, err := os.ReadFile(result.StatePath)
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	assertContains(t, string(stateFile), "- status: INITIAL")
	assertContains(t, string(stateFile), "- last_event: SESSION_CREATED")

	timeline, err := os.ReadFile(result.TimelinePath)
	if err != nil {
		t.Fatalf("read timeline: %v", err)
	}
	assertContains(t, string(timeline), "SESSION_CREATED")

	checkpoint, err := os.ReadFile(result.InitialCheckpoint)
	if err != nil {
		t.Fatalf("read checkpoint: %v", err)
	}
	assertContains(t, string(checkpoint), "state/effective_config.yaml")

	effectiveConfig, err := os.ReadFile(result.EffectiveConfigPath)
	if err != nil {
		t.Fatalf("read effective config: %v", err)
	}
	assertContains(t, string(effectiveConfig), "workflow_id: test-workflow")
	assertContains(t, string(effectiveConfig), "sessions_dir: .octopus/sessions")
}

func TestCreateRemovesSessionDirectoryOnFailure(t *testing.T) {
	workingDir := t.TempDir()
	configPath := filepath.Join(workingDir, "octopus.yaml")
	if err := os.WriteFile(configPath, []byte("version: \"2.1\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	previousNow := nowFunc
	previousID := sessionIDFunc
	previousWriter := writeFileFunc
	t.Cleanup(func() {
		nowFunc = previousNow
		sessionIDFunc = previousID
		writeFileFunc = previousWriter
	})
	nowFunc = func() time.Time {
		return time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	}
	sessionIDFunc = func() string {
		return "sess_broken"
	}
	writeFileFunc = func(path string, content []byte) error {
		if strings.HasSuffix(path, "session.state.md") {
			return errors.New("boom")
		}
		return writeFileAtomically(path, content)
	}

	_, err := Create(CreateOptions{Config: buildSessionTestConfig(), ConfigPath: configPath})
	if err == nil {
		t.Fatal("expected Create to fail")
	}

	sessionDir := filepath.Join(workingDir, ".octopus", "sessions", "sess_broken")
	if _, statErr := os.Stat(sessionDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected session dir %q to be removed, stat err=%v", sessionDir, statErr)
	}
}

func buildSessionTestConfig() model.RuntimeConfig {
	return model.RuntimeConfig{
		Version: "2.1",
		Meta: model.MetaConfig{
			WorkflowID: "test-workflow",
			Name:       "Test Workflow",
		},
		Runtime: model.RuntimeSection{
			Workspace: model.WorkspaceConfig{
				Root:        ".octopus",
				SessionsDir: filepath.Join(".octopus", "sessions"),
			},
		},
	}
}

func assertContains(t *testing.T, content string, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected content to contain %q, got %q", expected, content)
	}
}
