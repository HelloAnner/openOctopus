package model

type LLMProfile struct {
	Provider    string  `koanf:"provider" yaml:"provider"`
	Mode        string  `koanf:"mode" yaml:"mode"`
	CLIPath     string  `koanf:"cli_path" yaml:"cli_path"`
	TmuxCommand string  `koanf:"tmux_command" yaml:"tmux_command"`
	MaxTokens   int     `koanf:"max_tokens" yaml:"max_tokens"`
	Temperature float64 `koanf:"temperature" yaml:"temperature"`
}
