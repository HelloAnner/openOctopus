# CLI 001 Plan 05: `status` 命令与 root / main 接线

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 落地正式 `status` 命令，并把退出码映射接到 `main.go`，让 CLI 具备基础观测与稳定退出行为。

**Architecture:** `status` 调用 `internal/cli` 的只读服务聚合 session 状态；`main.go` 不再固定把任何错误都映射成 `1`，而是识别 CLI 错误码并 `os.Exit(code)`。

**Tech Stack:** Go、Cobra

---

### Task 1: 为 status 与退出码写失败测试

**Files:**
- Create: `cmd/openoctopus/status.go`
- Modify: `cmd/openoctopus/root.go`
- Modify: `cmd/openoctopus/main.go`
- Modify: `cmd/openoctopus/command_test.go`

**Step 1: Write the failing tests**
- `status --format json` 可输出当前 session 摘要。
- `status --session missing` 返回 `session not found` 错误。
- 主入口可将 `config validation failed` / `session not found` 映射到稳定退出码。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestStatus|TestExecute' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 增加 `status` 命令。
- 将 root command 接入 `status`。
- 抽出 `execute()` 或等价函数，供 `main.go` 与测试共享退出码逻辑。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestStatus|TestExecute' -v`
- Expected: PASS。

