# Recovery 001 Plan 06: E2E 夹具与黑盒验证

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 recovery 首版的黑盒 E2E，证明 `recover` 能对已创建 session 做续跑、修复与严格失败保护。

**Architecture:** 复用现有宿主机直跑与 deterministic executor 方案，保持“真实 CLI + 真实 session 目录 + 黑盒断言”。不引入 mock 服务，不绕过命令层。

**Tech Stack:** Python、pytest、subprocess、Go CLI

---

### Task 1: 准备 recovery E2E fixtures

**Files:**
- Create: `e2e/recovery/fixtures/valid-dispatched-session/octopus.yaml`
- Create: `e2e/recovery/fixtures/valid-missing-session-state/octopus.yaml`
- Create: `e2e/recovery/fixtures/valid-waiting-human/octopus.yaml`
- Create: `e2e/recovery/fixtures/valid-broken-events/octopus.yaml`
- Reference: `docs/recovery/001/e2e.md`

**Step 1: 写最小 fixture**

- 全部保持单 stage、单 role。
- 使用 deterministic executor，避免依赖线上模型。

**Step 2: 手工检查路径与命名**

- 确认目录结构与 `docs/timeline.md`、`e2e/README.md` 中的模块索引一致。

### Task 2: 编写黑盒 E2E

**Files:**
- Create: `e2e/recovery/test_recovery_flow.py`
- Modify: `e2e/README.md`
- Test: `python3 -m pytest e2e/recovery -v`

**Step 1: 写失败测试，覆盖四条链路**

- 已分发 session 续跑成功。
- 缺失 `session.state.md` 自动修复。
- 损坏 `events.md` 严格失败。
- `WAITING_HUMAN` 返回 `continued=false`。

Run: `python3 -m pytest e2e/recovery -v`
Expected: 先失败，说明 recovery E2E 尚未接通。

**Step 2: 对齐实现并回跑 E2E**

- 保持断言只依赖 CLI 输出与文件系统结果。

**Step 3: 扩大验证范围**

Run: `make check`
Expected: 全量检查通过。

