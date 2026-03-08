# Recovery 001 Plan 03: 恢复视图归约与 Replay 报告

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立恢复视图归约逻辑，能够基于 schedule / interrupts / conclusions 修复 `session.state.md`、`planner/blockers.md`，并写出 `audit/replay.md`。

**Architecture:** 这一步不直接继续执行，而是先把“恢复后系统应当是什么样”算清楚。通过一个 reducer 统一计算 workflow status、当前 stage/role、blocker summary，再把结果写回当前态文件和 replay 报告。

**Tech Stack:** Go、标准库

---

### Task 1: 锁定恢复视图归约规则

**Files:**
- Create: `internal/recovery/reducer.go`
- Test: `internal/recovery/reducer_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，覆盖四类状态归约**

- 全部完成 -> `COMPLETED`
- 存在 `DISPATCHED` -> `RUNNING`
- 存在 `BLOCKED` -> `WAITING_HUMAN`
- 存在 `FAILED` -> `FAILED`

Run: `go test ./internal/recovery -run TestReduceWorkflowStatus -v`
Expected: 先失败，说明归约规则尚未落地。

**Step 2: 实现状态归约器**

- 读取 `master_schedule.md` 与必要的 interrupt / conclusion 摘要。
- 计算 workflow status、current stage/role、blocker summary。

**Step 3: 回跑归约测试**

Run: `go test ./internal/recovery -run TestReduceWorkflowStatus -v`
Expected: 通过。

### Task 2: 实现 `session.state.md` / `blockers.md` 修复

**Files:**
- Create: `internal/recovery/repair.go`
- Test: `internal/recovery/reducer_test.go`
- Reference: `docs/recovery/001/e2e.md`

**Step 1: 写失败测试，锁定修复结果**

- 缺失 `session.state.md` 时可以自动重建。
- 非等待人工时 `blockers.md` 被修回 `clear`。

Run: `go test ./internal/recovery -run TestRepairSessionFiles -v`
Expected: 先失败，说明修复逻辑尚未实现。

**Step 2: 实现最小修复逻辑**

- 只重写当前态文件，不碰 `master_schedule.md` 结构。
- 保持 UTC 时间与现有 Markdown 风格一致。

**Step 3: 回跑修复测试**

Run: `go test ./internal/recovery -run TestRepairSessionFiles -v`
Expected: 通过。

### Task 3: 实现 `audit/replay.md` 报告

**Files:**
- Modify: `internal/recovery/repair.go`
- Test: `internal/recovery/reducer_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，锁定 replay 报告内容**

- 报告包含 session id、读取 checkpoint、repaired files、continued 标记。

Run: `go test ./internal/recovery -run TestWriteReplayReport -v`
Expected: 先失败，说明 replay 报告仍未实现。

**Step 2: 实现最小 replay 渲染**

- 用覆盖写生成最新 replay 结果。
- 避免写成长历史日志，首版只保留最新一次恢复摘要。

**Step 3: 回跑 replay 测试**

Run: `go test ./internal/recovery -run TestWriteReplayReport -v`
Expected: 通过。

