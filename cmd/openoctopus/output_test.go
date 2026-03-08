package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestOutputWritesJSONSuccess(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := writeCommandSuccess(stdout, stderr, "json", "status", "ignored", map[string]any{
		"session_id":      "sess_123",
		"workflow_status": "RUNNING",
	})
	if err != nil {
		t.Fatalf("writeCommandSuccess returned error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not valid json: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", payload["ok"])
	}
	if payload["command"] != "status" {
		t.Fatalf("expected command status, got %#v", payload["command"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %#v", payload["data"])
	}
	if data["session_id"] != "sess_123" {
		t.Fatalf("expected session_id sess_123, got %#v", data["session_id"])
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestErrorWritesJSONFailure(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := renderCommandError(stdout, stderr, "json", "validate", newCLIError("config_validation_failed", "config validation failed", exitCodeConfigValidationFailed, errors.New("boom"), map[string]any{"error_count": 1}))
	if err == nil {
		t.Fatal("expected renderCommandError to return cli error")
	}

	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("expected CLIError, got %T", err)
	}
	if cliErr.ExitCode != exitCodeConfigValidationFailed {
		t.Fatalf("expected exit code %d, got %d", exitCodeConfigValidationFailed, cliErr.ExitCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &payload); err != nil {
		t.Fatalf("stderr is not valid json: %v", err)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false, got %#v", payload["ok"])
	}
	errorBody, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object, got %#v", payload["error"])
	}
	if errorBody["code"] != "config_validation_failed" {
		t.Fatalf("expected config_validation_failed, got %#v", errorBody["code"])
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
}

func TestOutputWritesTextSuccess(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := writeCommandSuccess(stdout, stderr, "text", "run", "session created: /tmp/sess_123", map[string]any{"session_id": "sess_123"})
	if err != nil {
		t.Fatalf("writeCommandSuccess returned error: %v", err)
	}
	if stdout.String() != "session created: /tmp/sess_123\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestExitCodeFallbacks(t *testing.T) {
	if got := exitCodeForError(nil); got != exitCodeSuccess {
		t.Fatalf("expected success exit code, got %d", got)
	}
	if got := exitCodeForError(newCLIError("session_not_found", "session not found", exitCodeSessionNotFound, nil, nil)); got != exitCodeSessionNotFound {
		t.Fatalf("expected session exit code, got %d", got)
	}
	if got := exitCodeForError(errors.New("boom")); got != exitCodeCommandFailed {
		t.Fatalf("expected generic exit code, got %d", got)
	}
}
