# Event Bus 001 Plan 06: E2E 夹具、Harness 与黑盒测试

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `event-bus 001` 的宿主机黑盒 E2E，验证 `run` 能 bootstrap 总线，且通过测试专用 harness 可以稳定验证事件、锁、offset、interrupt 的真实文件系统行为，并将其纳入 `make check`。

**Architecture:** 复用现有 `e2e/conftest.py`、`openoctopus` 构建流程和 `e2e-test/` 工作目录；新增 `eventbus` fixtures、测试专用 harness、黑盒断言 helper 与 pytest 用例。E2E 继续只依赖 CLI / harness 输出与文件系统副作用，不直接调用 Go 私有函数。

**Tech Stack:** Python、pytest、Go CLI

---

### Task 1: 建立 event-bus E2E 脚手架与 harness

**Files:**
- Create: `e2e/eventbus/fixtures/valid-minimal/octopus.yaml`
- Create: `e2e/eventbus/fixtures/valid-repeat-bootstrap/octopus.yaml`
- Create: `e2e/eventbus/fixtures/valid-lock-conflict/octopus.yaml`
- Create: `e2e/eventbus/harness/main.go`
- Modify: `e2e/conftest.py`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 写失败测试，锁定脚手架可启动**

- 为 harness 写一个最小 smoke，用于执行 `bootstrap --help` 或等价命令。
- 断言三个 fixture 都能被复制到 `e2e-test/`。

Run: `python3 -m pytest e2e/eventbus -k smoke -v`
Expected: 先失败，说明脚手架和 harness 尚未建立。

**Step 2: 实现 event-bus E2E 公共夹具**

- 构建 `eventbus_harness_path`。
- 增加 `run_harness(...)` helper。
- 增加 `parse_session_dir(...)` helper。

**Step 3: 回跑 smoke 测试**

Run: `python3 -m pytest e2e/eventbus -k smoke -v`
Expected: 通过。

### Task 2: 建立 bootstrap 与事件链黑盒测试

**Files:**
- Create: `e2e/eventbus/test_run_bootstrap.py`
- Modify: `e2e/README.md`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 写失败测试，锁定 bootstrap 行为**

- `run` 成功后解析 `session created:` 输出。
- 断言 `bus/events.md` 含 `SESSION_CREATED`。
- 断言重复 bootstrap 不会产生第二条 `SESSION_CREATED`。

Run: `python3 -m pytest e2e/eventbus/test_run_bootstrap.py -v`
Expected: 先失败，说明 event-bus bootstrap 还未被黑盒锁定。

**Step 2: 实现 bootstrap 黑盒用例**

- 用 harness 执行重复 bootstrap。
- 增加读取 bus 文件内容的辅助断言。
- README 补充 `pytest e2e/eventbus -v` 执行方式。

**Step 3: 回跑 bootstrap 黑盒测试**

Run: `python3 -m pytest e2e/eventbus/test_run_bootstrap.py -v`
Expected: 通过。

### Task 3: 建立锁、offset、interrupt 与 `make check` 接入

**Files:**
- Create: `e2e/eventbus/test_bus_mutations.py`
- Modify: `Makefile`
- Modify: `e2e/README.md`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 写失败测试，锁定四类关键行为**

- stale lease 冲突拒绝。
- offset 回退阻断。
- interrupt 请求 / ack / clear 状态推进。
- 事件链损坏时读取失败。

Run: `python3 -m pytest e2e/eventbus/test_bus_mutations.py -v`
Expected: 先失败，说明总线行为还未被黑盒验证。

**Step 2: 实现黑盒 mutation 用例并接入检查链路**

- 使用 harness 调用 acquire / append / commit-offset / request-interrupt 等命令。
- 在 `Makefile` 中将 `e2e/eventbus` 纳入 `e2e` 或 `check`。
- README 同步说明 config + session + event-bus 的执行方式。

**Step 3: 回跑完整检查**

Run: `make check`
Expected: `go test` 与 `pytest e2e/config e2e/session e2e/eventbus` 全部通过。
