/*
Package orchestrator input_refs 负责把外部输入快照到 session 内。
Author: Anner
Created on 2026/3/8
*/
package orchestrator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/anner/openoctopus/internal/config/model"
)

func materializeStageInputs(sessionDir string, metadataPath string, roleDir string, stage model.StageConfig) (model.StageConfig, error) {
	updated := stage
	for index, input := range stage.Input {
		if !shouldMaterializeStageInput(input) {
			continue
		}
		resolvedSource, err := resolveStageInputSource(metadataPath, input.Path)
		if err != nil {
			return model.StageConfig{}, err
		}
		copiedRef, err := copyStageInputFile(sessionDir, roleDir, resolvedSource)
		if err != nil {
			return model.StageConfig{}, err
		}
		updated.Input[index].Path = copiedRef
	}
	return updated, nil
}

func shouldMaterializeStageInput(input model.StageIO) bool {
	return strings.TrimSpace(input.Path) != "" && input.Type != "artifact"
}

func resolveStageInputSource(metadataPath string, inputPath string) (string, error) {
	metadata, err := readFile(metadataPath)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(inputPath) {
		return inputPath, nil
	}
	configPath := strings.TrimSpace(leadingValues(metadata)["config_path"])
	resolvedFromConfig, err := resolveRelativeToConfig(configPath, inputPath)
	if err == nil && fileExists(resolvedFromConfig) {
		return resolvedFromConfig, nil
	}
	resolvedFromWorkdir, workdirErr := filepath.Abs(inputPath)
	if workdirErr == nil && fileExists(resolvedFromWorkdir) {
		return resolvedFromWorkdir, nil
	}
	if err == nil {
		return resolvedFromConfig, nil
	}
	return "", err
}

func resolveRelativeToConfig(configPath string, inputPath string) (string, error) {
	baseDir := filepath.Dir(configPath)
	if !filepath.IsAbs(baseDir) {
		absoluteBaseDir, err := filepath.Abs(baseDir)
		if err != nil {
			return "", err
		}
		baseDir = absoluteBaseDir
	}
	return filepath.Clean(filepath.Join(baseDir, inputPath)), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyStageInputFile(sessionDir string, roleDir string, sourcePath string) (string, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", err
	}
	targetPath := filepath.Join(roleDir, "inputs", filepath.Base(sourcePath))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(targetPath, content, 0o644); err != nil {
		return "", err
	}
	relativePath, err := filepath.Rel(sessionDir, targetPath)
	if err != nil {
		return filepath.ToSlash(targetPath), nil
	}
	return filepath.ToSlash(relativePath), nil
}
