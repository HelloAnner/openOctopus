package model

type MetaConfig struct {
	WorkflowID string `koanf:"workflow_id" yaml:"workflow_id"`
	Name       string `koanf:"name" yaml:"name"`
	Owner      string `koanf:"owner" yaml:"owner"`
}
