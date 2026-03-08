# CLI 001 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 OpenOctopus 首版正式 CLI 基线，让现有命令具备统一 `text/json` 输出、最小退出码和 `status` 状态观测能力。

**Architecture:** 采用“命令层统一输出 + `internal/cli` 只读支撑层”的做法。业务写逻辑仍停留在 `config`、`session`、`human-gate`、`orchestrator`、`role-runtime` 等模块中，CLI 只负责参数、展示、错误映射与状态聚合读取。

**Tech Stack:** Go、Cobra、标准库 `encoding/json`、Python `pytest`

---

### Task 1: 建立输出模型与退出码

**Files:**
- Create: `cmd/openoctopus/output.go`
- Create: `cmd/openoctopus/errors.go`
- Test: `cmd/openoctopus/output_test.go`
- Reference: `docs/cli/001/prd.md`

**Step 1: Write the failing test**
- 覆盖 success JSON 与 error JSON 的最小形状。
- 覆盖 `text` 与 `json` 两种格式的渲染分支。
- 覆盖 CLI 错误码 `1/2/3` 的映射。

**Step 2: Run test to verify it fails**
Run: `go test ./cmd/openoctopus -run 'TestOutput|TestError|TestExitCode' -v`
Expected: FAIL，提示输出工具和错误包装尚未定义。

**Step 3: Write minimal implementation**
- 实现格式解析。
- 实现 success / error 渲染。
- 实现 CLI 错误与退出码映射。

**Step 4: Run test to verify it passes**
Run: `go test ./cmd/openoctopus -run 'TestOutput|TestError|TestExitCode' -v`
Expected: PASS。

### Task 2: 建立 session 解析与状态聚合只读服务

**Files:**
- Create: `internal/cli/types.go`
- Create: `internal/cli/helpers.go`
- Create: `internal/cli/service.go`
- Test: `internal/cli/service_test.go`
- Reference: `docs/cli/001/prd.md`
- Reference: `docs/session/001/prd.md`

**Step 1: Write the failing test**
- 覆盖 session id / session dir 解析。
- 覆盖读取 `metadata.md`、`session.state.md`、`master_schedule.md`、`blockers.md` 后汇总为 `StatusSummary`。
- 覆盖 `session not found` 和 placeholder 回退行为。

**Step 2: Run test to verify it fails**
Run: `go test ./internal/cli -run 'TestResolve|TestReadStatus' -v`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 只实现读取与聚合，不实现写操作。
- 使用现有 Markdown bullet 协议做轻量解析。

**Step 4: Run test to verify it passes**
Run: `go test ./internal/cli -run 'TestResolve|TestReadStatus' -v`
Expected: PASS。

### Task 3: 给 `validate` / `run` 接入 `--format`

**Files:**
- Modify: `cmd/openoctopus/validate.go`
- Modify: `cmd/openoctopus/run.go`
- Modify: `cmd/openoctopus/command_test.go`

**Step 1: Write the failing test**
- `validate --format json` 成功输出合法 JSON。
- `validate --format json` 失败输出 stderr JSON，并返回 `config_validation_failed`。
- `run --format json` 成功输出 `session_id` / `session_dir`。

**Step 2: Run test to verify it fails**
Run: `go test ./cmd/openoctopus -run 'TestValidate|TestRun' -v`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 增加 `--format` flag。
- 把成功结果接到统一 renderer。
- 把配置校验失败映射到退出码 `2`。

**Step 4: Run test to verify it passes**
Run: `go test ./cmd/openoctopus -run 'TestValidate|TestRun' -v`
Expected: PASS。

### Task 4: 给 human-gate 正式命令接入 `--format`

**Files:**
- Modify: `cmd/openoctopus/interrupt.go`
- Modify: `cmd/openoctopus/interrupt_all.go`
- Modify: `cmd/openoctopus/inject.go`
- Modify: `cmd/openoctopus/resume.go`
- Modify: `cmd/openoctopus/human_gate_command_test.go`

**Step 1: Write the failing test**
- `interrupt --format json` 返回 `interrupt_id`。
- `interrupt-all --format json` 返回 `requested_count`。
- `inject --format json` 返回 `message_id`。
- `resume --format json` 返回 `session_dir`。

**Step 2: Run test to verify it fails**
Run: `go test ./cmd/openoctopus -run 'TestInterrupt|TestInject|TestResume' -v`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 新增 `--format`。
- 改为统一输出。
- 复用通用 session 解析服务。

**Step 4: Run test to verify it passes**
Run: `go test ./cmd/openoctopus -run 'TestInterrupt|TestInject|TestResume' -v`
Expected: PASS。

### Task 5: 实现 `status` 命令与主入口退出码

**Files:**
- Create: `cmd/openoctopus/status.go`
- Modify: `cmd/openoctopus/root.go`
- Modify: `cmd/openoctopus/main.go`
- Modify: `cmd/openoctopus/command_test.go`

**Step 1: Write the failing test**
- `status --format json` 输出 session 摘要。
- session 不存在时返回稳定错误。
- 主入口能把 `config validation failed` 映射到 `2`，`session not found` 映射到 `3`。

**Step 2: Run test to verify it fails**
Run: `go test ./cmd/openoctopus -run 'TestStatus|TestExecute' -v`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 新增 `status`。
- root 注册 `status`。
- 抽出共享执行函数给 `main.go` 使用。

**Step 4: Run test to verify it passes**
Run: `go test ./cmd/openoctopus -run 'TestStatus|TestExecute' -v`
Expected: PASS。

### Task 6: 建立 CLI E2E 与仓库检查入口

**Files:**
- Create: `e2e/cli/fixtures/valid-minimal/octopus.yaml`
- Create: `e2e/cli/fixtures/valid-deterministic-success/octopus.yaml`
- Create: `e2e/cli/fixtures/valid-interrupt-all/octopus.yaml`
- Create: `e2e/cli/test_validate_output.py`
- Create: `e2e/cli/test_run_status.py`
- Modify: `e2e/README.md`
- Modify: `Makefile`

**Step 1: Write the failing test**
- 覆盖 CLI 001 的 6 个黑盒核心场景。

**Step 2: Run test to verify it fails**
Run: `python3 -m pytest e2e/cli -v`
Expected: FAIL。

**Step 3: Write minimal implementation**
- 补 fixtures 与 pytest。
- 更新 `e2e/README.md` 与 `Makefile`，把 `e2e/cli` 接进统一入口。

**Step 4: Run test to verify it passes**
Run: `python3 -m pytest e2e/cli -v`
Expected: PASS。

### Task 7: Final verification

**Files:**
- Verify only

**Step 1: Run focused Go tests**
Run: `go test ./cmd/openoctopus ./internal/cli -v`

**Step 2: Run CLI E2E**
Run: `python3 -m pytest e2e/cli -v`

**Step 3: Run full repo checks**
Run: `make check`
Expected: PASS。

**Step 4: Sync docs tree references if changed**
- 同步 `docs/timeline.md`、`e2e/README.md` 与实际目录树。

**Note:** 仓库规则禁止 `git add` / `git commit`，因此本计划故意省略提交步骤。

