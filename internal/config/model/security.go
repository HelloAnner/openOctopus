package model

type SecurityConfig struct {
	Shell ShellSecurityConfig `koanf:"shell" yaml:"shell"`
	Path  PathSecurityConfig  `koanf:"path" yaml:"path"`
}

type ShellSecurityConfig struct {
	AllowlistPrefixes []string `koanf:"allowlist_prefixes" yaml:"allowlist_prefixes"`
	DenylistKeywords  []string `koanf:"denylist_keywords" yaml:"denylist_keywords"`
}

type PathSecurityConfig struct {
	WritableRoots []string `koanf:"writable_roots" yaml:"writable_roots"`
	ReadOnlyRoots []string `koanf:"read_only_roots" yaml:"read_only_roots"`
}
