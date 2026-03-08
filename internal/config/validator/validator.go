package validator

import (
	"fmt"
	"strings"

	configerrors "github.com/anner/openoctopus/internal/config/errors"
	"github.com/anner/openoctopus/internal/config/model"
)

func Validate(config model.RuntimeConfig) []configerrors.ConfigError {
	errors := make([]configerrors.ConfigError, 0)
	validateRoot(config, &errors)
	validateRoles(config, &errors)
	validateStages(config, &errors)
	validateTransitions(config, &errors)
	validateSecurity(config, &errors)
	validatePolicies(config, &errors)
	validateImmutableArtifacts(config, &errors)
	return errors
}

func validateRoot(config model.RuntimeConfig, errors *[]configerrors.ConfigError) {
	if config.Version != model.SupportedConfigVersion {
		appendError(errors, "CFG-SCHEMA-002", configerrors.CategorySchema, "version", "版本号不受支持", "使用 version: \"2.1\"。", "META-002")
	}
	if config.Meta.WorkflowID == "" {
		appendError(errors, "CFG-SCHEMA-003", configerrors.CategorySchema, "meta.workflow_id", "workflow_id 不能为空", "填写稳定的 workflow_id。", "META-003")
	}
	if config.Meta.Name == "" {
		appendError(errors, "CFG-SCHEMA-004", configerrors.CategorySchema, "meta.name", "name 不能为空", "填写工作流展示名称。", "META-004")
	}
	if len(config.LLMProfiles) == 0 {
		appendError(errors, "CFG-SCHEMA-005", configerrors.CategorySchema, "llm_profiles", "至少需要一个 llm profile", "定义至少一个 CLI profile。", "LLM-002")
	}
	if len(flattenTools(config.ToolRegistry)) == 0 {
		appendError(errors, "CFG-SCHEMA-006", configerrors.CategorySchema, "tool_registry", "至少需要一个工具注册项", "在 tool_registry 中定义至少一个可用工具。", "TOOL-001")
	}
	if len(config.Roles) == 0 {
		appendError(errors, "CFG-SCHEMA-007", configerrors.CategorySchema, "roles", "至少需要一个角色", "定义至少一个 role。", "ROLE-001")
	}
	if len(config.Stages) == 0 {
		appendError(errors, "CFG-SCHEMA-008", configerrors.CategorySchema, "stages", "至少需要一个阶段", "定义至少一个 stage。", "STAGE-001")
	}
	if len(config.Transitions) == 0 {
		appendError(errors, "CFG-SCHEMA-009", configerrors.CategorySchema, "transitions", "至少需要一个流转", "定义至少一个 transition。", "TRANS-001")
	}
	if config.Runtime.Scheduler.MaxParallelRoles < 0 {
		appendError(errors, "CFG-POLICY-001", configerrors.CategoryPolicy, "runtime.scheduler.max_parallel_roles", "并发角色数不能为负数", "移除该值或设置为正整数。", "RUNTIME-013")
	}
	if config.Runtime.Scheduler.MaxParallelRoles == 0 && hasFieldConfigured(config.Runtime.Scheduler) {
		appendError(errors, "CFG-POLICY-002", configerrors.CategoryPolicy, "runtime.scheduler.max_parallel_roles", "并发角色数必须大于 0", "设置为正整数，例如 1 或 2。", "RUNTIME-013")
	}
	for profileID, profile := range config.LLMProfiles {
		validateLLMProfile(profileID, profile, errors)
	}
}

func validateLLMProfile(profileID string, profile model.LLMProfile, errors *[]configerrors.ConfigError) {
	pathPrefix := fmt.Sprintf("llm_profiles.%s", profileID)
	if profile.Provider == "" {
		appendError(errors, "CFG-SCHEMA-010", configerrors.CategorySchema, pathPrefix+".provider", "provider 不能为空", "填写 provider，例如 codex。", "LLM-004")
	}
	if profile.Mode == "" {
		appendError(errors, "CFG-SCHEMA-011", configerrors.CategorySchema, pathPrefix+".mode", "mode 不能为空", "填写 mode，例如 cli。", "LLM-004")
	}
	if profile.Mode == "cli" && profile.CLIPath == "" {
		appendError(errors, "CFG-SCHEMA-012", configerrors.CategorySchema, pathPrefix+".cli_path", "cli 模式必须声明 cli_path", "为 CLI profile 设置可执行命令路径。", "LLM-005")
	}
	if profile.MaxTokens < 0 {
		appendError(errors, "CFG-POLICY-003", configerrors.CategoryPolicy, pathPrefix+".max_tokens", "max_tokens 不能为负数", "移除该值或设置为正整数。", "LLM-006")
	}
}

