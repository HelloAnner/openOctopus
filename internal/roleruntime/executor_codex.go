package roleruntime

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type codexExecutor struct{}

func (codexExecutor) Execute(request ExecuteRequest) (ExecuteResult, error) {
	start := time.Now()
	sessionDir, err := filepath.Abs(request.SessionDir)
	if err != nil {
		return ExecuteResult{}, err
	}
	outputFile := filepath.Join(sessionDir, ".codex-last-message.txt")
	_ = os.Remove(outputFile)
	args := []string{
		"exec",
		"--full-auto",
		"--skip-git-repo-check",
		"--color", "never",
		"-C", sessionDir,
		"-o", outputFile,
		request.Prompt,
	}
	cmd := exec.Command(request.Profile.CLIPath, args...)
	cmd.Dir = sessionDir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return ExecuteResult{}, err
		}
	}
	rawOutput := stdout.String()
	if outputContent, readErr := os.ReadFile(outputFile); readErr == nil && strings.TrimSpace(string(outputContent)) != "" {
		rawOutput = string(outputContent)
	}
	if strings.TrimSpace(rawOutput) == "" {
		rawOutput = "## role_result\n- status: FAILED\n- summary: codex produced empty output\n- output_refs: "
		if exitCode == 0 {
			exitCode = 1
		}
	}
	return ExecuteResult{
		Provider:   "codex",
		Command:    fmt.Sprintf("%s %s", request.Profile.CLIPath, strings.Join(args, " ")),
		ExitCode:   exitCode,
		DurationMS: time.Since(start).Milliseconds(),
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		RawOutput:  rawOutput,
	}, nil
}
