# Orchestrator 001 Plan 06: E2E 夹具、Harness 与黑盒验证

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `orchestrator 001` 的宿主机黑盒 E2E，验证 `run` 能完成 orchestrator bootstrap 与首批任务分发，且通过测试专用 harness 可以稳定验证完成、重试、阻塞与人工输入吸收等真实文件系统行为，并将其纳入 `make check`。

**Architecture:** 复用现有 `e2e/conftest.py`、`openoctopus` 构建流程、`eventbus-harness` 和 `e2e-test/` 工作目录；新增 `orchestrator` fixtures、测试专用 harness、黑盒断言 helper 与 pytest 用例。E2E 继续只依赖 CLI / harness 输出与文件系统副作用，不直接调用 Go 私有函数。

**Tech Stack:** Python、pytest、Go CLI

---

### Task 1: 建立 orchestrator E2E 脚手架与 harness

**Files:**
- Create: `e2e/orchestrator/fixtures/valid-minimal/octopus.yaml`
- Create: `e2e/orchestrator/fixtures/valid-two-entry/octopus.yaml`
- Create: `e2e/orchestrator/fixtures/valid-human-message/octopus.yaml`
- Create: `e2e/orchestrator/harness/main.go`
- Modify: `e2e/conftest.py`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定脚手架可启动**

- 为 harness 写一个最小 smoke，用于执行 `tick --help` 或等价命令。
- 断言三个 fixture 都能被复制到 `e2e-test/`。

Run: `python3 -m pytest e2e/orchestrator -k smoke -v`
Expected: 先失败，说明脚手架和 harness 尚未建立。

**Step 2: 实现 orchestrator E2E 公共夹具**

- 构建 `orchestrator_harness_path`。
- 增加 `run_orchestrator_harness(...)` helper。
- 复用 `parse_session_dir(...)` 与 `read_text(...)` 风格辅助函数。

**Step 3: 回跑 smoke 测试**

Run: `python3 -m pytest e2e/orchestrator -k smoke -v`
Expected: 通过。

### Task 2: 建立 bootstrap、首批分发与完成路径黑盒测试

**Files:**
- Create: `e2e/orchestrator/test_run_bootstrap.py`
- Create: `e2e/orchestrator/test_orchestrator_tick.py`
- Modify: `e2e/README.md`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定三类关键行为**

- `run` 成功后生成真实 planner 文件。
- `run` 成功后写出 `context.md` / `inbox.md`。
- synthetic `SUCCESS` conclusion + `tick` 后能完成 workflow。

Run: `python3 -m pytest e2e/orchestrator/test_run_bootstrap.py e2e/orchestrator/test_orchestrator_tick.py -v`
Expected: 先失败，说明 orchestrator 仍未被黑盒锁定。

**Step 2: 实现 bootstrap 与 tick 黑盒用例**

- 用 harness 写 synthetic conclusion。
- 增加读取 planner 文件内容的辅助断言。
- README 补充 `pytest e2e/orchestrator -v` 执行方式。

**Step 3: 回跑 bootstrap 与 tick 黑盒测试**

Run: `python3 -m pytest e2e/orchestrator/test_run_bootstrap.py e2e/orchestrator/test_orchestrator_tick.py -v`
Expected: 通过。

### Task 3: 建立重试、阻塞、人工输入与 `make check` 接入

**Files:**
- Create: `e2e/orchestrator/test_orchestrator_guards.py`
- Modify: `Makefile`
- Modify: `e2e/README.md`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定四类关键行为**

- `NEEDS_RETRY` 会重发任务且递增 attempt。
- `BLOCKED` 会写 `blockers.md` 并推进到 `WAITING_HUMAN`。
- 新人工输入会推进 requirement snapshot 游标。
- `valid-two-entry` 会受 `max_parallel_roles` 限制。

Run: `python3 -m pytest e2e/orchestrator/test_orchestrator_guards.py -v`
Expected: 先失败，说明守卫行为尚未被黑盒验证。

**Step 2: 实现 guard 黑盒用例并接入检查链路**

- 使用 harness 调用 `append-human-message`、`write-conclusion`、`tick`。
- 在 `Makefile` 中将 `e2e/orchestrator` 纳入 `e2e` 或 `check`。
- README 同步说明 config + session + eventbus + orchestrator 的执行方式。

**Step 3: 回跑完整检查**

Run: `make check`
Expected: `go test` 与 `pytest e2e/config e2e/session e2e/eventbus e2e/orchestrator` 全部通过。