func validateRoles(config model.RuntimeConfig, errors *[]configerrors.ConfigError) {
	roleIDs := make(map[string]struct{})
	registeredTools := flattenTools(config.ToolRegistry)
	for index, role := range config.Roles {
		pathPrefix := fmt.Sprintf("roles[%d]", index)
		if role.ID == "" {
			appendError(errors, "CFG-SCHEMA-013", configerrors.CategorySchema, pathPrefix+".id", "role id 不能为空", "为角色填写唯一 id。", "ROLE-002")
			continue
		}
		if _, exists := roleIDs[role.ID]; exists {
			appendError(errors, "CFG-SCHEMA-014", configerrors.CategorySchema, pathPrefix+".id", "role id 必须全局唯一", "为重复角色使用新的 id。", "ROLE-003")
		}
		roleIDs[role.ID] = struct{}{}
		if role.Name == "" {
			appendError(errors, "CFG-SCHEMA-015", configerrors.CategorySchema, pathPrefix+".name", "role name 不能为空", "填写角色展示名。", "ROLE-002")
		}
		if role.Type == "" {
			appendError(errors, "CFG-SCHEMA-016", configerrors.CategorySchema, pathPrefix+".type", "role type 不能为空", "填写受支持的角色类型。", "ROLE-005")
		}
		if role.LLMProfile == "" {
			appendError(errors, "CFG-SCHEMA-017", configerrors.CategorySchema, pathPrefix+".llm_profile", "llm_profile 不能为空", "引用已定义的 llm profile。", "ROLE-004")
		} else if _, exists := config.LLMProfiles[role.LLMProfile]; !exists {
			appendError(errors, "CFG-REFERENCE-001", configerrors.CategoryReference, pathPrefix+".llm_profile", "角色引用了不存在的 llm profile", "修正 llm_profile 引用或补定义该 profile。", "ROLE-004")
		}
		if strings.TrimSpace(role.SystemPrompt) == "" {
			appendError(errors, "CFG-SCHEMA-018", configerrors.CategorySchema, pathPrefix+".system_prompt", "system_prompt 不能为空", "填写可被运行时消费的提示词。", "ROLE-006")
		}
		for toolIndex, toolID := range role.Tools {
			if _, exists := registeredTools[toolID]; exists {
				continue
			}
			appendError(errors, "CFG-REFERENCE-002", configerrors.CategoryReference, fmt.Sprintf("%s.tools[%d]", pathPrefix, toolIndex), "角色引用了未注册工具", "在 tool_registry 中补充工具注册，或修正工具 id。", "TOOL-005")
		}
	}
}

func validateStages(config model.RuntimeConfig, errors *[]configerrors.ConfigError) {
	roleIDs := make(map[string]struct{})
	artifactNames := make(map[string]struct{})
	for _, role := range config.Roles {
		roleIDs[role.ID] = struct{}{}
	}
	stageIDs := make(map[string]struct{})
	for index, stage := range config.Stages {
		pathPrefix := fmt.Sprintf("stages[%d]", index)
		if stage.ID == "" {
			appendError(errors, "CFG-SCHEMA-019", configerrors.CategorySchema, pathPrefix+".id", "stage id 不能为空", "为阶段填写唯一 id。", "STAGE-002")
			continue
		}
		if _, exists := stageIDs[stage.ID]; exists {
			appendError(errors, "CFG-SCHEMA-020", configerrors.CategorySchema, pathPrefix+".id", "stage id 必须全局唯一", "修正重复的阶段 id。", "STAGE-003")
		}
		stageIDs[stage.ID] = struct{}{}
		if stage.Name == "" {
			appendError(errors, "CFG-SCHEMA-021", configerrors.CategorySchema, pathPrefix+".name", "stage name 不能为空", "填写阶段展示名。", "STAGE-002")
		}
		if _, exists := roleIDs[stage.Role]; !exists {
			appendError(errors, "CFG-REFERENCE-003", configerrors.CategoryReference, pathPrefix+".role", "阶段引用了不存在的角色", "修正 stage.role 或补充目标角色。", "STAGE-004")
		}
		if stage.Mode == "session_reset" {
			validateSessionResetStage(stage, pathPrefix, errors)
		}
		for outputIndex, output := range stage.Output {
			if output.Type != "artifact" || output.Name == "" {
				continue
			}
			artifactNames[output.Name] = struct{}{}
			_ = outputIndex
		}
		for inputIndex, input := range stage.Input {
			if input.Type != "artifact" || input.Ref == "" {
				continue
			}
			if _, exists := artifactNames[input.Ref]; exists {
				continue
			}
			appendError(errors, "CFG-REFERENCE-004", configerrors.CategoryReference, fmt.Sprintf("%s.input[%d].ref", pathPrefix, inputIndex), "artifact 引用尚未由前序阶段产出", "确认 ref 指向已定义输出 artifact。", "STAGE-006")
		}
	}
}

