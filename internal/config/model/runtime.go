package model

type RuntimeSection struct {
	Workspace   WorkspaceConfig   `koanf:"workspace" yaml:"workspace"`
	Scheduler   SchedulerConfig   `koanf:"scheduler" yaml:"scheduler"`
	RoleRuntime RoleRuntimeConfig `koanf:"role_runtime" yaml:"role_runtime"`
	MasterWatch MasterWatchConfig `koanf:"master_watch" yaml:"master_watch"`
}

type WorkspaceConfig struct {
	Root         string `koanf:"root" yaml:"root"`
	SessionsDir  string `koanf:"sessions_dir" yaml:"sessions_dir"`
	ArtifactsDir string `koanf:"artifacts_dir" yaml:"artifacts_dir"`
	LogsDir      string `koanf:"logs_dir" yaml:"logs_dir"`
}

type SchedulerConfig struct {
	MaxParallelRoles int    `koanf:"max_parallel_roles" yaml:"max_parallel_roles"`
	DispatchStrategy string `koanf:"dispatch_strategy" yaml:"dispatch_strategy"`
}

type RoleRuntimeConfig struct {
	TriggerMode         string   `koanf:"trigger_mode" yaml:"trigger_mode"`
	IdlePollSeconds     int      `koanf:"idle_poll_seconds" yaml:"idle_poll_seconds"`
	BootstrapReadOrder  []string `koanf:"bootstrap_read_order" yaml:"bootstrap_read_order"`
	SafeInterruptBoundary string `koanf:"safe_interrupt_boundary" yaml:"safe_interrupt_boundary"`
}

type MasterWatchConfig struct {
	Enabled               bool   `koanf:"enabled" yaml:"enabled"`
	ProgressFile          string `koanf:"progress_file" yaml:"progress_file"`
	BlockerFile           string `koanf:"blocker_file" yaml:"blocker_file"`
	AutoInterruptAllOnDeadlock bool `koanf:"auto_interrupt_all_on_deadlock" yaml:"auto_interrupt_all_on_deadlock"`
	MaxNoProgressRounds   int    `koanf:"max_no_progress_rounds" yaml:"max_no_progress_rounds"`
}
