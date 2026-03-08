package eventbus

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/session"
)

func createEventBusTestSession(t *testing.T) session.CreateResult {
	t.Helper()

	workingDir := t.TempDir()
	configPath := filepath.Join(workingDir, "octopus.yaml")
	if err := os.WriteFile(configPath, []byte("version: \"2.1\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	result, err := session.Create(session.CreateOptions{
		Config:     buildEventBusTestConfig(),
		ConfigPath: configPath,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	return result
}

func buildEventBusTestConfig() model.RuntimeConfig {
	return model.RuntimeConfig{
		Version: model.SupportedConfigVersion,
		Meta: model.MetaConfig{
			WorkflowID: "eventbus-workflow",
			Name:       "Event Bus Workflow",
		},
		Runtime: model.RuntimeSection{
			Workspace: model.WorkspaceConfig{
				Root:        ".octopus",
				SessionsDir: filepath.Join(".octopus", "sessions"),
			},
		},
	}
}

func useFixedEventBusClock(t *testing.T) {
	t.Helper()

	previousNow := nowFunc
	previousToken := leaseTokenFunc
	nowFunc = func() time.Time {
		return time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	}
	leaseTokenFunc = func() string {
		return "lease-fixed-token"
	}
	t.Cleanup(func() {
		nowFunc = previousNow
		leaseTokenFunc = previousToken
	})
}

func bootstrapEventBusStore(t *testing.T) (*Store, session.CreateResult) {
	t.Helper()

	useFixedEventBusClock(t)
	result := createEventBusTestSession(t)
	store := NewStore(result.SessionDir)
	err := store.Bootstrap(BootstrapOptions{
		SessionID:   result.SessionID,
		SessionDir:  result.SessionDir,
		WorkflowID:  "eventbus-workflow",
		MetadataRef: "metadata.md",
	})
	if err != nil {
		t.Fatalf("bootstrap bus: %v", err)
	}
	return store, result
}
