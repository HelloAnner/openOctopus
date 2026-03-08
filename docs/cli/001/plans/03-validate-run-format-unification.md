# CLI 001 Plan 03: `validate` / `run` 统一输出协议

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让 `validate` / `run` 在保留原有业务行为的前提下，提供 `--format text|json` 的正式输出协议。

**Architecture:** 只改命令层。成功输出走统一 renderer，失败时把已有错误分类和信息包装为 CLI 错误结果，不改底层 `config` / `session` / `orchestrator` / `role-runtime` 逻辑。

**Tech Stack:** Go、Cobra

---

### Task 1: 为 validate / run 写失败测试

**Files:**
- Modify: `cmd/openoctopus/command_test.go`
- Modify: `cmd/openoctopus/validate.go`
- Modify: `cmd/openoctopus/run.go`
- Reference: `docs/cli/001/e2e.md`

**Step 1: Write the failing tests**
- `validate --format json` 成功输出 JSON。
- `validate --format json` 失败输出 stderr JSON，错误码是 `config_validation_failed`。
- `run --format json` 成功输出 `session_id` / `session_dir`。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestValidate|TestRun' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 为两个命令加 `--format` flag。
- 接统一输出渲染器。
- 失败时返回稳定 CLI 错误。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestValidate|TestRun' -v`
- Expected: PASS。

