# Recovery 001 Plan 04: Recover 服务与正式 CLI

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 组装 recovery 服务主入口，并把它接入 `openoctopus recover` 正式命令，让用户可以用稳定 CLI 做恢复与续跑。

**Architecture:** 先在 `internal/recovery` 里把校验、checkpoint、归约、修复整合成 `Service.Recover(...)`，再由 Cobra 命令调用。`recover` 本身不重新实现 orchestrator / role-runtime，而是修好环境后复用现有 loop。

**Tech Stack:** Go、Cobra、标准库

---

### Task 1: 实现 recovery 服务主流程

**Files:**
- Create: `internal/recovery/service.go`
- Test: `internal/recovery/service_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，锁定 recover 主链路**

- session 可恢复时返回 `continued=true`。
- `WAITING_HUMAN` 时返回 `continued=false`，但不是错误。
- 事件链损坏时返回稳定错误。

Run: `go test ./internal/recovery -run TestRecover -v`
Expected: 先失败，说明主流程尚未建立。

**Step 2: 实现最小 recover 服务**

- 按 `validate -> checkpoint -> reduce -> repair -> maybe continue` 顺序执行。
- 复用 orchestrator / role-runtime 继续推进。

**Step 3: 回跑服务测试**

Run: `go test ./internal/recovery -run TestRecover -v`
Expected: 通过。

### Task 2: 接入正式 CLI `recover`

**Files:**
- Create: `cmd/openoctopus/recover.go`
- Modify: `cmd/openoctopus/root.go`
- Modify: `cmd/openoctopus/errors.go`
- Test: `cmd/openoctopus/command_test.go`
- Reference: `docs/recovery/001/e2e.md`

**Step 1: 写失败测试，锁定命令输出协议**

- 文本模式返回 session 路径与恢复后状态。
- JSON 模式返回 `continued`、`recovered_status`、`checkpoint_ref`。

Run: `go test ./cmd/openoctopus -run TestRecoverCommand -v`
Expected: 先失败，说明命令尚未注册或输出未定义。

**Step 2: 实现 `recover` 命令**

- 复用 CLI session 解析逻辑。
- 将 recovery 服务结果映射到统一输出协议。

**Step 3: 回跑命令测试**

Run: `go test ./cmd/openoctopus -run TestRecoverCommand -v`
Expected: 通过。

