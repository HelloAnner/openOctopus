package service

import (
	"os"

	"github.com/anner/openoctopus/internal/config/defaults"
	configerrors "github.com/anner/openoctopus/internal/config/errors"
	"github.com/anner/openoctopus/internal/config/loader"
	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/config/validator"
)

type LoadOptions struct {
	ConfigPath    string
	Environment   map[string]string
	FlagOverrides map[string]any
}

type ValidateResult struct {
	Config          model.RuntimeConfig
	AppliedDefaults []defaults.AppliedDefault
	Errors          []configerrors.ConfigError
}

func LoadForValidate(options LoadOptions) (ValidateResult, error) {
	config, err := loader.Load(loader.Options{
		ConfigPath:   options.ConfigPath,
		Environment:  normalizeEnvironment(options.Environment),
		FlagOverrides: options.FlagOverrides,
	})
	if err != nil {
		configError, ok := err.(configerrors.ConfigError)
		if !ok {
			return ValidateResult{}, err
		}
		return ValidateResult{Errors: []configerrors.ConfigError{configError}}, nil
	}
	appliedDefaults := defaults.Apply(&config)
	validationErrors := validator.Validate(config)
	return ValidateResult{Config: config, AppliedDefaults: appliedDefaults, Errors: validationErrors}, nil
}

func normalizeEnvironment(environment map[string]string) map[string]string {
	if environment != nil {
		return environment
	}
	result := make(map[string]string)
	for _, item := range os.Environ() {
		for index := 0; index < len(item); index++ {
			if item[index] != '=' {
				continue
			}
			result[item[:index]] = item[index+1:]
			break
		}
	}
	return result
}
