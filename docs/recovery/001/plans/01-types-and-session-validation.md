# Recovery 001 Plan 01: 类型基线与会话校验

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 recovery 基础类型、错误模型、关键文件校验与 `events.md` 哈希链校验能力，让恢复流程先具备可信入口。

**Architecture:** 先定义 `internal/recovery` 的对外输入输出结构，再实现 session 验证与事件链校验。该阶段只回答“这个 session 能不能恢复”，不负责真正续跑。

**Tech Stack:** Go、标准库

---

### Task 1: 定义 recovery 基础类型与错误模型

**Files:**
- Create: `internal/recovery/types.go`
- Create: `internal/recovery/errors.go`
- Test: `internal/recovery/service_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，锁定基础输入输出结构**

- 为 `RecoverOptions`、`RecoverResult`、`ReplayReport`、稳定错误类型写最小测试。

Run: `go test ./internal/recovery -run TestRecoveryTypes -v`
Expected: 先失败，提示类型或错误尚未定义。

**Step 2: 实现基础类型与错误**

- 定义 recovery 服务需要的输入输出结构。
- 定义稳定错误变量，避免后续靠字符串判断恢复失败原因。

**Step 3: 回跑类型测试**

Run: `go test ./internal/recovery -run TestRecoveryTypes -v`
Expected: 通过。

### Task 2: 实现 session 关键文件校验

**Files:**
- Create: `internal/recovery/validate.go`
- Test: `internal/recovery/service_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，锁定关键文件校验行为**

- 缺少 `state/effective_config.yaml` 时返回稳定错误。
- 缺少 `bus/events.md` 时返回稳定错误。
- 缺少 `session.state.md` 不直接失败，而是进入“待修复”列表。

Run: `go test ./internal/recovery -run TestValidateSessionLayout -v`
Expected: 先失败，说明校验规则未实现。

**Step 2: 实现最小 session 校验**

- 检查 metadata / config / bus / planner 关键文件。
- 区分“必须存在”和“允许恢复时修复”的文件。

**Step 3: 回跑校验测试**

Run: `go test ./internal/recovery -run TestValidateSessionLayout -v`
Expected: 通过。

### Task 3: 实现 `events.md` 哈希链校验

**Files:**
- Modify: `internal/recovery/validate.go`
- Test: `internal/recovery/service_test.go`
- Reference: `docs/event-bus/001/prd.md`
- Reference: `docs/recovery/001/e2e.md`

**Step 1: 写失败测试，锁定两类关键行为**

- 合法 `events.md` 通过校验。
- 篡改某条事件后哈希链校验失败。

Run: `go test ./internal/recovery -run TestValidateEventChain -v`
Expected: 先失败，说明 recovery 尚未接入事件校验。

**Step 2: 实现最小哈希链校验**

- 复用 `event-bus` 现有解析逻辑读取事件。
- 顺序校验 `prev_event_hash` 与 `event_hash`。

**Step 3: 回跑事件校验测试**

Run: `go test ./internal/recovery -run TestValidateEventChain -v`
Expected: 通过。

