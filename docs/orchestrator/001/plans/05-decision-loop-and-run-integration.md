# Orchestrator 001 Plan 05: 决策循环与 run 接入

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把 bootstrap、snapshot、schedule、dispatch 能力收口为 `Tick()` 主循环，并在 `openoctopus run` 成功创建 session、bootstrap event-bus 后立即完成首轮 orchestrator 调度。

**Architecture:** 保持 `cmd/openoctopus/run.go` 只做参数解析、配置校验和调用服务；真正的阶段图推进、结论消费、重试/阻塞/完成判定、planner 文件更新和 bus 事件写入都收敛在 `internal/orchestrator`。`run` 只负责初始化和首轮 tick，不引入后台常驻循环。

**Tech Stack:** Go、Cobra、现有 `internal/eventbus`

---

### Task 1: 实现 `conclusion.md` 解析与阶段收口

**Files:**
- Create: `internal/orchestrator/conclusion.go`
- Create: `internal/orchestrator/engine.go`
- Test: `internal/orchestrator/engine_conclusion_test.go`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定四类结论推进**

- `SUCCESS` 会把 stage 推进到 `COMPLETED`。
- `NEEDS_RETRY` 会把 stage 推进到 `RETRY_PENDING` 并递增 attempt。
- `BLOCKED` 会把 session 状态推进到 `WAITING_HUMAN`。
- `FAILED` 会把 workflow 推进到 `FAILED`。

Run: `go test ./internal/orchestrator -run TestApplyConclusion -v`
Expected: 先失败，说明结论收口尚未建立。

**Step 2: 实现结论解析与推进**

- 从 `roles/{role_id}/conclusion.md` 读取最小契约字段。
- 不依赖 role-runtime 的临时上下文或内存态。
- 对非法 `status` 返回稳定错误，不静默跳过。

**Step 3: 回跑结论推进测试**

Run: `go test ./internal/orchestrator -run TestApplyConclusion -v`
Expected: 通过。

### Task 2: 实现 `Tick()` 主循环与全局投影更新

**Files:**
- Modify: `internal/orchestrator/engine.go`
- Test: `internal/orchestrator/engine_tick_test.go`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定完整 tick 顺序**

- `Tick()` 会先吸收新消息，再收口已有结论，再分发新的 ready stage。
- 有实质进展时 `global_progress.md` 和 `decision_log.md` 会更新。
- 无实质进展达到阈值时会写 `blockers.md`。

Run: `go test ./internal/orchestrator -run TestTick -v`
Expected: 先失败，说明主循环尚未串起来。

**Step 2: 实现主循环**

- 固定顺序执行：锁 -> 读配置 -> 吸收输入 -> 应用结论 -> 选择 ready -> 分发 -> 写投影 -> 提交 offset -> 释放锁。
- 维持单次 tick 确定性，不做后台 watch。
- `decision_log.md` 只追加关键决策，不复制完整 schedule。

**Step 3: 回跑 tick 测试**

Run: `go test ./internal/orchestrator -run TestTick -v`
Expected: 通过。

### Task 3: 接入 `run` 成功路径并保持旧约束不回退

**Files:**
- Modify: `cmd/openoctopus/run.go`
- Modify: `cmd/openoctopus/command_test.go`
- Reference: `docs/session/001/prd.md`
- Reference: `docs/event-bus/001/prd.md`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定 `run` 后 orchestrator 已启动**

- 执行最小合法 `run`。
- 断言 session 目录下 `planner/master_schedule.md` 含真实 stage 块。
- 断言 `roles/{role_id}/context.md` 与 `inbox.md` 已被创建。

Run: `go test ./cmd/openoctopus -run TestRunCommandBootstrapsOrchestrator -v`
Expected: 先失败，说明 `run` 仍只创建 session 和 bus。

**Step 2: 在 `run` 成功路径调用 orchestrator bootstrap + 首轮 tick**

- 保持顺序：先配置校验，再创建 session，再 bootstrap bus，再 orchestrator bootstrap + tick。
- orchestrator 失败时，明确返回错误，不输出伪成功信息。
- 不在命令层拼 planner 文件路径。

**Step 3: 回跑命令级测试**

Run: `go test ./cmd/openoctopus -run TestRunCommandBootstrapsOrchestrator -v`
Expected: 通过。
