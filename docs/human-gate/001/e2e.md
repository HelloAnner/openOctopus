# Human Gate 模块 001 阶段 E2E 方案

## 1. 目标

`human-gate 001` 的 E2E 不追求覆盖未来所有人工审批玩法，而是验证首版最重要的 4 条黑盒链路：

1. 人工中断请求真的能写进 session，总线和角色状态一致。
2. 中断被 ACK 后，角色在 clear 前不会继续执行。
3. 人工补充消息后，可以通过 `resume` 清 interrupt 并继续跑完。
4. `BLOCKED` 工作流在人工补充后，可以通过 `resume` 重新入队并继续推进。

---

## 2. 测试原则

- 继续复用宿主机真实 `~/.codex/` 环境，但 `human-gate 001` 主链路仍以 deterministic executor 为主，保证测试稳定。
- 全部走正式 CLI：`run`、`interrupt`、`interrupt-all`、`inject`、`resume`。
- 只在必要时复用现有 harness：
  - `role-runtime-harness` 用来显式触发角色 tick，验证 ACK 前后行为。
  - 其余流程优先使用正式 CLI，避免再次退化为“测试工具直接改文件”。

---

## 3. 场景矩阵

| 场景 | 入口 | 重点验证 |
| --- | --- | --- |
| 单角色中断并 ACK | `run` + `interrupt` + `tick-role` | `interrupts.md` 从 `REQUESTED` 变 `ACKNOWLEDGED`，角色变 `INTERRUPTED`，且无新 turn |
| 注入后恢复单角色 | `inject` + `resume` | `human_messages.md` 追加消息，interrupt 被 clear，角色继续执行并完成 |
| 阻塞后人工恢复 | `run(BLOCKED)` + `inject` + `resume` | `master_schedule.md` 里的阻塞阶段重新入队，生成新的 task，工作流继续推进 |
| 全量中断等待人工 | `interrupt-all` | `session.state.md` 立即为 `WAITING_HUMAN`，`blockers.md` 留下原因，interrupt 记录覆盖全部未完成角色 |

---

## 4. 夹具设计

### 4.1 `valid-interrupt-resume`

用途：验证最小“中断 -> ACK -> 注入 -> 恢复 -> 成功”闭环。

配置要求：

- 单角色 `agent_a`
- 单阶段 `stage_a`
- executor 使用 `deterministic`

运行步骤：

1. `run` 创建 session，但禁用自动 role-runtime loop。
2. `interrupt` 指向 `agent_a`。
3. `tick-role` 一次，确认 interrupt 变 `ACKNOWLEDGED`，角色状态为 `INTERRUPTED`，且 turns 目录仍为空。
4. `inject --message "继续执行 stage_a"`。
5. `resume`，并通过环境变量让 deterministic 返回 `SUCCESS`。
6. 断言工作流 `COMPLETED`。

### 4.2 `valid-blocked-resume`

用途：验证 `BLOCKED` 工作流的人为恢复。

配置要求：

- 单角色 `agent_a`
- deterministic 首次返回 `BLOCKED`，恢复后返回 `SUCCESS`

运行步骤：

1. `run` 时启用自动 role-runtime loop，并让 deterministic 首次返回 `BLOCKED`。
2. 断言 session 进入 `WAITING_HUMAN`，`planner/blockers.md` 记录阻塞原因。
3. `inject --message "补充约束后继续"`。
4. `resume`，并把 deterministic 结果改成 `SUCCESS`。
5. 断言新的 `task_id` 已出现，最终工作流完成。

### 4.3 `valid-interrupt-all`

用途：验证批量中断与人工等待态。

配置要求：

- 两个 entry stage 或至少两个未完成角色，便于验证批量 interrupt

运行步骤：

1. `run` 创建 session，禁用自动 role-runtime loop。
2. `interrupt-all --reason "manual review"`。
3. 断言 `session.state.md` 为 `WAITING_HUMAN`。
4. 断言 `planner/blockers.md` 包含原因。
5. 断言 `bus/interrupts.md` 至少存在两条 `REQUESTED` 记录。

---

## 5. 断言重点

### 5.1 中断链路

- `bus/events.md` 中存在 `INTERRUPT_REQUESTED` / `INTERRUPT_ACKNOWLEDGED` / `INTERRUPT_CLEARED`
- `bus/interrupts.md` 的当前态与事件链一致
- `roles/{role_id}/state.md` 在 ACK 后为 `INTERRUPTED`

### 5.2 人工消息链路

- `planner/human_messages.md` 生成递增 `message_id`
- `source` 为 `human-gate`
- 若指定 `--role`，块中记录 `target_role_id`
- `planner/requirement.snapshot.md` 会在 resume 后吸收新消息

### 5.3 恢复链路

- clear 前 role-runtime 不生成新 turn
- `resume` 后 `ACKNOWLEDGED` interrupt 变为 `CLEARED`
- `BLOCKED` 阶段重新入队后得到新的 `task_id`
- 工作流恢复后 `session.state.md` 不再停留在 `WAITING_HUMAN`

---

## 6. 测试文件规划

建议新增：

- `e2e/human-gate/fixtures/valid-interrupt-resume/octopus.yaml`
- `e2e/human-gate/fixtures/valid-blocked-resume/octopus.yaml`
- `e2e/human-gate/fixtures/valid-interrupt-all/octopus.yaml`
- `e2e/human-gate/test_human_gate_interrupt.py`
- `e2e/human-gate/test_human_gate_resume.py`

并同步更新：

- `e2e/README.md`
- `Makefile`

---

## 7. 通过标准

以下命令通过，视为 `human-gate 001` E2E 合格：

```bash
python3 -m pytest e2e/human-gate -v
make check
```
