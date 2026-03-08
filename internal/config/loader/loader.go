package loader

import (
	"fmt"
	"strings"

	configerrors "github.com/anner/openoctopus/internal/config/errors"
	"github.com/anner/openoctopus/internal/config/model"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Options struct {
	ConfigPath   string
	Environment  map[string]string
	FlagOverrides map[string]any
}

func Load(options Options) (model.RuntimeConfig, error) {
	instance := koanf.NewWithConf(koanf.Conf{Delim: ".", StrictMerge: true})
	if err := instance.Load(file.Provider(options.ConfigPath), yaml.Parser()); err != nil {
		return model.RuntimeConfig{}, configerrors.ConfigError{
			Code:       "CFG-SYNTAX-001",
			Category:   configerrors.CategorySyntax,
			Path:       "config",
			Message:    fmt.Sprintf("无法读取或解析配置文件: %v", err),
			Suggestion: "检查 octopus.yaml 的路径与 YAML 语法。",
			RuleID:     "ROOT-001",
		}
	}
	if err := loadOverrideMap(instance, buildEnvMap(options.Environment)); err != nil {
		return model.RuntimeConfig{}, err
	}
	if err := loadOverrideMap(instance, options.FlagOverrides); err != nil {
		return model.RuntimeConfig{}, err
	}
	config := model.RuntimeConfig{}
	if err := instance.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "koanf"}); err != nil {
		return model.RuntimeConfig{}, configerrors.ConfigError{
			Code:       "CFG-SCHEMA-001",
			Category:   configerrors.CategorySchema,
			Path:       "config",
			Message:    fmt.Sprintf("配置无法映射到 RuntimeConfig: %v", err),
			Suggestion: "检查字段类型与层级是否符合 yaml-rules.md。",
			RuleID:     "ROOT-002",
		}
	}
	return config, nil
}

func loadOverrideMap(instance *koanf.Koanf, values map[string]any) error {
	if len(values) == 0 {
		return nil
	}
	return instance.Load(confmap.Provider(values, "."), nil)
}

func buildEnvMap(environment map[string]string) map[string]any {
	values := make(map[string]any)
	for key, value := range environment {
		if !strings.HasPrefix(key, "OPENOCTOPUS_") {
			continue
		}
		trimmed := strings.TrimPrefix(key, "OPENOCTOPUS_")
		path := strings.ToLower(strings.ReplaceAll(trimmed, "__", "."))
		values[path] = parseScalar(value)
	}
	return values
}

func parseScalar(value string) any {
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, item := range parts {
			result = append(result, strings.TrimSpace(item))
		}
		return result
	}
	var intValue int
	if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil && fmt.Sprintf("%d", intValue) == value {
		return intValue
	}
	var floatValue float64
	if _, err := fmt.Sscanf(value, "%f", &floatValue); err == nil && strings.Contains(value, ".") {
		return floatValue
	}
	return value
}
