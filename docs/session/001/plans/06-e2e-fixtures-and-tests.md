# Session 001 Plan 06: E2E 夹具与黑盒测试

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `session 001` 的宿主机黑盒 E2E，验证 `run` 成功时会创建标准 session 骨架，失败时不会留下半初始化目录，并将其纳入 `make check`。

**Architecture:** 复用现有 `e2e/conftest.py` 与构建流程，只新增 `session` 夹具、断言 helper 和黑盒测试文件。E2E 继续只依赖 CLI 输出与文件系统副作用，不读取 Go 内部结构。

**Tech Stack:** Python、pytest、Go CLI

---

### Task 1: 建立 `session` E2E 脚手架与 fixtures

**Files:**
- Create: `e2e/session/fixtures/valid-minimal/octopus.yaml`
- Create: `e2e/session/fixtures/valid-custom-sessions-dir/octopus.yaml`
- Create: `e2e/session/fixtures/valid-path-collision/octopus.yaml`
- Modify: `e2e/conftest.py`
- Reference: `docs/session/001/e2e.md`

**Step 1: 写失败测试，锁定 fixtures 能被会话测试读取**

- 补会话模块专用 fixture 定位 helper。
- 断言三个 fixture 目录都存在且可被测试复制到 `e2e-test/`。

Run: `python3 -m pytest e2e/session -k fixture -v`
Expected: 先失败，说明 session E2E 脚手架尚未建立。

**Step 2: 建立 session E2E 目录与样例 YAML**

- `valid-minimal`：最小合法配置。
- `valid-custom-sessions-dir`：显式自定义 `sessions_dir`。
- `valid-path-collision`：为冲突场景预留简单路径。

**Step 3: 回跑 fixture 脚手架测试**

Run: `python3 -m pytest e2e/session -k fixture -v`
Expected: 通过。

### Task 2: 建立成功场景黑盒测试

**Files:**
- Create: `e2e/session/test_run_session_init.py`
- Modify: `e2e/conftest.py`
- Modify: `e2e/README.md`

**Step 1: 写失败测试，锁定成功场景断言**

- `run` 成功后解析 `session created:` 输出。
- 断言关键目录和关键文件存在。
- 断言 `timeline.md` 包含 `SESSION_CREATED`。

Run: `python3 -m pytest e2e/session/test_run_session_init.py -v`
Expected: 先失败，说明 session 骨架尚未被黑盒验证。

**Step 2: 增加测试 helper**

- 增加 `parse_session_dir(stdout)`。
- 增加 `assert_session_skeleton(session_dir)`。
- README 补充 `pytest e2e/session -v` 的执行方式。

**Step 3: 回跑成功场景 E2E**

Run: `python3 -m pytest e2e/session/test_run_session_init.py -v`
Expected: 通过。

### Task 3: 建立失败回滚与 `make check` 接入

**Files:**
- Create: `e2e/session/test_run_session_failure.py`
- Modify: `Makefile`
- Modify: `e2e/README.md`

**Step 1: 写失败测试，锁定路径冲突回滚**

- 在 `valid-path-collision` 对应工作目录预创建同名普通文件。
- 断言 `run` 失败且无新的 session 子目录。

Run: `python3 -m pytest e2e/session/test_run_session_failure.py -v`
Expected: 先失败，说明失败回滚未被黑盒锁定。

**Step 2: 接入 `make check`**

- 将 `e2e/session` 纳入 `make e2e` 或等价检查目标。
- README 同步说明 config + session 的执行命令。

**Step 3: 回跑完整检查**

Run: `make check`
Expected: `go test` 与 `pytest e2e/config e2e/session` 全部通过。
