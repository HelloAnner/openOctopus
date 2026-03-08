# Recovery 001 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 OpenOctopus 落地首版 recovery 模块，补齐正式 `recover` CLI、增量 checkpoint、恢复视图修复与黑盒 E2E。

**Architecture:** 采用 `cmd -> internal/recovery -> existing orchestrator/role-runtime` 的分层方式。`internal/recovery` 负责 session 校验、事件链校验、checkpoint 写入、恢复视图归约、`session.state.md` / `blockers.md` / `audit/replay.md` 修复，以及在安全前提下复用现有 loop 继续执行。human-gate 仍独占 `WAITING_HUMAN` 场景，recovery 不越权替代 `resume`。

**Tech Stack:** Go、Cobra、pytest、标准库

---

### Task 1: 建立 recovery 类型、错误与会话校验

**Files:**
- Create: `internal/recovery/types.go`
- Create: `internal/recovery/errors.go`
- Create: `internal/recovery/validate.go`
- Test: `internal/recovery/service_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: Write the failing test**

- 覆盖 `RecoverOptions` / `RecoverResult` 形状。
- 覆盖缺失 `effective_config.yaml` / `events.md` 返回稳定错误。
- 覆盖缺失 `session.state.md` 进入可修复列表。

Run: `go test ./internal/recovery -run 'TestRecoveryTypes|TestValidateSessionLayout|TestValidateEventChain' -v`
Expected: FAIL，提示 recovery 包或对应能力尚未定义。

**Step 2: Write minimal implementation**

- 实现基础类型与稳定错误。
- 实现关键 session 布局校验。
- 实现 `events.md` 哈希链校验。

**Step 3: Run test to verify it passes**

Run: `go test ./internal/recovery -run 'TestRecoveryTypes|TestValidateSessionLayout|TestValidateEventChain' -v`
Expected: PASS。

### Task 2: 落地 checkpoint writer

**Files:**
- Create: `internal/recovery/checkpoint.go`
- Create: `internal/recovery/checkpoint_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: Write the failing test**

- 覆盖 `0001-kind.md` 递增命名。
- 覆盖 checkpoint 内容至少包含 `workflow_status`、`current_stage_id`、`last_event`。

Run: `go test ./internal/recovery -run 'TestRenderCheckpoint|TestWriteCheckpoint' -v`
Expected: FAIL，说明 checkpoint writer 未建立。

**Step 2: Write minimal implementation**

- 扫描 checkpoint 目录中的最大序号。
- 生成标准文件名与内容模板。
- 采用原子写方式落盘并返回相对路径。

**Step 3: Run test to verify it passes**

Run: `go test ./internal/recovery -run 'TestRenderCheckpoint|TestWriteCheckpoint' -v`
Expected: PASS。

### Task 3: 落地恢复视图归约、修复与 replay 报告

**Files:**
- Create: `internal/recovery/reducer.go`
- Create: `internal/recovery/repair.go`
- Create: `internal/recovery/reducer_test.go`
- Reference: `docs/recovery/001/e2e.md`

**Step 1: Write the failing test**

- 覆盖 `COMPLETED` / `RUNNING` / `WAITING_HUMAN` / `FAILED` 归约。
- 覆盖重建 `session.state.md`。
- 覆盖 `blockers.md` 修复与 `audit/replay.md` 输出。

Run: `go test ./internal/recovery -run 'TestReduceWorkflowStatus|TestRepairSessionFiles|TestWriteReplayReport' -v`
Expected: FAIL，说明恢复视图与修复逻辑尚未实现。

**Step 2: Write minimal implementation**

- 从 `master_schedule.md`、`interrupts.md`、角色结论归约 workflow status。
- 修复 `session.state.md` 与 `planner/blockers.md`。
- 覆盖写 `audit/replay.md`。

**Step 3: Run test to verify it passes**

