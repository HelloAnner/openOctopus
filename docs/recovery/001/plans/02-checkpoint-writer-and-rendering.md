# Recovery 001 Plan 02: Checkpoint 渲染与写入

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立增量 checkpoint 的序号分配、渲染模板与原子写入能力，让 recovery 不再只有 `0000-init.md` 一个初始化快照。

**Architecture:** 在 `internal/recovery` 内实现轻量 checkpoint writer，统一负责读取现有 checkpoint 序号、生成下一个文件名并渲染标准内容。该 writer 后续会被 orchestrator、human-gate 与 recover 服务共同复用。

**Tech Stack:** Go、标准库

---

### Task 1: 锁定 checkpoint 文件命名与内容模板

**Files:**
- Create: `internal/recovery/checkpoint.go`
- Test: `internal/recovery/checkpoint_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: 写失败测试，锁定模板形状**

- 断言文件名按 `0001-kind.md` 递增。
- 断言内容至少包含 `checkpoint_seq`、`kind`、`workflow_status`、`current_stage_id`、`last_event`。

Run: `go test ./internal/recovery -run TestRenderCheckpoint -v`
Expected: 先失败，说明 checkpoint 模板尚未建立。

**Step 2: 实现模板与文件名生成**

- 扫描 `state/checkpoints/` 现有文件序号。
- 统一渲染标准 Markdown 模板。

**Step 3: 回跑模板测试**

Run: `go test ./internal/recovery -run TestRenderCheckpoint -v`
Expected: 通过。

### Task 2: 实现 checkpoint 原子写入与返回结果

**Files:**
- Modify: `internal/recovery/checkpoint.go`
- Modify: `internal/recovery/types.go`
- Test: `internal/recovery/checkpoint_test.go`

**Step 1: 写失败测试，锁定写入行为**

- 断言第一次增量写入会生成 `0001-...`。
- 再次写入会生成下一个序号，而不是覆盖旧文件。

Run: `go test ./internal/recovery -run TestWriteCheckpoint -v`
Expected: 先失败，说明原子写入或序号分配未落地。

**Step 2: 实现最小 writer**

- 用 `*.tmp + rename` 写入 checkpoint。
- 返回相对路径，便于 CLI / replay 报告复用。

**Step 3: 回跑写入测试**

Run: `go test ./internal/recovery -run TestWriteCheckpoint -v`
Expected: 通过。

