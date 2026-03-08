# Orchestrator 001 Plan 02: 阶段图与排程投影

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `effective_config.yaml` 到 `master_schedule.md`、`task_board.md`、`task_graph.mmd` 的稳定投影能力，让 orchestrator 具备可解释的首版阶段图与当前排程。

**Architecture:** 先实现对 `stages` / `transitions` 的图构建和首版约束校验，再把图渲染成 schedule、看板与 Mermaid 图。首版只支持 `transition.to` 的直接流转与多入口起步，不做条件表达式或多前驱 join。

**Tech Stack:** Go、标准库

---

### Task 1: 实现阶段图构建与首版约束校验

**Files:**
- Create: `internal/orchestrator/graph.go`
- Test: `internal/orchestrator/graph_test.go`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定图构建行为**

- 单 stage workflow 可被识别为一个 entry stage。
- 两个互不依赖的 stage 会被识别为两个 entry stage。
- 使用 `on_true` / `on_false` / 多前驱 join 的配置会被拒绝并返回稳定错误。

Run: `go test ./internal/orchestrator -run TestBuildGraph -v`
Expected: 先失败，说明图构建和约束校验尚未建立。

**Step 2: 实现图构建与约束校验**

- 从 `effective_config.yaml` 读取 stages / transitions。
- 计算 entry stage 集合和 `next_stage_id`。
- 对 `001` 不支持的形状显式报错，不做静默忽略。

**Step 3: 回跑图构建测试**

Run: `go test ./internal/orchestrator -run TestBuildGraph -v`
Expected: 通过。

### Task 2: 实现 `master_schedule.md` 渲染与状态初始化

**Files:**
- Create: `internal/orchestrator/schedule.go`
- Test: `internal/orchestrator/schedule_test.go`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定 schedule 初始形状**

- entry stage 初始状态为 `READY`。
- 非 entry stage 初始状态为 `PENDING`。
- 文档中每个 stage 都包含 `stage_id`、`role_id`、`status`、`attempt`、`next_stage_id`。

Run: `go test ./internal/orchestrator -run TestRenderMasterSchedule -v`
Expected: 先失败，说明 schedule 投影尚未建立。

**Step 2: 实现 schedule 投影**

- 生成 `schedule_version: 1` 的首版文档。
- 固定 stage 输出顺序，保证 diff 稳定。
- 只把 `master_schedule.md` 作为事实源，其他投影从它生成。

**Step 3: 回跑 schedule 测试**

Run: `go test ./internal/orchestrator -run TestRenderMasterSchedule -v`
Expected: 通过。

### Task 3: 实现 `task_board.md`、`task_graph.mmd` 与 `global_progress.md` 投影

**Files:**
- Modify: `internal/orchestrator/render.go`
- Test: `internal/orchestrator/projections_test.go`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定三类投影**

- `task_board.md` 会按 `Todo / Doing / Done / Blocked` 分组。
- `task_graph.mmd` 会包含全部 stage 与 `to` 边。
- `global_progress.md` 会输出 workflow 当前状态、完成数和 active dispatch 数。

Run: `go test ./internal/orchestrator -run TestRenderProjections -v`
Expected: 先失败，说明三类投影尚未完成。

**Step 2: 实现投影渲染**

- 保持投影只读，不新增事实字段。
- Mermaid 图只渲染 `001` 支持的直接边。
- `global_progress.md` 聚焦可读摘要，不复制完整 schedule。

**Step 3: 回跑投影测试**

Run: `go test ./internal/orchestrator -run TestRenderProjections -v`
Expected: 通过。
