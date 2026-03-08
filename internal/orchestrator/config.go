package orchestrator

import (
	"fmt"
	"os"
	"strings"

	"github.com/anner/openoctopus/internal/config/model"
	"gopkg.in/yaml.v3"
)

func (e *Engine) loadConfig() (model.RuntimeConfig, error) {
	content, err := os.ReadFile(e.paths.effectiveConfig)
	if err != nil {
		return model.RuntimeConfig{}, err
	}
	config := model.RuntimeConfig{}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return model.RuntimeConfig{}, err
	}
	return config, nil
}

func (e *Engine) readMetadataValue(key string) (string, error) {
	content, err := readFile(e.paths.metadata)
	if err != nil {
		return "", err
	}
	prefix := fmt.Sprintf("- %s:", key)
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix)), nil
		}
	}
	return "", ErrPlannerNotInitialized
}

func effectiveMaxParallelRoles(config model.RuntimeConfig) int {
	if config.Runtime.Scheduler.MaxParallelRoles > 0 {
		return config.Runtime.Scheduler.MaxParallelRoles
	}
	return 1
}
