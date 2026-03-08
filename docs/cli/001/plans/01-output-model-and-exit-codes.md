# CLI 001 Plan 01: 输出模型与退出码基线

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 CLI 建立统一成功 / 失败输出模型、`text/json` 格式切换和最小退出码协议，避免各命令继续各写各的文本。

**Architecture:** 先在 `cmd/openoctopus` 中定义轻量输出与错误包装层，不改变底层业务服务，只在命令层统一成功输出、错误输出和退出码映射。首版只覆盖 `0/1/2/3` 四个退出码。

**Tech Stack:** Go、Cobra、标准库 `encoding/json`

---

### Task 1: 定义输出与错误模型

**Files:**
- Create: `cmd/openoctopus/output.go`
- Create: `cmd/openoctopus/errors.go`
- Test: `cmd/openoctopus/output_test.go`
- Reference: `docs/cli/001/prd.md`

**Step 1: Write the failing tests**
- 断言 success JSON 包含 `ok`、`command`、`data`。
- 断言 error JSON 输出到 stderr 时包含 `ok=false`、`error.code`、`error.message`。
- 断言 `text` 模式仍保留可读文本。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestOutput|TestError' -v`
- Expected: FAIL，提示输出工具尚未定义。

**Step 3: Write minimal implementation**
- 实现格式枚举与渲染函数。
- 实现 CLI 错误包装与退出码映射。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestOutput|TestError' -v`
- Expected: PASS。

