# Role Runtime 001 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 OpenOctopus 补齐首版 `role-runtime`，让 `run` 能从 orchestrator 分发继续推进到真实角色执行、turn 落盘、结论回写与工作流收口。

**Architecture:** 新增 `internal/roleruntime` 作为独立运行时层，负责消费 `context.md` / `inbox.md`、写 `state.md` / `heartbeat.md` / `outbox.md` / `conclusion.md` / `turns/*.md`，并通过 deterministic executor 先跑通最小闭环。`cmd/openoctopus/run.go` 只增加一个有界同步 runtime loop，不让 orchestrator 越权写角色运行态文件。

**Tech Stack:** Go、Cobra、pytest、宿主机 `codex` CLI、现有 `event-bus` / `orchestrator` / `session` 模块。

---

### Task 1: 角色运行态与 turn 渲染

**Files:**
- Create: `internal/roleruntime/types.go`
- Create: `internal/roleruntime/state.go`
- Create: `internal/roleruntime/turns.go`
- Test: `internal/roleruntime/state_test.go`
- Test: `internal/roleruntime/turns_test.go`

**Step 1: Write the failing tests**
- 覆盖 `state.md` 初始渲染、重复读取、`session_generation` 前进。
- 覆盖 `turns/0001-input.md` / `turns/0001-output.md` 编号、字段和路径。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/roleruntime -run 'TestRender|TestRead' -v`
- Expected: FAIL，提示缺少 `roleruntime` 包或目标函数未定义。

**Step 3: Write minimal implementation**
- 只实现 state / heartbeat / reset / turn 的最小读写与渲染。
- 不提前实现执行器和 run 集成。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/roleruntime -run 'TestRender|TestRead' -v`
- Expected: PASS。

### Task 2: deterministic 执行器与单角色 tick

**Files:**
- Create: `internal/roleruntime/errors.go`
- Create: `internal/roleruntime/executor.go`
- Create: `internal/roleruntime/executor_deterministic.go`
- Create: `internal/roleruntime/engine.go`
- Test: `internal/roleruntime/executor_deterministic_test.go`
- Test: `internal/roleruntime/engine_test.go`

**Step 1: Write the failing tests**
- 覆盖 deterministic 结果序列 `SUCCESS`、`NEEDS_RETRY,SUCCESS`、`BLOCKED`。
- 覆盖同一 `task_id + inbox_version + context_version + session_generation` 重复 tick 不重复生成新 turn。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/roleruntime -run 'TestDeterministic|TestTick' -v`
- Expected: FAIL，提示执行器或 `TickRole` 未定义。

**Step 3: Write minimal implementation**
- 从环境变量读取 deterministic 结果序列。
- 解析 `context.md` / `inbox.md`，生成 turn 输入输出、`conclusion.md`、`outbox.md`。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/roleruntime -run 'TestDeterministic|TestTick' -v`
- Expected: PASS。

### Task 3: interrupt / reset 守卫

**Files:**
- Modify: `internal/roleruntime/engine.go`
- Test: `internal/roleruntime/engine_test.go`

**Step 1: Write the failing tests**
- 覆盖 interrupt 先到达时不启动新 turn。
- 覆盖 reset 请求应用后 `session_generation` 递增，历史 turn 保留。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/roleruntime -run 'TestInterrupt|TestReset' -v`
- Expected: FAIL，状态不符合预期。

**Step 3: Write minimal implementation**
- tick 前读取 interrupt / reset。
- interrupt 命中则写 `INTERRUPTED`；reset 命中则提升 generation 并清当前任务态。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/roleruntime -run 'TestInterrupt|TestReset' -v`
- Expected: PASS。

### Task 4: `run` 集成最小闭环

**Files:**
- Modify: `cmd/openoctopus/run.go`
- Modify: `cmd/openoctopus/command_test.go`
- Modify: `Makefile`

**Step 1: Write the failing tests**
- 为 `run` 增加 deterministic success / retry 的命令级测试。
- 断言 `run` 结束后存在角色 turn 与 `conclusion.md`，并完成工作流收口。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestRunCommand.*RoleRuntime' -v`
- Expected: FAIL，角色文件未生成或状态未推进。

**Step 3: Write minimal implementation**
- 在 `run` 中加入有界同步 runtime loop。
- 只允许基于 runtime / orchestrator 的公开能力推进，不直接手改角色文件。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestRunCommand.*RoleRuntime' -v`
- Expected: PASS。

### Task 5: role-runtime harness 与 E2E

**Files:**
- Create: `e2e/role-runtime/harness/main.go`
- Create: `e2e/role-runtime/fixtures/valid-deterministic-success/octopus.yaml`
- Create: `e2e/role-runtime/fixtures/valid-retry-once/octopus.yaml`
- Create: `e2e/role-runtime/fixtures/valid-blocked/octopus.yaml`
- Create: `e2e/role-runtime/fixtures/valid-interrupt-before-start/octopus.yaml`
- Create: `e2e/role-runtime/fixtures/valid-reset-generation/octopus.yaml`
- Create: `e2e/role-runtime/test_role_runtime_bootstrap.py`
- Create: `e2e/role-runtime/test_role_runtime_tick.py`
- Create: `e2e/role-runtime/test_role_runtime_guards.py`
- Modify: `e2e/conftest.py`
- Modify: `e2e/README.md`
- Modify: `Makefile`

**Step 1: Write the failing tests**
- 覆盖 deterministic success / retry / blocked / interrupt / reset 五个黑盒链路。

**Step 2: Run tests to verify they fail**
- Run: `python3 -m pytest e2e/role-runtime -v`
- Expected: FAIL，目录或 harness 尚不存在。

**Step 3: Write minimal implementation**
- 补 harness。
- 补 fixtures。
- 接入 `Makefile` 的 `e2e` 目标。

**Step 4: Run tests to verify they pass**
- Run: `python3 -m pytest e2e/role-runtime -v`
- Expected: PASS。

### Task 6: Codex 执行器骨架与全量验证

**Files:**
- Create: `internal/roleruntime/executor_codex.go`
- Test: `internal/roleruntime/executor_codex_test.go`
- Optional later fixture: `e2e/role-runtime/fixtures/valid-codex-smoke/octopus.yaml`
- Optional later test: `e2e/role-runtime/test_role_runtime_codex.py`

**Step 1: Write the failing tests**
- 覆盖 `provider=codex` 时命令拼装正确、stdin prompt 写入正确、输出文件可回读。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/roleruntime -run 'TestCodex' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 用 `codex exec - --skip-git-repo-check -C <dir> -o <file>` 作为最小非交互执行器。
- smoke E2E 仅在 deterministic 主链路稳定后再补。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/roleruntime -run 'TestCodex' -v`
- Expected: PASS。

### Task 7: Final verification

**Files:**
- Verify only

**Step 1: Run focused Go tests**
- Run: `go test ./internal/roleruntime ./cmd/openoctopus -v`

**Step 2: Run role-runtime E2E**
- Run: `python3 -m pytest e2e/role-runtime -v`

**Step 3: Run full repo checks**
- Run: `make check`
- Expected: PASS。

**Step 4: Sync docs tree references if changed**
- 若新增了 `docs/plans/` 或新的 `e2e/role-runtime/` 目录引用，需要同步相关说明文档。

**Note:** 仓库规则禁止 `git add` / `git commit`，因此本计划故意省略提交步骤。
