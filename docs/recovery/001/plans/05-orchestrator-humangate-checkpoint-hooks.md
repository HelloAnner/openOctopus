# Recovery 001 Plan 05: orchestrator / human-gate Checkpoint 接入

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把 checkpoint writer 接到真正的状态边界上，确保 recovery 不是只有手动 `recover` 才会产生新 checkpoint。

**Architecture:** 选择最小但关键的边界：stage 分发、stage 结论应用、`interrupt-all`、`resume`。不把所有文件写入都变成 checkpoint，只在真正影响恢复语义的边界留痕。

**Tech Stack:** Go、标准库

---

### Task 1: orchestrator 分发与结论应用时写 checkpoint

**Files:**
- Modify: `internal/orchestrator/engine.go`
- Test: `internal/orchestrator/engine_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，锁定 stage 边界 checkpoint**

- 首次 dispatch 后新增 `stage-...-dispatched` checkpoint。
- stage 结论被消费后新增 `completed` / `blocked` / `failed` checkpoint。

Run: `go test ./internal/orchestrator -run TestCheckpoint -v`
Expected: 先失败，说明 orchestrator 还未接入 checkpoint。

**Step 2: 实现最小 checkpoint hook**

- 只在状态真正变化时写 checkpoint。
- 不改变 orchestrator 现有主逻辑顺序。

**Step 3: 回跑测试**

Run: `go test ./internal/orchestrator -run TestCheckpoint -v`
Expected: 通过。

### Task 2: human-gate 等待人工与恢复时写 checkpoint

**Files:**
- Modify: `internal/humangate/service.go`
- Test: `internal/humangate/service_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，锁定两类 checkpoint**

- `interrupt-all` 后新增 `session-waiting-human` checkpoint。
- `resume` 后新增 `session-resumed` checkpoint。

Run: `go test ./internal/humangate -run TestCheckpoint -v`
Expected: 先失败，说明 human-gate 还未接入 recovery checkpoint。

**Step 2: 实现最小 hook**

- 保持 human-gate 服务职责不变，只增加边界留痕。

**Step 3: 回跑测试**

Run: `go test ./internal/humangate -run TestCheckpoint -v`
Expected: 通过。

