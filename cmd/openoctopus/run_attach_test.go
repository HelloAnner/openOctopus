/*
Package main run_attach_test 验证 run 成功后的 tmux 自动进入行为。
Author: Anner
Created on 2026/3/8
*/
package main

import (
	"errors"
	"strings"
	"testing"

	configmodel "github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/session"
	"github.com/anner/openoctopus/internal/tmux"
	"github.com/spf13/cobra"
)

func TestHandleRunSuccessAutoAttachesInteractiveTmux(t *testing.T) {
	steps := make([]string, 0, 2)
	err := handleRunSuccess(newTestRunCommand(), "text", buildRunAttachConfig(true), buildRunAttachResult(), buildRunAttachTmuxResult(), runSuccessHooks{
		IsInteractive: func(_ *cobra.Command) bool {
			return true
		},
		WriteSuccess: func(_ *cobra.Command, _ string, _ session.CreateResult) error {
			steps = append(steps, "write")
			return nil
		},
		AttachTmux: func(_ tmux.BootstrapResult) error {
			steps = append(steps, "attach")
			return nil
		},
		WriteWarning: func(_ *cobra.Command, _ string) error {
			steps = append(steps, "warn")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("handleRunSuccess returned error: %v", err)
	}
	assertRunAttachSteps(t, steps, "write,attach")
}

func TestHandleRunSuccessSkipsAttachForJSON(t *testing.T) {
	steps := make([]string, 0, 2)
	err := handleRunSuccess(newTestRunCommand(), "json", buildRunAttachConfig(true), buildRunAttachResult(), buildRunAttachTmuxResult(), runSuccessHooks{
		IsInteractive: func(_ *cobra.Command) bool {
			return true
		},
		WriteSuccess: func(_ *cobra.Command, _ string, _ session.CreateResult) error {
			steps = append(steps, "write")
			return nil
		},
		AttachTmux: func(_ tmux.BootstrapResult) error {
			steps = append(steps, "attach")
			return nil
		},
		WriteWarning: func(_ *cobra.Command, _ string) error {
			steps = append(steps, "warn")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("handleRunSuccess returned error: %v", err)
	}
	assertRunAttachSteps(t, steps, "write")
}

func TestHandleRunSuccessWarnsWhenAttachFails(t *testing.T) {
	steps := make([]string, 0, 3)
	err := handleRunSuccess(newTestRunCommand(), "text", buildRunAttachConfig(true), buildRunAttachResult(), buildRunAttachTmuxResult(), runSuccessHooks{
		IsInteractive: func(_ *cobra.Command) bool {
			return true
		},
		WriteSuccess: func(_ *cobra.Command, _ string, _ session.CreateResult) error {
			steps = append(steps, "write")
			return nil
		},
		AttachTmux: func(_ tmux.BootstrapResult) error {
			steps = append(steps, "attach")
			return errors.New("attach failed")
		},
		WriteWarning: func(_ *cobra.Command, message string) error {
			steps = append(steps, message)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("handleRunSuccess returned error: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected three steps, got %#v", steps)
	}
	if steps[0] != "write" || steps[1] != "attach" {
		t.Fatalf("unexpected step order: %#v", steps)
	}
	if !strings.Contains(steps[2], "tmux attach failed") {
		t.Fatalf("expected warning to mention tmux attach failure, got %q", steps[2])
	}
}

func newTestRunCommand() *cobra.Command {
	command := &cobra.Command{}
	return command
}

func buildRunAttachConfig(enabled bool) configmodel.RuntimeConfig {
	return configmodel.RuntimeConfig{Runtime: configmodel.RuntimeSection{Tmux: configmodel.TmuxConfig{Enabled: enabled}}}
}

func buildRunAttachResult() session.CreateResult {
	return session.CreateResult{SessionID: "sess_001", SessionDir: "/tmp/sess_001"}
}

func buildRunAttachTmuxResult() tmux.BootstrapResult {
	return tmux.BootstrapResult{SocketName: "octopus-sess_001", SessionName: "octopus-sess_001"}
}

func assertRunAttachSteps(t *testing.T, steps []string, expected string) {
	t.Helper()
	joined := strings.Join(steps, ",")
	if joined != expected {
		t.Fatalf("expected steps %q, got %q", expected, joined)
	}
}
