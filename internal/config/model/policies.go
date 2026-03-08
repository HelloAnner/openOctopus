package model

type PoliciesConfig struct {
	Retry              RetryPolicy              `koanf:"retry" yaml:"retry"`
	Timeout            TimeoutPolicy            `koanf:"timeout" yaml:"timeout"`
	LoopGuard          LoopGuardPolicy          `koanf:"loop_guard" yaml:"loop_guard"`
	HumanGate          HumanGatePolicy          `koanf:"human_gate" yaml:"human_gate"`
	SessionReset       SessionResetPolicy       `koanf:"session_reset" yaml:"session_reset"`
	Review             ReviewPolicy             `koanf:"review" yaml:"review"`
	Artifact           ArtifactPolicy           `koanf:"artifact" yaml:"artifact"`
	ImmutableArtifacts ImmutableArtifactsPolicy `koanf:"immutable_artifacts" yaml:"immutable_artifacts"`
	DeadlockGuard      DeadlockGuardPolicy      `koanf:"deadlock_guard" yaml:"deadlock_guard"`
}

type RetryPolicy struct {
	MaxRetryPerStage int   `koanf:"max_retry_per_stage" yaml:"max_retry_per_stage"`
	BackoffSeconds   []int `koanf:"backoff_seconds" yaml:"backoff_seconds"`
}

type TimeoutPolicy struct {
	StageTimeoutSeconds         int `koanf:"stage_timeout_seconds" yaml:"stage_timeout_seconds"`
	RoleHeartbeatTimeoutSeconds int `koanf:"role_heartbeat_timeout_seconds" yaml:"role_heartbeat_timeout_seconds"`
}

type LoopGuardPolicy struct {
	MaxRoundsPerTask int     `koanf:"max_rounds_per_task" yaml:"max_rounds_per_task"`
	MinQualityGain   float64 `koanf:"min_quality_gain" yaml:"min_quality_gain"`
}

type HumanGatePolicy struct {
	OnHighRisk                bool    `koanf:"on_high_risk" yaml:"on_high_risk"`
	HighRiskThreshold         float64 `koanf:"high_risk_threshold" yaml:"high_risk_threshold"`
	WriteManualInputToMarkdown bool   `koanf:"write_manual_input_to_markdown" yaml:"write_manual_input_to_markdown"`
	MainAgentAckRequired      bool    `koanf:"main_agent_ack_required" yaml:"main_agent_ack_required"`
}

type SessionResetPolicy struct {
	Enabled          bool `koanf:"enabled" yaml:"enabled"`
	PreserveFiles    bool `koanf:"preserve_files" yaml:"preserve_files"`
	KeepTurnHistory  bool `koanf:"keep_turn_history" yaml:"keep_turn_history"`
}

type ReviewPolicy struct {
	RequireDesignApproval bool `koanf:"require_design_approval" yaml:"require_design_approval"`
	RequireCodeDocDiff    bool `koanf:"require_code_doc_diff" yaml:"require_code_doc_diff"`
}

type ArtifactPolicy struct {
	HashAlgo           string `koanf:"hash_algo" yaml:"hash_algo"`
	KeepLatestVersions int    `koanf:"keep_latest_versions" yaml:"keep_latest_versions"`
}

type ImmutableArtifactsPolicy struct {
	Paths        []string `koanf:"paths" yaml:"paths"`
	AllowWriters []string `koanf:"allow_writers" yaml:"allow_writers"`
}

type DeadlockGuardPolicy struct {
	Enabled             bool     `koanf:"enabled" yaml:"enabled"`
	MaxNoProgressRounds int      `koanf:"max_no_progress_rounds" yaml:"max_no_progress_rounds"`
	BlockedStatuses     []string `koanf:"blocked_statuses" yaml:"blocked_statuses"`
	OnTrigger           string   `koanf:"on_trigger" yaml:"on_trigger"`
}
