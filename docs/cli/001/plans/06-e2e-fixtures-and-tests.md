# CLI 001 Plan 06: E2E 夹具、测试与最终验证

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `e2e/cli` 黑盒测试，验证 JSON 输出、状态观测和退出码，并把 CLI 001 纳入统一检查入口。

**Architecture:** 复用现有 `e2e/conftest.py` 的二进制构建与真实 Codex 检查，只增加 `cli` fixtures 和 pytest 文件；最后把 `Makefile` 与 `e2e/README.md` 同步到最新目录树。

**Tech Stack:** Python、pytest、subprocess

---

### Task 1: 为 CLI E2E 写失败测试

**Files:**
- Create: `e2e/cli/fixtures/valid-minimal/octopus.yaml`
- Create: `e2e/cli/fixtures/valid-deterministic-success/octopus.yaml`
- Create: `e2e/cli/fixtures/valid-interrupt-all/octopus.yaml`
- Create: `e2e/cli/test_validate_output.py`
- Create: `e2e/cli/test_run_status.py`
- Modify: `e2e/README.md`
- Modify: `Makefile`

**Step 1: Write the failing tests**
- 覆盖 `validate --format json` 成功 / 失败。
- 覆盖 `run --format json`。
- 覆盖 `status --format json` 的完成态与等待人工态。
- 覆盖 session 不存在退出码。

**Step 2: Run tests to verify they fail**
- Run: `python3 -m pytest e2e/cli -v`
- Expected: FAIL，提示 fixtures 或命令能力尚未齐备。

**Step 3: Write minimal implementation**
- 补 fixtures。
- 补 pytest。
- 更新文档与 `Makefile`。

**Step 4: Run tests to verify they pass**
- Run: `python3 -m pytest e2e/cli -v`
- Expected: PASS。

