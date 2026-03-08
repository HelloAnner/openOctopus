# CLI 001 Plan 02: Session 解析与状态读取服务

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 CLI 可复用的 session 引用解析和状态聚合读模型，为 `status` 与 human-gate 相关命令提供统一底座。

**Architecture:** 新增 `internal/cli` 只读支撑层，负责解析 session id / dir、读取 `metadata.md`、`session.state.md`、`master_schedule.md`、`blockers.md`，并汇总为一个稳定的 `StatusSummary`。

**Tech Stack:** Go、标准库

---

### Task 1: 写状态读取失败测试

**Files:**
- Create: `internal/cli/types.go`
- Create: `internal/cli/service_test.go`
- Reference: `docs/cli/001/prd.md`
- Reference: `docs/session/001/prd.md`

**Step 1: Write the failing tests**
- 覆盖 session id 解析为真实目录。
- 覆盖读取 `RUNNING` / `WAITING_HUMAN` 的聚合结果。
- 覆盖 blocker 占位文件时的回退行为。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/cli -run 'TestResolve|TestReadStatus' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 实现路径解析、Markdown leading values 解析与状态聚合。
- 为 `session not found` 定义稳定错误。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/cli -run 'TestResolve|TestReadStatus' -v`
- Expected: PASS。

