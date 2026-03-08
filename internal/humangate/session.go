/*
Package humangate session 提供 session 目录解析与服务构建入口。
Author: Anner
Created on 2026/3/8
*/
package humangate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anner/openoctopus/internal/eventbus"
)

// ResolveSessionDir 将 session 参数解析为真实的 session 目录。
func ResolveSessionDir(sessionRef string, workingDir string) (string, error) {
	if strings.TrimSpace(sessionRef) == "" {
		return "", fmt.Errorf("session is required")
	}
	if workingDir == "" {
		current, err := os.Getwd()
		if err != nil {
			return "", err
		}
		workingDir = current
	}
	for _, candidate := range resolveCandidates(sessionRef, workingDir) {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return filepath.Clean(candidate), nil
		}
	}
	return "", fmt.Errorf("session not found: %s", sessionRef)
}

// NewService 创建一个 human-gate 服务实例。
func NewService(sessionDir string) *Service {
	return &Service{
		sessionDir: sessionDir,
		paths: filePaths{
			humanMessages: filepath.Join(sessionDir, "planner", "human_messages.md"),
			schedule:      filepath.Join(sessionDir, "planner", "master_schedule.md"),
			blockers:      filepath.Join(sessionDir, "planner", "blockers.md"),
			decisionLog:   filepath.Join(sessionDir, "planner", "decision_log.md"),
			sessionState:  filepath.Join(sessionDir, "session.state.md"),
		},
		bus: eventbus.NewStore(sessionDir),
	}
}

func resolveCandidates(sessionRef string, workingDir string) []string {
	candidates := []string{sessionRef}
	if !filepath.IsAbs(sessionRef) {
		candidates = append(candidates, filepath.Join(workingDir, sessionRef))
	}
	if !strings.Contains(sessionRef, string(os.PathSeparator)) {
		candidates = append(candidates, filepath.Join(workingDir, ".octopus", "sessions", sessionRef))
	}
	return candidates
}
