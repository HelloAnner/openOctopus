package model

type StageConfig struct {
	ID              string        `koanf:"id" yaml:"id"`
	Name            string        `koanf:"name" yaml:"name"`
	Role            string        `koanf:"role" yaml:"role"`
	Mode            string        `koanf:"mode" yaml:"mode"`
	ClearCLIContext bool          `koanf:"clear_cli_context" yaml:"clear_cli_context"`
	Preserve        PreserveSpec  `koanf:"preserve" yaml:"preserve"`
	Input           []StageIO     `koanf:"input" yaml:"input"`
	Output          []StageIO     `koanf:"output" yaml:"output"`
}

type PreserveSpec struct {
	Artifacts []string `koanf:"artifacts" yaml:"artifacts"`
}

type StageIO struct {
	Type   string `koanf:"type" yaml:"type"`
	Ref    string `koanf:"ref" yaml:"ref"`
	Name   string `koanf:"name" yaml:"name"`
	Path   string `koanf:"path" yaml:"path"`
	Access string `koanf:"access" yaml:"access"`
}
