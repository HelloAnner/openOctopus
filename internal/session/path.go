package session

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const sessionCreateRetryLimit = 3

var sessionIDFunc = func() string {
	return fmt.Sprintf("sess_%d", time.Now().UnixNano())
}

func resolveSessionsDir(configPath string, sessionsDir string) string {
	if filepath.IsAbs(sessionsDir) {
		return sessionsDir
	}
	return filepath.Join(filepath.Dir(configPath), sessionsDir)
}

func createSessionDir(sessionsDir string) (string, string, error) {
	for attempt := 0; attempt < sessionCreateRetryLimit; attempt++ {
		sessionID := sessionIDFunc()
		sessionDir := filepath.Join(sessionsDir, sessionID)
		if err := os.Mkdir(sessionDir, 0o755); err == nil {
			return sessionID, sessionDir, nil
		} else if !os.IsExist(err) {
			return "", "", err
		}
	}
	return "", "", fmt.Errorf("failed to allocate session directory after %d attempts", sessionCreateRetryLimit)
}

func buildCreateResult(sessionID string, sessionDir string) CreateResult {
	return CreateResult{
		SessionID:           sessionID,
		SessionDir:          sessionDir,
		MetadataPath:        filepath.Join(sessionDir, "metadata.md"),
		StatePath:           filepath.Join(sessionDir, "session.state.md"),
		TimelinePath:        filepath.Join(sessionDir, "timeline.md"),
		EffectiveConfigPath: filepath.Join(sessionDir, "state", "effective_config.yaml"),
		InitialCheckpoint:   filepath.Join(sessionDir, "state", "checkpoints", "0000-init.md"),
	}
}
