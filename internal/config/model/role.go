package model

type RoleConfig struct {
	ID           string            `koanf:"id" yaml:"id"`
	Name         string            `koanf:"name" yaml:"name"`
	Type         string            `koanf:"type" yaml:"type"`
	LLMProfile   string            `koanf:"llm_profile" yaml:"llm_profile"`
	SystemPrompt string            `koanf:"system_prompt" yaml:"system_prompt"`
	ResetPrompt  string            `koanf:"reset_prompt" yaml:"reset_prompt"`
	ReactConfig  map[string]any    `koanf:"react_config" yaml:"react_config"`
	Constraints  RoleConstraints   `koanf:"constraints" yaml:"constraints"`
	Tools        []string          `koanf:"tools" yaml:"tools"`
}

type RoleConstraints struct {
	MustReadFiles  []string `koanf:"must_read_files" yaml:"must_read_files"`
	ForbiddenWrites []string `koanf:"forbidden_writes" yaml:"forbidden_writes"`
}
