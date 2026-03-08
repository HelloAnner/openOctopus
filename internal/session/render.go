package session

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var nowFunc = time.Now

type renderedFile struct {
	Path    string
	Content []byte
}

func renderSessionFiles(result CreateResult, options CreateOptions) ([]renderedFile, error) {
	createdAt := nowFunc().UTC().Format(time.RFC3339)
	effectiveConfig, effectiveConfigHash, err := marshalEffectiveConfig(options)
	if err != nil {
		return nil, err
	}
	files := []renderedFile{
		{Path: result.EffectiveConfigPath, Content: effectiveConfig},
		{Path: result.MetadataPath, Content: renderMetadata(result, options, createdAt, effectiveConfigHash)},
		{Path: result.StatePath, Content: renderSessionState(result, createdAt)},
		{Path: result.TimelinePath, Content: renderTimeline(result, createdAt)},
		{Path: result.InitialCheckpoint, Content: renderInitialCheckpoint(result, createdAt)},
	}
	for _, item := range buildPlaceholderFiles(result.SessionDir) {
		files = append(files, item)
	}
	return files, nil
}

func marshalEffectiveConfig(options CreateOptions) ([]byte, string, error) {
	yamlBytes, err := yaml.Marshal(options.Config)
	if err != nil {
		return nil, "", err
	}
	jsonBytes, err := json.Marshal(options.Config)
	if err != nil {
		return nil, "", err
	}
	hash := sha256.Sum256(jsonBytes)
	return yamlBytes, fmt.Sprintf("%x", hash), nil
}

func renderMetadata(result CreateResult, options CreateOptions, createdAt string, effectiveConfigHash string) []byte {
	lines := []string{
		"# Session Metadata",
		"",
		fmt.Sprintf("- session_id: %s", result.SessionID),
		fmt.Sprintf("- workflow_id: %s", options.Config.Meta.WorkflowID),
		fmt.Sprintf("- workflow_name: %s", options.Config.Meta.Name),
		"- status: INITIAL",
		fmt.Sprintf("- created_at: %s", createdAt),
		fmt.Sprintf("- config_path: %s", options.ConfigPath),
		fmt.Sprintf("- sessions_dir: %s", resolveSessionsDir(options.ConfigPath, options.Config.Runtime.Workspace.SessionsDir)),
		fmt.Sprintf("- workspace_root: %s", options.Config.Runtime.Workspace.Root),
		fmt.Sprintf("- effective_config_path: %s", sessionRelativePath(result.SessionDir, result.EffectiveConfigPath)),
		fmt.Sprintf("- effective_config_hash: %s", effectiveConfigHash),
		fmt.Sprintf("- applied_defaults_count: %d", len(options.AppliedDefaults)),
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderSessionState(result CreateResult, createdAt string) []byte {
	lines := []string{
		"# Session State",
		"",
		fmt.Sprintf("- session_id: %s", result.SessionID),
		"- status: INITIAL",
		"- current_stage_id: ",
		"- current_role_id: ",
		"- checkpoint_seq: 0",
		"- last_event: SESSION_CREATED",
		fmt.Sprintf("- created_at: %s", createdAt),
		fmt.Sprintf("- updated_at: %s", createdAt),
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderTimeline(result CreateResult, createdAt string) []byte {
	lines := []string{
		"# Session Timeline",
		"",
		fmt.Sprintf("- at: %s", createdAt),
		"- event: SESSION_CREATED",
		fmt.Sprintf("- session_id: %s", result.SessionID),
		"- status: INITIAL",
		"- note: session skeleton initialized",
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderInitialCheckpoint(result CreateResult, createdAt string) []byte {
	lines := []string{
		"# Session Checkpoint 0000",
		"",
		"- checkpoint_seq: 0",
		fmt.Sprintf("- session_id: %s", result.SessionID),
		"- status: INITIAL",
		fmt.Sprintf("- created_at: %s", createdAt),
		fmt.Sprintf("- effective_config_path: %s", sessionRelativePath(result.SessionDir, result.EffectiveConfigPath)),
		"- timeline_head: SESSION_CREATED",
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func buildPlaceholderFiles(sessionDir string) []renderedFile {
	paths := make([]string, 0, len(placeholderFiles))
	for path := range placeholderFiles {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	files := make([]renderedFile, 0, len(paths))
	for _, path := range paths {
		files = append(files, renderedFile{
			Path:    filepath.Join(sessionDir, path),
			Content: placeholderDocument(placeholderFiles[path]),
		})
	}
	return files
}

func placeholderDocument(title string) []byte {
	return []byte(fmt.Sprintf("# %s\n\nInitialized by session 001.\n", title))
}

func sessionRelativePath(sessionDir string, targetPath string) string {
	relativePath, err := filepath.Rel(sessionDir, targetPath)
	if err != nil {
		return filepath.ToSlash(targetPath)
	}
	return filepath.ToSlash(relativePath)
}
