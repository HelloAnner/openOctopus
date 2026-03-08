package eventbus

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func atomicWrite(path string, content []byte) error {
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func readMarkdown(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func isPlaceholderDocument(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == "" || strings.Contains(content, "Initialized by session 001.")
}

func bulletValue(content string, key string) string {
	prefix := fmt.Sprintf("- %s:", key)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

func utcNow() string {
	return nowFunc().UTC().Format(time.RFC3339)
}

func (s *Store) ensureBusDirectory() error {
	return os.MkdirAll(filepath.Dir(s.paths.events), 0o755)
}

func (s *Store) readSessionID() (string, error) {
	content, err := readMarkdown(s.paths.metadata)
	if err != nil {
		return "", err
	}
	sessionID := bulletValue(content, "session_id")
	if sessionID == "" {
		return "", ErrBusNotInitialized
	}
	return sessionID, nil
}

func (s *Store) requireActiveLease(lease Lease) error {
	current, err := s.ReadLock()
	if err != nil {
		return err
	}
	if current.Status == LockStatusExpired {
		return ErrLeaseExpired
	}
	if current.Status != LockStatusHeld {
		return ErrLeaseConflict
	}
	if current.LeaseToken != lease.LeaseToken || current.LeaseVersion != lease.LeaseVersion || current.Holder != lease.Holder {
		return ErrLeaseConflict
	}
	return nil
}
