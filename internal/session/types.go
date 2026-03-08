package session

import (
	"github.com/anner/openoctopus/internal/config/defaults"
	"github.com/anner/openoctopus/internal/config/model"
)

type CreateOptions struct {
	Config          model.RuntimeConfig
	ConfigPath      string
	AppliedDefaults []defaults.AppliedDefault
}

type CreateResult struct {
	SessionID           string
	SessionDir          string
	MetadataPath        string
	StatePath           string
	TimelinePath        string
	EffectiveConfigPath string
	InitialCheckpoint   string
}