Run: `go test ./internal/recovery -run 'TestReduceWorkflowStatus|TestRepairSessionFiles|TestWriteReplayReport' -v`
Expected: PASS。

### Task 4: 组装 recovery 服务并接入 CLI

**Files:**
- Create: `internal/recovery/service.go`
- Create: `cmd/openoctopus/recover.go`
- Modify: `cmd/openoctopus/root.go`
- Modify: `cmd/openoctopus/errors.go`
- Modify: `cmd/openoctopus/command_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: Write the failing test**

- 覆盖 `recover` 成功输出文本 / JSON。
- 覆盖 `WAITING_HUMAN` 返回 `continued=false`。
- 覆盖损坏事件链时返回稳定错误。

Run: `go test ./cmd/openoctopus -run TestRecoverCommand -v`
Expected: FAIL，说明命令尚未注册或结果不符合协议。

**Step 2: Write minimal implementation**

- 组装 `internal/recovery.Service.Recover(...)`。
- 接入 `openoctopus recover` 命令。
- 映射稳定输出与错误码。

**Step 3: Run test to verify it passes**

Run: `go test ./cmd/openoctopus -run TestRecoverCommand -v`
Expected: PASS。

### Task 5: 在 orchestrator / human-gate 边界接入 checkpoint

**Files:**
- Modify: `internal/orchestrator/engine.go`
- Modify: `internal/humangate/service.go`
- Modify: `internal/orchestrator/engine_test.go`
- Modify: `internal/humangate/service_test.go`
- Reference: `docs/recovery/001/prd.md`

**Step 1: Write the failing test**

- dispatch 后生成 `stage-...-dispatched` checkpoint。
- stage 完成 / blocked / failed 后生成结果 checkpoint。
- `interrupt-all` 与 `resume` 生成 session 级 checkpoint。

Run: `go test ./internal/orchestrator ./internal/humangate -run TestCheckpoint -v`
Expected: FAIL，说明 checkpoint hook 尚未接入。

**Step 2: Write minimal implementation**

- 只在边界状态变化时写 checkpoint。
- 不改变 orchestrator / human-gate 原有职责分层。

**Step 3: Run test to verify it passes**

Run: `go test ./internal/orchestrator ./internal/humangate -run TestCheckpoint -v`
Expected: PASS。

### Task 6: 建立 recovery E2E 并完成全量验证

**Files:**
- Create: `e2e/recovery/fixtures/valid-dispatched-session/octopus.yaml`
- Create: `e2e/recovery/fixtures/valid-missing-session-state/octopus.yaml`
- Create: `e2e/recovery/fixtures/valid-waiting-human/octopus.yaml`
- Create: `e2e/recovery/fixtures/valid-broken-events/octopus.yaml`
- Create: `e2e/recovery/test_recovery_flow.py`
- Modify: `e2e/README.md`
- Modify: `docs/timeline.md`

**Step 1: Write the failing test**

- 已分发 session 可被 `recover` 续跑完成。
- 缺失 `session.state.md` 时可自动修复。
- 损坏 `events.md` 时严格失败。
- `WAITING_HUMAN` 时不自动继续。

Run: `python3 -m pytest e2e/recovery -v`
Expected: FAIL，说明 recovery E2E 或实现尚未完成。

**Step 2: Write minimal implementation**

- 准备 recovery fixtures。
- 让测试只通过真实 CLI 和 session 文件断言。
- 同步更新 `e2e/README.md` 与 `docs/timeline.md`。

**Step 3: Run test to verify it passes**

Run: `python3 -m pytest e2e/recovery -v`
Expected: PASS。

### Task 7: 全量验证

**Files:**
- Reference: `Makefile`

**Step 1: Run focused verification**

Run: `go test ./internal/recovery ./internal/orchestrator ./internal/humangate ./cmd/openoctopus -v`
Expected: PASS。

**Step 2: Run full project verification**

Run: `make check`
Expected: PASS。
