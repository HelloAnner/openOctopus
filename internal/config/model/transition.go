package model

type TransitionConfig struct {
	From      string          `koanf:"from" yaml:"from"`
	To        string          `koanf:"to" yaml:"to"`
	Condition ConditionConfig `koanf:"condition" yaml:"condition"`
	OnTrue    string          `koanf:"on_true" yaml:"on_true"`
	OnFalse   string          `koanf:"on_false" yaml:"on_false"`
	Repeat    RepeatConfig    `koanf:"repeat" yaml:"repeat"`
}

type RepeatConfig struct {
	MaxRounds  int    `koanf:"max_rounds" yaml:"max_rounds"`
	OnComplete string `koanf:"on_complete" yaml:"on_complete"`
}

type ConditionConfig struct {
	Type  string   `koanf:"type" yaml:"type"`
	Expr  string   `koanf:"expr" yaml:"expr"`
	Mode  string   `koanf:"mode" yaml:"mode"`
	Rules []string `koanf:"rules" yaml:"rules"`
}
