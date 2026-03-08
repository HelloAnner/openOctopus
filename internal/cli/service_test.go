package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSessionDirSupportsAbsoluteAndSessionID(t *testing.T) {
	workingDir := t.TempDir()
	sessionDir := filepath.Join(workingDir, ".octopus", "sessions", "sess_test")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}

	service := NewService(workingDir)
	resolved, err := service.ResolveSessionDir(sessionDir)
	if err != nil {
		t.Fatalf("ResolveSessionDir returned error: %v", err)
	}
	if resolved != sessionDir {
		t.Fatalf("expected absolute path %q, got %q", sessionDir, resolved)
	}

	resolved, err = service.ResolveSessionDir("sess_test")
	if err != nil {
		t.Fatalf("ResolveSessionDir by id returned error: %v", err)
	}
	if resolved != sessionDir {
		t.Fatalf("expected session id to resolve %q, got %q", sessionDir, resolved)
	}
}

func TestResolveSessionDirReturnsSessionNotFound(t *testing.T) {
	service := NewService(t.TempDir())

	_, err := service.ResolveSessionDir("missing")
	if err == nil {
		t.Fatal("expected missing session to fail")
	}
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestReadStatusAggregatesSessionFiles(t *testing.T) {
	sessionDir := prepareStatusSession(t)
	service := NewService(filepath.Dir(filepath.Dir(filepath.Dir(sessionDir))))

	summary, err := service.ReadStatus(sessionDir)
	if err != nil {
		t.Fatalf("ReadStatus returned error: %v", err)
	}
	if summary.SessionID != "sess_demo" {
		t.Fatalf("expected session id sess_demo, got %q", summary.SessionID)
	}
	if summary.WorkflowStatus != "WAITING_HUMAN" {
		t.Fatalf("expected workflow status WAITING_HUMAN, got %q", summary.WorkflowStatus)
	}
	if summary.ScheduleVersion != 3 {
		t.Fatalf("expected schedule version 3, got %d", summary.ScheduleVersion)
	}
	if summary.ActiveDispatchCount != 1 {
		t.Fatalf("expected active dispatch count 1, got %d", summary.ActiveDispatchCount)
	}
	if summary.BlockerSummary != "manual review" {
		t.Fatalf("expected blocker summary manual review, got %q", summary.BlockerSummary)
	}
	if summary.SessionDir != sessionDir {
		t.Fatalf("expected session dir %q, got %q", sessionDir, summary.SessionDir)
	}
}

func TestReadStatusFallsBackForPlaceholderFiles(t *testing.T) {
	sessionDir := prepareStatusSession(t)
	if err := os.WriteFile(filepath.Join(sessionDir, "planner", "master_schedule.md"), []byte("# Master Schedule\n\nInitialized by session 001.\n"), 0o644); err != nil {
		t.Fatalf("write placeholder schedule: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "planner", "blockers.md"), []byte("# Blockers\n\nInitialized by session 001.\n"), 0o644); err != nil {
		t.Fatalf("write placeholder blockers: %v", err)
	}

	service := NewService(t.TempDir())
	summary, err := service.ReadStatus(sessionDir)
	if err != nil {
		t.Fatalf("ReadStatus returned error: %v", err)
	}
	if summary.ScheduleVersion != 0 {
		t.Fatalf("expected placeholder schedule version 0, got %d", summary.ScheduleVersion)
	}
	if summary.BlockerSummary != "clear" {
		t.Fatalf("expected placeholder blocker summary clear, got %q", summary.BlockerSummary)
	}
}

func prepareStatusSession(t *testing.T) string {
	t.Helper()
	workingDir := t.TempDir()
	sessionDir := filepath.Join(workingDir, ".octopus", "sessions", "sess_demo")
	if err := os.MkdirAll(filepath.Join(sessionDir, "planner"), 0o755); err != nil {
		t.Fatalf("mkdir planner: %v", err)
	}
	writeStatusFile(t, filepath.Join(sessionDir, "metadata.md"), "# Session Metadata\n\n- session_id: sess_demo\n- workflow_id: wf-demo\n- workflow_name: Demo Workflow\n")
	writeStatusFile(t, filepath.Join(sessionDir, "session.state.md"), "# Session State\n\n- session_id: sess_demo\n- status: WAITING_HUMAN\n- current_stage_id: stage_a\n- current_role_id: agent_a\n- updated_at: 2026-03-08T12:00:00Z\n")
	writeStatusFile(t, filepath.Join(sessionDir, "planner", "master_schedule.md"), "# Master Schedule\n\n- schedule_version: 3\n- workflow_status: RUNNING\n- active_dispatch_count: 1\n- updated_at: 2026-03-08T12:00:01Z\n\n## stage: stage_a\n- stage_id: stage_a\n- role_id: agent_a\n- status: DISPATCHED\n")
	writeStatusFile(t, filepath.Join(sessionDir, "planner", "blockers.md"), "# Blockers\n\n- summary: manual review\n- updated_at: 2026-03-08T12:00:02Z\n")
	return sessionDir
}

func writeStatusFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
