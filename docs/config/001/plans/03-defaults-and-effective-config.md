# Config 001 Plan 03: 默认值注入与有效配置

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 在加载后的原始配置上注入首版保守默认值，并输出 `AppliedDefaults` 记录，形成可被 CLI 和后续模块消费的有效配置对象。

**Architecture:** 默认值逻辑单独放到 `defaults` 层，严格按 PRD 与 `yaml-rules.md` 列出的字段补全，避免散落在模型初始化、CLI 或校验器内部。默认值注入之后立即产出一份“有效配置快照”，但不在这里做复杂引用校验。

**Tech Stack:** Go、配置模型包

---

### Task 1: 定义 `AppliedDefault` 结果模型

**Files:**
- Create: `internal/config/defaults/applied_default.go`
- Test: `internal/config/defaults/applied_default_test.go`
- Reference: `docs/config/001/prd.md:130`

**Step 1: 写失败测试，固定记录格式**

- 测试 `AppliedDefault` 至少包含 `path`、`value`、`reason`。
- 测试多个默认值可以稳定排序输出。

Run: `go test ./internal/config/defaults -run TestAppliedDefaultShape -v`
Expected: 先失败，提示结果模型不存在。

**Step 2: 实现结果模型**

- 保证后续 CLI 可直接序列化展示。
- 路径格式与错误输出路径保持一致。

**Step 3: 回跑结果模型测试**

Run: `go test ./internal/config/defaults -run TestAppliedDefaultShape -v`
Expected: 通过。

### Task 2: 实现保守默认值注入

**Files:**
- Create: `internal/config/defaults/defaults.go`
- Test: `internal/config/defaults/defaults_test.go`
- Reference: `docs/config/001/prd.md:130`
- Reference: `docs/config/001/yaml-rules.md:74`

**Step 1: 写失败测试，覆盖三类默认值**

- 工作目录类默认值。
- 运行安全类默认值。
- 可观测性类默认值。

Run: `go test ./internal/config/defaults -run TestApplyDefaults -v`
Expected: 先失败，提示默认值未注入。

**Step 2: 实现默认值注入器**

- 仅对 PRD 列出的字段补默认值。
- 已显式填写的字段不覆盖。
- 每次补默认值都追加一条 `AppliedDefault` 记录。

**Step 3: 回跑默认值测试**

Run: `go test ./internal/config/defaults -run TestApplyDefaults -v`
Expected: 缺省字段被补齐，显式字段保持不变。

### Task 3: 组装有效配置对象

**Files:**
- Create: `internal/config/effective/effective_config.go`
- Modify: `internal/config/loader/loader.go`
- Test: `internal/config/effective/effective_config_test.go`

**Step 1: 写失败测试，锁定“加载后 + 默认值”流程**

- 使用最小合法 YAML，断言加载后可以拿到完整有效配置。
- 断言 `AppliedDefaults` 非空。

Run: `go test ./internal/config/effective -run TestBuildEffectiveConfig -v`
Expected: 先失败，提示组装流程未定义。

**Step 2: 实现有效配置组装**

- 让 loader 输出的原始模型进入 defaults 层。
- 返回 `RuntimeConfig + AppliedDefaults`。

**Step 3: 回跑 defaults 与 effective 全量测试**

Run: `go test ./internal/config/defaults ./internal/config/effective -v`
Expected: 全部通过。
