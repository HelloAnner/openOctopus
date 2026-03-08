package defaults

import "github.com/anner/openoctopus/internal/config/model"

type AppliedDefault struct {
	Path   string
	Value  any
	Reason string
}

func Apply(config *model.RuntimeConfig) []AppliedDefault {
	applied := make([]AppliedDefault, 0)
	applyString(&config.Runtime.Workspace.Root, ".octopus", "runtime.workspace.root", &applied)
	applyString(&config.Runtime.Workspace.SessionsDir, ".octopus/sessions", "runtime.workspace.sessions_dir", &applied)
	applyString(&config.Runtime.Workspace.ArtifactsDir, ".octopus/artifacts", "runtime.workspace.artifacts_dir", &applied)
	applyString(&config.Runtime.Workspace.LogsDir, ".octopus/logs", "runtime.workspace.logs_dir", &applied)
	applyString(&config.Runtime.Tmux.SocketName, "octopus-{session_id}", "runtime.tmux.socket_name", &applied)
	applyFloat(&config.Runtime.Tmux.MainPaneRatio, 0.5, "runtime.tmux.main_pane_ratio", &applied)
	applyString(&config.Runtime.Tmux.RoleLayout, "adaptive_grid", "runtime.tmux.role_layout", &applied)
	applyInt(&config.Runtime.RoleRuntime.IdlePollSeconds, 2, "runtime.role_runtime.idle_poll_seconds", &applied)
	applyBool(&config.Runtime.MasterWatch.Enabled, true, "runtime.master_watch.enabled", &applied)
	applyString(&config.Runtime.MasterWatch.ProgressFile, "planner/global_progress.md", "runtime.master_watch.progress_file", &applied)
	applyString(&config.Runtime.MasterWatch.BlockerFile, "planner/blockers.md", "runtime.master_watch.blocker_file", &applied)
	applyInt(&config.Runtime.MasterWatch.MaxNoProgressRounds, 3, "runtime.master_watch.max_no_progress_rounds", &applied)
	applyInt(&config.Policies.Retry.MaxRetryPerStage, 2, "policies.retry.max_retry_per_stage", &applied)
	applyIntSlice(&config.Policies.Retry.BackoffSeconds, []int{5, 20}, "policies.retry.backoff_seconds", &applied)
	applyInt(&config.Policies.Timeout.StageTimeoutSeconds, 1800, "policies.timeout.stage_timeout_seconds", &applied)
	applyInt(&config.Policies.Timeout.RoleHeartbeatTimeoutSeconds, 120, "policies.timeout.role_heartbeat_timeout_seconds", &applied)
	applyInt(&config.Policies.LoopGuard.MaxRoundsPerTask, 6, "policies.loop_guard.max_rounds_per_task", &applied)
	applyFloat(&config.Policies.LoopGuard.MinQualityGain, 0.05, "policies.loop_guard.min_quality_gain", &applied)
	return applied
}

func applyString(target *string, value string, path string, applied *[]AppliedDefault) {
	if *target != "" {
		return
	}
	*target = value
	*applied = append(*applied, AppliedDefault{Path: path, Value: value, Reason: "使用 config 001 保守默认值"})
}

func applyInt(target *int, value int, path string, applied *[]AppliedDefault) {
	if *target != 0 {
		return
	}
	*target = value
	*applied = append(*applied, AppliedDefault{Path: path, Value: value, Reason: "使用 config 001 保守默认值"})
}

func applyFloat(target *float64, value float64, path string, applied *[]AppliedDefault) {
	if *target != 0 {
		return
	}
	*target = value
	*applied = append(*applied, AppliedDefault{Path: path, Value: value, Reason: "使用 config 001 保守默认值"})
}

func applyBool(target *bool, value bool, path string, applied *[]AppliedDefault) {
	if *target {
		return
	}
	*target = value
	*applied = append(*applied, AppliedDefault{Path: path, Value: value, Reason: "使用 config 001 保守默认值"})
}

func applyIntSlice(target *[]int, value []int, path string, applied *[]AppliedDefault) {
	if len(*target) != 0 {
		return
	}
	*target = append([]int(nil), value...)
	*applied = append(*applied, AppliedDefault{Path: path, Value: value, Reason: "使用 config 001 保守默认值"})
}
