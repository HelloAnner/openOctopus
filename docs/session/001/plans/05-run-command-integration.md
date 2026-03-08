# Session 001 Plan 05: Run 命令接入

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `session 001` 的创建流程正式接入 `openoctopus run`，确保合法配置能创建完整 session，非法配置仍在 session 创建前被阻断。

**Architecture:** `run` 命令继续复用现有 config 校验入口；CLI 只负责参数解析、调用 session 服务和打印结果。session 创建的真实路径、模板文件和回滚逻辑全部留在 `internal/session`，避免命令层再次膨胀。

**Tech Stack:** Go、Cobra

---

### Task 1: 写命令级成功用例，锁定 `run` 正常创建行为

**Files:**
- Modify: `cmd/openoctopus/command_test.go`
- Reference: `docs/session/001/e2e.md`

**Step 1: 写失败测试，锁定合法 `run` 输出与骨架结果**

- 使用最小合法 YAML 执行 `run`。
- 断言 stdout 包含 `session created:`。
- 断言解析出的 session 目录下存在 `metadata.md`、`session.state.md`、`timeline.md`、`state/effective_config.yaml`。

Run: `go test ./cmd/openoctopus -run TestRunCommandCreatesSessionSkeleton -v`
Expected: 先失败，说明 CLI 仍只有旧的最小目录创建行为。

**Step 2: 扩展成功用例断言**

- 不只断言目录存在，还要断言关键文件存在。
- 保持测试只做命令级黑盒，不直接调用私有 helper。

**Step 3: 回跑命令级成功测试**

Run: `go test ./cmd/openoctopus -run TestRunCommandCreatesSessionSkeleton -v`
Expected: 通过。

### Task 2: 保持非法配置前置阻断不回退

**Files:**
- Modify: `cmd/openoctopus/command_test.go`
- Modify: `cmd/openoctopus/run.go`

**Step 1: 回放现有失败测试，锁定旧约束继续生效**

- 非法 YAML 仍必须在创建 session 之前失败。

Run: `go test ./cmd/openoctopus -run TestRunCommandDoesNotCreateSessionForInvalidConfig -v`
Expected: 先红或回归风险可见。

**Step 2: 接入新的 session 创建入口**

- `run` 继续先调 `LoadForValidate`。
- 校验通过后再调用新的 `session.Create...` 入口。
- 将 `AppliedDefaults` 传入 session 模块，供元数据写入使用。

**Step 3: 回跑 run 命令相关测试**

Run: `go test ./cmd/openoctopus -run TestRunCommand -v`
Expected: 合法创建、非法阻断都通过。

### Task 3: 稳定 CLI 输出与结果展示

**Files:**
- Modify: `cmd/openoctopus/run.go`
- Modify: `cmd/openoctopus/command_test.go`

**Step 1: 写失败测试，锁定 stdout 结果格式**

- 输出至少包含 session 绝对或可定位路径。
- 失败时不打印伪成功信息。

Run: `go test ./cmd/openoctopus -run TestRunCommandOutput -v`
Expected: 先失败，说明输出格式还未固化。

**Step 2: 实现最小结果展示**

- 成功时打印 `session created: {sessionDir}`。
- 保持英文日志风格，与仓库已有输出一致。

**Step 3: 回跑 CLI 全量测试**

Run: `go test ./cmd/openoctopus -v`
Expected: 命令级测试通过。