func validateSessionResetStage(stage model.StageConfig, pathPrefix string, errors *[]configerrors.ConfigError) {
	if !stage.ClearCLIContext {
		appendError(errors, "CFG-SCHEMA-022", configerrors.CategorySchema, pathPrefix+".clear_cli_context", "session_reset 阶段必须声明 clear_cli_context=true", "为 session_reset 阶段显式声明 clear_cli_context。", "STAGE-011")
	}
	if len(stage.Preserve.Artifacts) == 0 {
		appendError(errors, "CFG-SCHEMA-023", configerrors.CategorySchema, pathPrefix+".preserve.artifacts", "session_reset 阶段必须声明保留策略", "至少声明需要保留的 artifacts。", "STAGE-012")
	}
}

func validateTransitions(config model.RuntimeConfig, errors *[]configerrors.ConfigError) {
	stageIDs := make(map[string]struct{})
	for _, stage := range config.Stages {
		stageIDs[stage.ID] = struct{}{}
	}
	for index, transition := range config.Transitions {
		pathPrefix := fmt.Sprintf("transitions[%d]", index)
		if _, exists := stageIDs[transition.From]; !exists {
			appendError(errors, "CFG-REFERENCE-005", configerrors.CategoryReference, pathPrefix+".from", "transition.from 引用了不存在的阶段", "修正 from 指向已定义阶段。", "TRANS-003")
		}
		validateTransitionTarget(pathPrefix+".to", transition.To, stageIDs, errors)
		validateTransitionTarget(pathPrefix+".on_true", transition.OnTrue, stageIDs, errors)
		validateTransitionTarget(pathPrefix+".on_false", transition.OnFalse, stageIDs, errors)
	}
}

func validateTransitionTarget(path string, target string, stageIDs map[string]struct{}, errors *[]configerrors.ConfigError) {
	if target == "" {
		return
	}
	if target == model.EndStage {
		return
	}
	if _, exists := stageIDs[target]; exists {
		return
	}
	appendError(errors, "CFG-REFERENCE-006", configerrors.CategoryReference, path, "transition 目标不存在", "修正目标阶段 id 或使用 __END__。", "TRANS-004")
}

func validateSecurity(config model.RuntimeConfig, errors *[]configerrors.ConfigError) {
	if !usesShellExec(config.Roles) {
		return
	}
	if len(config.Security.Shell.AllowlistPrefixes) == 0 {
		appendError(errors, "CFG-SECURITY-001", configerrors.CategorySecurity, "security.shell.allowlist_prefixes", "启用 shell_exec 时必须配置 allowlist_prefixes", "在 security.shell 中补充 allowlist_prefixes。", "SEC-001")
	}
}

