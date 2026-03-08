/*
Package roleruntime executor_codex_test 验证 codex executor 的路径构造。
Author: Anner
Created on 2026/3/8
*/
package roleruntime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anner/openoctopus/internal/config/model"
)

func TestCodexExecutorUsesAbsoluteSessionAndOutputPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root, err := os.MkdirTemp(cwd, "executor-codex-*")
	if err != nil {
		t.Fatalf("mkdir temp root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	sessionDir := filepath.Join(root, "sessions", "sess_test")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}
	scriptPath := filepath.Join(root, "fake-codex.sh")
	script := `#!/bin/sh
cd_arg=""
out_arg=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -C)
      cd_arg="$2"
      shift 2
      ;;
    -o)
      out_arg="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
case "$cd_arg" in
  /*) ;;
  *) echo "relative -C path" >&2; exit 21 ;;
esac
case "$out_arg" in
  /*) ;;
  *) echo "relative -o path" >&2; exit 22 ;;
esac
printf '## role_result\n- status: SUCCESS\n- summary: ok\n- output_refs: \n' > "$out_arg"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	relativeSessionDir, err := filepath.Rel(cwd, sessionDir)
	if err != nil {
		t.Fatalf("relative session dir: %v", err)
	}

	result, execErr := codexExecutor{}.Execute(ExecuteRequest{
		SessionDir: relativeSessionDir,
		Profile:    model.LLMProfile{Provider: "codex", Mode: "cli", CLIPath: scriptPath},
		Prompt:     "say hi",
	})
	if execErr != nil {
		t.Fatalf("execute returned error: %v", execErr)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%q", result.ExitCode, result.Stderr)
	}
}
