package roleruntime

import (
	"os"

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

func findRole(config model.RuntimeConfig, roleID string) (model.RoleConfig, model.LLMProfile, bool) {
	for _, role := range config.Roles {
		if role.ID != roleID {
			continue
		}
		profile, ok := config.LLMProfiles[role.LLMProfile]
		if !ok {
			return model.RoleConfig{}, model.LLMProfile{}, false
		}
		return role, profile, true
	}
	return model.RoleConfig{}, model.LLMProfile{}, false
}
