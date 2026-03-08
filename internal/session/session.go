package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anner/openoctopus/internal/config/model"
)

func CreateSkeleton(config model.RuntimeConfig, configPath string) (string, error) {
	baseDir := filepath.Dir(configPath)
	sessionsDir := resolvePath(baseDir, config.Runtime.Workspace.SessionsDir)
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return "", err
	}
	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())
	sessionDir := filepath.Join(sessionsDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return "", err
	}
	metadataPath := filepath.Join(sessionDir, "metadata.md")
	metadata := fmt.Sprintf("# Session Metadata\n\n- session_id: %s\n- workflow_id: %s\n", sessionID, config.Meta.WorkflowID)
	if err := os.WriteFile(metadataPath, []byte(metadata), 0o644); err != nil {
		return "", err
	}
	return sessionDir, nil
}

func resolvePath(baseDir string, target string) string {
	if filepath.IsAbs(target) {
		return target
	}
	return filepath.Join(baseDir, target)
}
