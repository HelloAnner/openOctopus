# Config 001 Plan 06: E2E 夹具与黑盒测试

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 按 `docs/config/001/e2e.md` 建立 config 首版黑盒 E2E，证明 `validate` / `run` 在真实 Docker 环境里对合法和非法配置都能给出稳定结果。

**Architecture:** E2E 只通过 CLI 命令、退出码、标准输出和文件系统副作用断言，不读取内部 Go 对象。通过固定 fixtures 覆盖最小合法、优先级覆盖、结构错误、引用错误、安全错误和策略错误几个核心场景。

**Tech Stack:** Python、`pytest`、`requests`、`docker compose`

---

### Task 1: 建立 E2E 基础脚手架

**Files:**
- Create: `e2e/requirements.txt`
- Create: `e2e/conftest.py`
- Create: `e2e/docker-compose.test.yml`
- Create: `e2e/config/config.test.yaml`
- Create: `e2e/config/.env.test`
- Create: `e2e/README.md`
- Test: `e2e/config/test_validate.py`
- Reference: `docs/config/001/e2e.md:38`

**Step 1: 写失败测试，确认测试框架可启动**

- 建立最小 `pytest` 用例，至少能启动容器并执行一次 `openoctopus --help` 或等价命令。

Run: `pytest e2e/config -k smoke -v`
Expected: 先失败，说明 E2E 脚手架未齐。

**Step 2: 实现 E2E 公共 fixture**

- 提供 `clean_environment()`。
- 提供 `run_cli()`。
- 提供 `assert_no_session_created()`。

**Step 3: 回跑 smoke 测试**

Run: `pytest e2e/config -k smoke -v`
Expected: smoke 通过。

### Task 2: 建立 fixtures 样例目录

**Files:**
- Create: `e2e/config/fixtures/valid-minimal/octopus.yaml`
- Create: `e2e/config/fixtures/valid-env-override/octopus.yaml`
- Create: `e2e/config/fixtures/invalid-syntax/octopus.yaml`
- Create: `e2e/config/fixtures/invalid-missing-role/octopus.yaml`
- Create: `e2e/config/fixtures/invalid-shell-security/octopus.yaml`
- Create: `e2e/config/fixtures/invalid-immutable-conflict/octopus.yaml`
- Create: `e2e/config/fixtures/invalid-threshold/octopus.yaml`
- Reference: `docs/config/001/e2e.md:88`
- Reference: `docs/config/001/yaml-rules.md`

**Step 1: 写失败测试，锁定 fixtures 语义**

- 逐个夹具写最小断言，先确保它们真的覆盖对应错误类型。

Run: `pytest e2e/config/test_validate.py -v`
Expected: 先失败，说明 fixtures 或断言未完成。

**Step 2: 实现 fixtures**

- 合法夹具必须可通过 validate。
- 非法夹具每个只制造一种主错误，避免混叠。

**Step 3: 回跑 validate 测试**

Run: `pytest e2e/config/test_validate.py -v`
Expected: 用例可稳定区分合法与非法夹具。

### Task 3: 建立 `run` 阻断和优先级 E2E

**Files:**
- Create: `e2e/config/test_run_gate.py`
- Modify: `e2e/conftest.py`
- Reference: `docs/config/001/e2e.md:130`

**Step 1: 写失败测试，锁定两个关键行为**

- 非法配置执行 `run` 时不创建 session。
- env 覆盖 YAML 时，最终以 env 生效。

Run: `pytest e2e/config/test_run_gate.py -v`
Expected: 先失败，说明 run 阻断或优先级行为还未打通。

**Step 2: 实现对应 E2E 用例**

- 断言退出码。
- 断言输出错误类别。
- 断言 `.octopus/sessions` 为空。

**Step 3: 回跑 config E2E 全量**

Run: `docker compose -f e2e/docker-compose.test.yml down -v && pytest e2e/config -v`
Expected: 全部通过且结果稳定。
