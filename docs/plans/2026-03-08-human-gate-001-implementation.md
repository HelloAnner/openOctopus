# Human Gate 001 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 OpenOctopus 补齐首版 `human-gate`，让正式 CLI 能完成人工中断、人工补充、恢复续跑与阻塞重入队的最小闭环。

**Architecture:** 新增 `internal/humangate` 作为人工协作服务层，负责 session 定位、消息落盘、interrupt 申请/清理与阻塞阶段恢复。CLI 只做参数解析；实际恢复继续复用已有 orchestrator / role-runtime 同步 loop，并修正 role-runtime 对 `ACKNOWLEDGED` interrupt 的暂停语义。

**Tech Stack:** Go、Cobra、pytest、现有 `event-bus` / `orchestrator` / `role-runtime` / `session` 模块。

---

### Task 1: Session 解析与 inject 协议

**Files:**
- Create: `internal/humangate/types.go`
- Create: `internal/humangate/helpers.go`
- Create: `internal/humangate/session.go`
- Create: `internal/humangate/messages.go`
- Test: `internal/humangate/service_test.go`

**Step 1: Write the failing tests**
- 覆盖 `--session` 传目录与传 session id 两种解析方式。
- 覆盖消息 ID 自增与 `inject` 写入块格式。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/humangate -run 'TestResolve|TestInject' -v`
- Expected: FAIL，提示包或函数尚不存在。

**Step 3: Write minimal implementation**
- 只实现 session 目录解析、消息块渲染、文件追加。
- 不提前实现 interrupt / resume。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/humangate -run 'TestResolve|TestInject' -v`
- Expected: PASS。

### Task 2: interrupt 服务与命令

**Files:**
- Modify: `internal/humangate/service.go`
- Create: `cmd/openoctopus/interrupt.go`
- Modify: `cmd/openoctopus/root.go`
- Test: `cmd/openoctopus/human_gate_command_test.go`

**Step 1: Write the failing tests**
- 覆盖 `interrupt` 命令写入 `INTERRUPT_REQUESTED` 与 `interrupts.md`。
- 覆盖缺少参数时的命令失败。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestInterrupt' -v`
- Expected: FAIL，命令不存在或 bus 文件未更新。

**Step 3: Write minimal implementation**
- 走 `eventbus.AcquireLock` + `RequestInterrupt`。
- 命令只返回简洁成功信息。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestInterrupt' -v`
- Expected: PASS。

### Task 3: role-runtime 中断闸门

**Files:**
- Create: `internal/roleruntime/interrupts.go`
- Modify: `internal/roleruntime/engine.go`
- Modify: `internal/roleruntime/engine_test.go`

**Step 1: Write the failing tests**
- 覆盖 `REQUESTED` -> ACK -> `INTERRUPTED`。
- 覆盖 `ACKNOWLEDGED` 且未 clear 时重复 tick 不执行 turn。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/roleruntime -run 'TestInterrupt' -v`
- Expected: FAIL，角色仍继续执行或状态不正确。

**Step 3: Write minimal implementation**
- 抽出 interrupt 查询逻辑。
- clear 前直接 no-op 返回。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/roleruntime -run 'TestInterrupt' -v`
- Expected: PASS。

### Task 4: resume 恢复与 blocked 重入队

**Files:**
- Modify: `internal/humangate/service.go`
- Create: `internal/humangate/schedule.go`
- Create: `cmd/openoctopus/resume.go`
- Modify: `cmd/openoctopus/run.go`
- Modify: `cmd/openoctopus/human_gate_command_test.go`

**Step 1: Write the failing tests**
- 覆盖 clear acknowledged interrupt 后恢复同一任务执行。
- 覆盖 `BLOCKED` 阶段恢复后 `task_id` 递增。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/humangate ./cmd/openoctopus -run 'TestResume' -v`
- Expected: FAIL，interrupt 未清理或 blocked 阶段未重新分发。

**Step 3: Write minimal implementation**
- 把 `BLOCKED` 阶段改为 `RETRY_PENDING` 并递增 attempt。
- 复用现有 orchestrator / role-runtime loop 继续推进。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/humangate ./cmd/openoctopus -run 'TestResume' -v`
- Expected: PASS。

### Task 5: interrupt-all 与 inject 命令收口

**Files:**
- Modify: `internal/humangate/service.go`
- Create: `cmd/openoctopus/interrupt_all.go`
- Create: `cmd/openoctopus/inject.go`
- Modify: `cmd/openoctopus/root.go`
- Modify: `cmd/openoctopus/human_gate_command_test.go`

**Step 1: Write the failing tests**
- 覆盖 `interrupt-all` 把 session 拉到 `WAITING_HUMAN`。
- 覆盖 `inject --message` / `inject --input` 两种路径。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestInterruptAll|TestInject' -v`
- Expected: FAIL，命令不存在或文件未更新。

**Step 3: Write minimal implementation**
- `interrupt-all` 只处理中断未完成角色、写 blockers、更新 session state。
- `inject` 只负责参数归一与服务调用。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestInterruptAll|TestInject' -v`
- Expected: PASS。

### Task 6: human-gate E2E

**Files:**
- Create: `e2e/human-gate/fixtures/valid-interrupt-resume/octopus.yaml`
- Create: `e2e/human-gate/fixtures/valid-blocked-resume/octopus.yaml`
- Create: `e2e/human-gate/fixtures/valid-interrupt-all/octopus.yaml`
- Create: `e2e/human-gate/test_human_gate_interrupt.py`
- Create: `e2e/human-gate/test_human_gate_resume.py`
- Modify: `e2e/README.md`
- Modify: `Makefile`

**Step 1: Write the failing tests**
- 覆盖 interrupt ACK 暂停、inject+resume 恢复、blocked 恢复、interrupt-all 四条主链路。

**Step 2: Run tests to verify they fail**
- Run: `python3 -m pytest e2e/human-gate -v`
- Expected: FAIL，目录或命令尚不存在。

**Step 3: Write minimal implementation**
- 复用现有 `run_cli`、`run_role_runtime_harness`。
- 不额外引入新的测试 harness。

**Step 4: Run tests to verify they pass**
- Run: `python3 -m pytest e2e/human-gate -v`
- Expected: PASS。

### Task 7: Final verification

**Files:**
- Verify only

**Step 1: Run focused Go tests**
- Run: `go test ./internal/humangate ./internal/roleruntime ./cmd/openoctopus -v`

**Step 2: Run human-gate E2E**
- Run: `python3 -m pytest e2e/human-gate -v`

**Step 3: Run full repo checks**
- Run: `make check`
- Expected: PASS。

**Step 4: Sync docs tree references if changed**
- 核对 `docs/timeline.md`、`e2e/README.md` 与真实目录结构一致。

**Note:** 仓库规则禁止 `git add` / `git commit`，因此本计划故意省略提交步骤。
