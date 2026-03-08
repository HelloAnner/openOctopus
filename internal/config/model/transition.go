package model

type TransitionConfig struct {
	From      string            `koanf:"from" yaml:"from"`
	To        string            `koanf:"to" yaml:"to"`
	Condition ConditionConfig   `koanf:"condition" yaml:"condition"`
	OnTrue    string            `koanf:"on_true" yaml:"on_true"`
	OnFalse   string            `koanf:"on_false" yaml:"on_false"`
}

type ConditionConfig struct {
	Type  string   `koanf:"type" yaml:"type"`
	Expr  string   `koanf:"expr" yaml:"expr"`
	Mode  string   `koanf:"mode" yaml:"mode"`
	Rules []string `koanf:"rules" yaml:"rules"`
}
