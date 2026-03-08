package model

type ToolRegistry struct {
	Builtin map[string]BuiltinToolConfig `koanf:"builtin" yaml:"builtin"`
	MCP     map[string]MCPToolConfig     `koanf:"mcp" yaml:"mcp"`
}

type BuiltinToolConfig struct {
	Module string `koanf:"module" yaml:"module"`
	Class  string `koanf:"class" yaml:"class"`
}

type MCPToolConfig struct {
	Command string            `koanf:"command" yaml:"command"`
	Args    []string          `koanf:"args" yaml:"args"`
	Env     map[string]string `koanf:"env" yaml:"env"`
}
