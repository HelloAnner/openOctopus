# Config 001 Plan 01: 强类型配置模型

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `config 001` 所需的 Go 强类型配置模型、基础常量和首版结构约束，作为后续加载、默认值和校验的唯一数据基线。

**Architecture:** 先定义清晰的 `RuntimeConfig` 顶层结构，再把 `runtime`、`policies`、`roles`、`stages`、`transitions` 拆到多个小文件，避免一个超大配置文件承载全部逻辑。模型层只负责结构与轻量标签，不掺杂加载、默认值注入和跨字段复杂校验。

**Tech Stack:** Go、YAML 反序列化库、`go-playground/validator`

---

### Task 1: 定义常量与顶层模型

**Files:**
- Create: `internal/config/model/constants.go`
- Create: `internal/config/model/runtime_config.go`
- Create: `internal/config/model/meta.go`
- Test: `internal/config/model/runtime_config_test.go`
- Reference: `docs/config/001/prd.md`
- Reference: `docs/config/001/yaml-rules.md`

**Step 1: 写失败测试，锁定最小合法结构**

- 为最小 YAML 骨架写测试，确认可以反序列化到 `RuntimeConfig`。
- 为缺失 `version`、`meta.workflow_id`、`roles`、`stages` 的场景写失败测试。

Run: `go test ./internal/config/model -run TestRuntimeConfig_MinimalShape -v`
Expected: 先失败，提示类型或结构未定义。

**Step 2: 实现顶层常量与结构体**

- 定义首版常量，例如支持的配置版本和 `__END__`。
- 定义 `RuntimeConfig`、`MetaConfig` 等顶层结构体。
- 对未知字段采用严格模式，避免静默吞错。

**Step 3: 回跑模型测试**

Run: `go test ./internal/config/model -run TestRuntimeConfig_MinimalShape -v`
Expected: 通过最小结构测试，缺失字段测试得到稳定失败信息。

### Task 2: 拆分 runtime / policies / workflow 模型

**Files:**
- Create: `internal/config/model/runtime.go`
- Create: `internal/config/model/security.go`
- Create: `internal/config/model/policies.go`
- Create: `internal/config/model/role.go`
- Create: `internal/config/model/stage.go`
- Create: `internal/config/model/transition.go`
- Test: `internal/config/model/sub_models_test.go`

**Step 1: 写失败测试，覆盖复杂子结构**

- 反序列化 `runtime.role_runtime`、`policies.deadlock_guard`、`roles[*].constraints`、`transitions[*].condition`。
- 确认数组与对象字段的 YAML 标签正确。

Run: `go test ./internal/config/model -run TestSubModels_Decode -v`
Expected: 先失败，提示子结构缺失或映射错误。

**Step 2: 实现子模型**

- 将 runtime、安全、策略、角色、阶段、流转分文件定义。
- 保持每个文件只负责一个子域。

**Step 3: 回跑完整模型测试**

Run: `go test ./internal/config/model -v`
Expected: 模型测试全部通过。

### Task 3: 补轻量 validator 标签

**Files:**
- Modify: `internal/config/model/runtime_config.go`
- Modify: `internal/config/model/meta.go`
- Modify: `internal/config/model/role.go`
- Modify: `internal/config/model/stage.go`
- Modify: `internal/config/model/transition.go`
- Test: `internal/config/model/validator_tags_test.go`

**Step 1: 写失败测试，锁定基础标签约束**

- 测试 `workflow_id`、`roles[*].id`、`stages[*].id`、`llm_profiles` 空对象等基础约束。

Run: `go test ./internal/config/model -run TestValidatorTags -v`
Expected: 先失败，提示标签未生效或未接入验证器。

**Step 2: 增加只靠字段自身即可表达的基础标签**

- 只写必填、非空、正整数、基础枚举等局部约束。
- 不在模型层做跨字段引用校验。

**Step 3: 回跑模型层全部测试**

Run: `go test ./internal/config/model -v`
Expected: 全部通过。
