# Orchestrator 001 Plan 04: 分发包、角色 Context 与 Inbox

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 ready stage 的任务分发能力，让 orchestrator 可以为目标角色生成 `context.md` 和 `inbox.md`，并通过 event-bus 记录分发事件与位点推进。

**Architecture:** 在持有有效 lease 的前提下，orchestrator 统一挑选 ready stage、生成 `task_id`、渲染角色上下文和 inbox，然后追加 dispatch 事件并提交 `orchestrator/master` offset。首版坚持“同一角色同一时刻一个 active task”。

**Tech Stack:** Go、标准库、现有 `internal/eventbus`

---

### Task 1: 实现角色目录物化与 `context.md` / `inbox.md` 渲染

**Files:**
- Create: `internal/orchestrator/dispatch.go`
- Test: `internal/orchestrator/dispatch_render_test.go`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定任务包形状**

- 首次分发时会创建 `roles/{role_id}/` 子目录。
- `context.md` 与 `inbox.md` 中的 `task_id`、`stage_id`、`role_id` 一致。
- 重试重发时 `context_version` 与 `inbox_version` 会递增。

Run: `go test ./internal/orchestrator -run TestRenderDispatchPackage -v`
Expected: 先失败，说明任务包渲染尚未建立。

**Step 2: 实现任务包渲染**

- 生成稳定 `task_id`，例如 `task-<stage_id>-<attempt>`。
- 角色目录不存在时自动创建。
- `context.md` 只写主控提供的上下文，不抢 role-runtime 的 `state.md` 职责。

**Step 3: 回跑任务包测试**

Run: `go test ./internal/orchestrator -run TestRenderDispatchPackage -v`
Expected: 通过。

### Task 2: 实现 ready stage 选择与并发限制

**Files:**
- Modify: `internal/orchestrator/dispatch.go`
- Test: `internal/orchestrator/dispatch_select_test.go`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定分发节流**

- 多个 entry stage 同时 ready 时，最多只分发 `max_parallel_roles` 个。
- 同一角色有两个 ready stage 时，只分发一个。
- 已处于 `DISPATCHED` 的 stage 不会被重复分发。

Run: `go test ./internal/orchestrator -run TestSelectReadyStages -v`
Expected: 先失败，说明 ready stage 选择逻辑尚未完成。

**Step 2: 实现分发选择逻辑**

- 先过滤已 active 的 role。
- 再按 schedule 顺序选择 ready stage。
- 不做复杂优先级系统，保持首版确定性与可预测性。

**Step 3: 回跑分发选择测试**

Run: `go test ./internal/orchestrator -run TestSelectReadyStages -v`
Expected: 通过。

### Task 3: 接入 bus 事件、dispatch log 与 consumer offset

**Files:**
- Modify: `internal/orchestrator/dispatch.go`
- Test: `internal/orchestrator/dispatch_events_test.go`
- Reference: `docs/event-bus/001/prd.md`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定分发副作用**

- 成功分发后 bus 中会追加 `TASK_DISPATCHED` 事件。
- `planner/dispatch_log.md` 会追加一条新记录。
- `offsets.md` 中 `consumer_id=orchestrator/master` 会推进到本轮最新事件。

Run: `go test ./internal/orchestrator -run TestDispatchSideEffects -v`
Expected: 先失败，说明分发副作用尚未完成。

**Step 2: 实现分发副作用**

- 通过 `event-bus` 持锁追加事件。
- 只在任务包与 schedule 更新成功后写 dispatch log。
- 提交 offset 时使用总线尾事件，而不是自造游标。

**Step 3: 回跑分发副作用测试**

Run: `go test ./internal/orchestrator -run TestDispatchSideEffects -v`
Expected: 通过。
