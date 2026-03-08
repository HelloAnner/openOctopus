# Config 001 Plan 04: 校验器与错误模型

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `config 001` 的统一错误模型、五层静态校验器以及与 `yaml-rules.md` 的 `rule_id` 回链能力。

**Architecture:** 采用“字段基础约束 + 分层业务校验器”的设计：模型层只做局部标签约束，复杂规则统一进入 `validator` 层。错误统一归一为稳定结构，CLI 和 E2E 只消费错误结构，不关心内部校验函数如何组织。

**Tech Stack:** Go、`go-playground/validator`

---

### Task 1: 定义统一错误结构

**Files:**
- Create: `internal/config/errors/config_error.go`
- Create: `internal/config/errors/error_codes.go`
- Test: `internal/config/errors/config_error_test.go`
- Reference: `docs/config/001/prd.md:213`
- Reference: `docs/config/001/yaml-rules.md:11`

**Step 1: 写失败测试，锁定错误字段**

- 断言错误结构包含 `code`、`category`、`path`、`message`、`suggestion`、`rule_id`。
- 断言错误数组可稳定排序和格式化。

Run: `go test ./internal/config/errors -run TestConfigErrorShape -v`
Expected: 先失败，说明错误结构未定义。

**Step 2: 实现错误结构与类别常量**

- 错误分类固定为 `syntax`、`schema`、`reference`、`security`、`policy`。
- 错误码前缀统一使用 `CFG-*`。

**Step 3: 回跑错误模型测试**

Run: `go test ./internal/config/errors -run TestConfigErrorShape -v`
Expected: 通过。

### Task 2: 实现分层校验器

**Files:**
- Create: `internal/config/validator/validator.go`
- Create: `internal/config/validator/schema_validator.go`
- Create: `internal/config/validator/reference_validator.go`
- Create: `internal/config/validator/security_validator.go`
- Create: `internal/config/validator/policy_validator.go`
- Test: `internal/config/validator/validator_test.go`

**Step 1: 写失败测试，覆盖核心 6 类错误**

- 语法错误。
- 缺失角色引用。
- 工具未注册。
- `shell_exec` 缺少 `security.shell`。
- `immutable_artifacts` 冲突。
- 监控阈值非法。

Run: `go test ./internal/config/validator -run TestValidateConfig -v`
Expected: 先失败，提示复杂校验尚未实现。

**Step 2: 实现五层校验链**

- 先执行语法/反序列化错误归类。
- 再执行 schema、reference、security、policy 校验。
- 允许一次返回多条错误，避免单条逐个修复。

**Step 3: 回跑核心校验测试**

Run: `go test ./internal/config/validator -run TestValidateConfig -v`
Expected: 非法样例返回稳定错误列表，合法样例通过。

### Task 3: 建立 `rule_id` 映射

**Files:**
- Create: `internal/config/validator/rule_mapping.go`
- Modify: `internal/config/validator/*.go`
- Test: `internal/config/validator/rule_mapping_test.go`
- Reference: `docs/config/001/yaml-rules.md`

**Step 1: 写失败测试，锁定典型错误到规则编号的映射**

- `stage.role` 不存在 -> `STAGE-004`
- `shell_exec` 缺少安全配置 -> `SEC-001`
- 只读冲突 -> `IMM-*`

Run: `go test ./internal/config/validator -run TestRuleMapping -v`
Expected: 先失败，说明错误未携带 `rule_id`。

**Step 2: 实现规则映射表**

- 映射表必须直接引用文档中的稳定编号。
- 后续新增校验时，先补文档，再补映射。

**Step 3: 回跑 validator 全量测试**

Run: `go test ./internal/config/validator -v`
Expected: 错误结构和 `rule_id` 全部稳定。
