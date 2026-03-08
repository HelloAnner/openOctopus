package session

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/anner/openoctopus/internal/config/model"
)

// Create 创建 session 001 约定的标准工作目录骨架与初始化文件。
func Create(options CreateOptions) (CreateResult, error) {
	if options.ConfigPath == "" {
		return CreateResult{}, errors.New("config path is required")
	}
	sessionsDir := resolveSessionsDir(options.ConfigPath, options.Config.Runtime.Workspace.SessionsDir)
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		return CreateResult{}, err
	}
	sessionID, sessionDir, err := createSessionDir(sessionsDir)
	if err != nil {
		return CreateResult{}, err
	}
	result := buildCreateResult(sessionID, sessionDir)
	if err := initializeSessionDirectories(result.SessionDir); err != nil {
		cleanupSessionDir(result.SessionDir)
		return CreateResult{}, err
	}
	files, err := renderSessionFiles(result, options)
	if err != nil {
		cleanupSessionDir(result.SessionDir)
		return CreateResult{}, err
	}
	if err := writeSessionFiles(files); err != nil {
		cleanupSessionDir(result.SessionDir)
		return CreateResult{}, err
	}
	return result, nil
}

// CreateSkeleton 兼容当前调用方，返回已创建的 session 目录。
func CreateSkeleton(config model.RuntimeConfig, configPath string) (string, error) {
	result, err := Create(CreateOptions{Config: config, ConfigPath: configPath})
	if err != nil {
		return "", err
	}
	return result.SessionDir, nil
}

func cleanupSessionDir(sessionDir string) {
	_ = os.RemoveAll(sessionDir)
}

func sessionJoin(sessionDir string, relativePath string) string {
	return filepath.Join(sessionDir, relativePath)
}
