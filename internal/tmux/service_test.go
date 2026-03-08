package tmux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandSocketName(t *testing.T) {
	t.Parallel()

	value, err := expandSocketName("octopus-{session_id}", "sess_001")
	if err != nil {
		t.Fatalf("expandSocketName returned error: %v", err)
	}
	if value != "octopus-sess_001" {
		t.Fatalf("expected expanded socket, got %q", value)
	}
}

func TestReadLayoutParsesPaneMap(t *testing.T) {
	t.Parallel()

	sessionDir := t.TempDir()
	layoutDir := filepath.Join(sessionDir, "state", "tmux")
	if err := os.MkdirAll(layoutDir, 0o755); err != nil {
		t.Fatalf("mkdir layout dir: %v", err)
	}
	content := `# TMUX Layout

- session_id: sess_001
- socket_name: octopus-sess_001
- session_name: octopus-sess_001
- window_name: workspace
- role_layout: adaptive_grid
- main_pane_ratio: 0.5
- main_pane_id: %0
- updated_at: 2026-03-08T00:00:00Z

## Pane Map

### main
- role_id: main
- pane_id: %0
- title: main

### role
- role_id: agent_a
- pane_id: %1
- title: role:agent_a
`
	if err := os.WriteFile(filepath.Join(layoutDir, "layout.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write layout: %v", err)
	}

	layout, err := ReadLayout(sessionDir)
	if err != nil {
		t.Fatalf("ReadLayout returned error: %v", err)
	}
	if layout.SocketName != "octopus-sess_001" {
		t.Fatalf("expected socket name, got %q", layout.SocketName)
	}
	if layout.MainPaneID != "%0" {
		t.Fatalf("expected main pane id, got %q", layout.MainPaneID)
	}
	if layout.RolePanes["agent_a"].PaneID != "%1" {
		t.Fatalf("expected role pane id, got %q", layout.RolePanes["agent_a"].PaneID)
	}
}

func TestResolveTargetFailsForMissingRole(t *testing.T) {
	t.Parallel()

	layout := Layout{RolePanes: map[string]PaneBinding{"agent_a": {RoleID: "agent_a", PaneID: "%1", Title: "role:agent_a"}}}
	_, err := resolveTarget(layout, ResolveTargetOptions{RoleID: "agent_b"})
	if err == nil {
		t.Fatal("expected resolveTarget to fail")
	}
}
