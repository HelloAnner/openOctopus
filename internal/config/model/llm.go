package model

type LLMProfile struct {
	Provider    string  `koanf:"provider" yaml:"provider"`
	Mode        string  `koanf:"mode" yaml:"mode"`
	CLIPath     string  `koanf:"cli_path" yaml:"cli_path"`
	MaxTokens   int     `koanf:"max_tokens" yaml:"max_tokens"`
	Temperature float64 `koanf:"temperature" yaml:"temperature"`
}