func validatePolicies(config model.RuntimeConfig, errors *[]configerrors.ConfigError) {
	if config.Runtime.MasterWatch.MaxNoProgressRounds < 0 {
		appendError(errors, "CFG-POLICY-004", configerrors.CategoryPolicy, "runtime.master_watch.max_no_progress_rounds", "max_no_progress_rounds 不能为负数", "设置为正整数。", "RUNTIME-024")
	}
	if config.Runtime.MasterWatch.MaxNoProgressRounds == 0 && hasFieldConfigured(config.Runtime.MasterWatch) {
		appendError(errors, "CFG-POLICY-005", configerrors.CategoryPolicy, "runtime.master_watch.max_no_progress_rounds", "max_no_progress_rounds 必须大于 0", "设置为正整数。", "RUNTIME-024")
	}
	if config.Policies.DeadlockGuard.MaxNoProgressRounds < 0 {
		appendError(errors, "CFG-POLICY-006", configerrors.CategoryPolicy, "policies.deadlock_guard.max_no_progress_rounds", "deadlock_guard 阈值不能为负数", "设置为正整数。", "POL-013")
	}
	if config.Policies.DeadlockGuard.MaxNoProgressRounds == 0 && hasFieldConfigured(config.Policies.DeadlockGuard) {
		appendError(errors, "CFG-POLICY-007", configerrors.CategoryPolicy, "policies.deadlock_guard.max_no_progress_rounds", "deadlock_guard 阈值必须大于 0", "设置为正整数。", "POL-013")
	}
	if config.Policies.LoopGuard.MaxRoundsPerTask < 0 {
		appendError(errors, "CFG-POLICY-008", configerrors.CategoryPolicy, "policies.loop_guard.max_rounds_per_task", "loop_guard 阈值不能为负数", "设置为正整数。", "POL-005")
	}
	if config.Policies.LoopGuard.MaxRoundsPerTask == 0 && hasFieldConfigured(config.Policies.LoopGuard) {
		appendError(errors, "CFG-POLICY-009", configerrors.CategoryPolicy, "policies.loop_guard.max_rounds_per_task", "loop_guard 阈值必须大于 0", "设置为正整数。", "POL-005")
	}
}

func validateImmutableArtifacts(config model.RuntimeConfig, errors *[]configerrors.ConfigError) {
	if len(config.Policies.ImmutableArtifacts.Paths) == 0 {
		return
	}
	allowed := make(map[string]struct{})
	for _, roleID := range config.Policies.ImmutableArtifacts.AllowWriters {
		allowed[roleID] = struct{}{}
	}
	for roleIndex, role := range config.Roles {
		if _, exists := allowed[role.ID]; exists {
			continue
		}
		for patternIndex, pattern := range role.Constraints.ForbiddenWrites {
			if containsImmutablePattern(pattern, config.Policies.ImmutableArtifacts.Paths) {
				continue
			}
			_ = patternIndex
		}
		for _, immutablePath := range config.Policies.ImmutableArtifacts.Paths {
			if roleCanWriteImmutable(role, immutablePath) {
				appendError(errors, "CFG-SECURITY-002", configerrors.CategorySecurity, fmt.Sprintf("roles[%d].constraints.forbidden_writes", roleIndex), "角色可写路径与 immutable_artifacts 冲突", "在 forbidden_writes 中阻断只读路径，或把角色加入 allow_writers。", "IMM-005")
				break
			}
		}
	}
}

func roleCanWriteImmutable(role model.RoleConfig, immutablePath string) bool {
	if len(role.Constraints.ForbiddenWrites) == 0 {
		return true
	}
	for _, forbidden := range role.Constraints.ForbiddenWrites {
		if forbidden == immutablePath {
			return false
		}
	}
	return true
}

func containsImmutablePattern(pattern string, immutablePaths []string) bool {
	for _, immutablePath := range immutablePaths {
		if immutablePath == pattern {
			return true
		}
	}
	return false
}

func flattenTools(registry model.ToolRegistry) map[string]struct{} {
	result := make(map[string]struct{})
	for toolID := range registry.Builtin {
		result[toolID] = struct{}{}
	}
	for toolID := range registry.MCP {
		result[toolID] = struct{}{}
	}
	return result
}

func usesShellExec(roles []model.RoleConfig) bool {
	for _, role := range roles {
		for _, toolID := range role.Tools {
			if toolID == "shell_exec" {
				return true
			}
		}
	}
	return false
}

func hasFieldConfigured[T any](value T) bool {
	return fmt.Sprintf("%+v", value) != fmt.Sprintf("%+v", *new(T))
}

func appendError(errors *[]configerrors.ConfigError, code string, category configerrors.Category, path string, message string, suggestion string, ruleID string) {
	*errors = append(*errors, configerrors.ConfigError{
		Code:       code,
		Category:   category,
		Path:       path,
		Message:    message,
		Suggestion: suggestion,
		RuleID:     ruleID,
	})
}
