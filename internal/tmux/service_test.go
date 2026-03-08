package tmux

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestFinalizeLayoutUsesPaneStartupScriptAndFocusesFirstRole(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{}
	sessionDir := t.TempDir()
	service := &Service{sessionDir: sessionDir, runner: runner}
	rolePanes := map[string]PaneBinding{
		"agent_a": {RoleID: "agent_a", PaneID: "%1", Title: "role:agent_a"},
		"agent_b": {RoleID: "agent_b", PaneID: "%2", Title: "role:agent_b"},
	}
	options := BootstrapOptions{
		SessionID:      "sess_001",
		RoleIDs:        []string{"agent_a", "agent_b"},
		MainPaneRatio:  0.5,
		RoleLayout:     "adaptive_grid",
		LaunchCommands: map[string]string{"agent_a": "printf 'boot' && codex 'agent_a'"},
	}

	err := service.finalizeLayout("octopus-sess_001", "octopus-sess_001", "%0", rolePanes, options)
	if err != nil {
		t.Fatalf("finalizeLayout returned error: %v", err)
	}
	if runner.ContainsCommand("send-keys") {
		t.Fatalf("expected startup without send-keys, got %#v", runner.calls)
	}
	if !runner.ContainsCommand("respawn-pane") {
		t.Fatalf("expected respawn-pane startup, got %#v", runner.calls)
	}
	scriptPath := filepath.Join(sessionDir, "state", "tmux", "scripts", "agent_a.sh")
	content, readErr := os.ReadFile(scriptPath)
	if readErr != nil {
		t.Fatalf("read startup script: %v", readErr)
	}
	if string(content) == "" || !containsAll(
		string(content),
		"printf 'boot'",
		"codex 'agent_a'",
		"planner/requirement.snapshot.md",
		"roles/agent_a/context.md",
		"roles/agent_a/inbox.md",
		"exec \"${SHELL:-/bin/zsh}\" -l",
	) {
		t.Fatalf("unexpected startup script content: %q", string(content))
	}
	last := runner.LastArgs()
	expectedLast := []string{"select-pane", "-t", "%1"}
	if !reflect.DeepEqual(last, expectedLast) {
		t.Fatalf("expected final focus %v, got %v", expectedLast, last)
	}
}

type recordingRunner struct {
	calls [][]string
}

func (r *recordingRunner) Run(_ string, args ...string) (string, error) {
	cloned := append([]string(nil), args...)
	r.calls = append(r.calls, cloned)
	return "", nil
}

func (r *recordingRunner) Contains(expected []string) bool {
	for _, call := range r.calls {
		if reflect.DeepEqual(call, expected) {
			return true
		}
	}
	return false
}

func (r *recordingRunner) ContainsCommand(command string) bool {
	for _, call := range r.calls {
		if len(call) != 0 && call[0] == command {
			return true
		}
	}
	return false
}

func (r *recordingRunner) LastArgs() []string {
	if len(r.calls) == 0 {
		return nil
	}
	return r.calls[len(r.calls)-1]
}

func containsAll(value string, items ...string) bool {
	for _, item := range items {
		if !strings.Contains(value, item) {
			return false
		}
	}
	return true
}
