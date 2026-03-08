# CLI 001 Plan 04: Human Gate 命令统一输出协议

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 `interrupt`、`interrupt-all`、`inject`、`resume` 接入统一的 `text/json` 输出，避免这些正式命令继续停留在仅文本模式。

**Architecture:** 继续保持 business logic 在 `internal/humangate`，只在 `cmd/openoctopus` 改输出层与 session 解析复用方式。

**Tech Stack:** Go、Cobra

---

### Task 1: 为 human-gate 命令写失败测试

**Files:**
- Modify: `cmd/openoctopus/human_gate_command_test.go`
- Modify: `cmd/openoctopus/interrupt.go`
- Modify: `cmd/openoctopus/interrupt_all.go`
- Modify: `cmd/openoctopus/inject.go`
- Modify: `cmd/openoctopus/resume.go`

**Step 1: Write the failing tests**
- `interrupt --format json` 返回 `interrupt_id`。
- `interrupt-all --format json` 返回 `requested_count`。
- `inject --format json` 返回 `message_id`。
- `resume --format json` 返回 `session_dir`。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestInterrupt|TestInject|TestResume' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 加 `--format`。
- 统一成功输出与错误包装。
- 改为复用通用 session 解析器。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestInterrupt|TestInject|TestResume' -v`
- Expected: PASS。

