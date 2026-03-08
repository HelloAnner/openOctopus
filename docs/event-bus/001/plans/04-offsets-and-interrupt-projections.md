# Event Bus 001 Plan 04: Offsets 与 Interrupts 投影

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `offsets.md` 和 `interrupts.md` 的原子投影更新能力，让未来的 orchestrator、role-runtime、human-gate 都能基于总线读到稳定当前态，而不是每次都去扫全量事件日志。

**Architecture:** 在持有有效 lease 的前提下，offset 提交与 interrupt 状态推进统一采用“先写事件，再重写投影文件”的模式。投影文件只表达当前态，不承担完整历史；完整历史仍以 `events.md` 为准。

**Tech Stack:** Go、标准库

---

### Task 1: 实现 offset 提交与回退阻断

**Files:**
- Create: `internal/eventbus/offsets.go`
- Test: `internal/eventbus/offsets_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定 offset 提交语义**

- 首次提交 `consumer_id=orchestrator/master` 成功。
- 同一消费者提交更大的 `last_sequence` 成功。
- 提交更小的 `last_sequence` 失败并返回 `offset regression`。

Run: `go test ./internal/eventbus -run TestCommitOffset -v`
Expected: 先失败，说明 offsets 投影尚未实现。

**Step 2: 实现 `CommitOffset()` 与 `ReadOffsets()`**

- 要求调用方传入有效 lease。
- 先追加 `OFFSET_COMMITTED` 事件，再原子重写 `offsets.md`。
- 保持消费者块按 `consumer_id` 排序。

**Step 3: 回跑 offset 测试**

Run: `go test ./internal/eventbus -run TestCommitOffset -v`
Expected: 通过。

### Task 2: 实现 interrupt 请求与状态推进

**Files:**
- Create: `internal/eventbus/interrupts.go`
- Test: `internal/eventbus/interrupts_test.go`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 写失败测试，锁定三步状态机**

- `RequestInterrupt()` 成功后，投影状态为 `REQUESTED`。
- `AcknowledgeInterrupt()` 成功后，状态为 `ACKNOWLEDGED`。
- `ClearInterrupt()` 成功后，状态为 `CLEARED`。

Run: `go test ./internal/eventbus -run TestInterruptLifecycle -v`
Expected: 先失败，说明 interrupt 投影尚未完成。

**Step 2: 实现 interrupt 三类操作**

- 先追加 `INTERRUPT_REQUESTED` / `INTERRUPT_ACKNOWLEDGED` / `INTERRUPT_CLEARED` 事件。
- 再原子重写 `interrupts.md`。
- `interrupt_id` 直接复用请求事件的 `event_id`。

**Step 3: 回跑 interrupt 测试**

Run: `go test ./internal/eventbus -run TestInterruptLifecycle -v`
Expected: 通过。

### Task 3: 校验非法状态迁移与丢失 interrupt

**Files:**
- Modify: `internal/eventbus/interrupts.go`
- Test: `internal/eventbus/interrupts_invalid_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定非法迁移**

- 对不存在的 `interrupt_id` 做 ack / clear，必须失败。
- 对已经 `CLEARED` 的 interrupt 再次 ack，必须失败。
- 对没有 lease 的调用，必须失败。

Run: `go test ./internal/eventbus -run TestInterruptInvalidTransitions -v`
Expected: 先失败，说明非法状态分支尚未补齐。

**Step 2: 实现非法迁移保护**

- 明确 `REQUESTED -> ACKNOWLEDGED -> CLEARED` 的单向状态机。
- 对 `interrupt not found`、`lease expired`、`lease conflict` 返回稳定错误。
- 不允许静默覆盖已有状态。

**Step 3: 回跑 offsets / interrupts 全量测试**

Run: `go test ./internal/eventbus -run 'Test(CommitOffset|Interrupt)' -v`
Expected: 全部通过。
