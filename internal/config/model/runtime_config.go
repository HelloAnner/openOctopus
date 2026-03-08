package model

type RuntimeConfig struct {
	Version     string                 `koanf:"version" yaml:"version"`
	Meta        MetaConfig             `koanf:"meta" yaml:"meta"`
	Runtime     RuntimeSection         `koanf:"runtime" yaml:"runtime"`
	LLMProfiles map[string]LLMProfile  `koanf:"llm_profiles" yaml:"llm_profiles"`
	ToolRegistry ToolRegistry          `koanf:"tool_registry" yaml:"tool_registry"`
	Security    SecurityConfig         `koanf:"security" yaml:"security"`
	Policies    PoliciesConfig         `koanf:"policies" yaml:"policies"`
	Roles       []RoleConfig           `koanf:"roles" yaml:"roles"`
	Stages      []StageConfig          `koanf:"stages" yaml:"stages"`
	Transitions []TransitionConfig     `koanf:"transitions" yaml:"transitions"`
}

const (
	SupportedConfigVersion = "2.1"
	EndStage               = "__END__"
)
