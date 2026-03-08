# Recovery 模块 001 阶段 PRD

## 1. 阶段定位

`recovery 001` 的目标不是一次性做完整的“事件回放平台”，而是先把 **单机会话在中断、进程退出、状态文件缺失后的最小恢复闭环** 跑通。

当前仓库已经具备以下基础：

1. `session 001` 已落地 `state/effective_config.yaml` 与 `state/checkpoints/0000-init.md`。
2. `event-bus 001` 已提供 `events.md`、哈希链、`offsets.md` 与 `interrupts.md`。
3. `orchestrator 001` 已能基于 `master_schedule.md`、`conclusion.md`、`human_messages.md` 推进主控状态。
4. `role-runtime 001` 已能基于 `context.md` / `inbox.md` / `conclusion.md` 继续执行单角色回合。
5. `human-gate 001` 已覆盖 `WAITING_HUMAN` 场景下的人工恢复入口。

但系统仍缺一层正式的 recovery 门面，导致几个问题没有解决：

1. 用户无法对“不是等待人工，而是进程退出 / 文件损坏 / 运行中断”的 session 做正式恢复。
2. `state/checkpoints/` 目前只有 `0000-init.md`，缺少后续可用的阶段边界快照。
3. `audit/replay.md` 仍停留在占位文件，恢复过程没有审计结果。
4. `session.state.md`、`planner/blockers.md`、`master_schedule.md` 一旦出现缺失或不一致，没有统一修复入口。

因此，`001` 的职责非常明确：**把已有 session / bus / orchestrator / role-runtime 协议收口成一个“可验证、可修复、可继续执行”的首版恢复模块。**

---

## 2. 本阶段要解决的问题

### 2.1 当前问题

当前恢复能力仍处在“协议预留，但未正式落地”的状态：

1. **没有正式恢复命令**：`resume` 只面向 `WAITING_HUMAN`，并不处理普通崩溃恢复。
2. **checkpoint 基线不足**：除了初始化快照，没有阶段开始 / 阶段完成 / 等待人工等关键恢复锚点。
3. **状态修复缺位**：如果 `session.state.md` 或 `blockers.md` 缺失，当前只能手工改文件。
4. **回放审计缺失**：恢复前做了什么检查、修复了什么、为什么可以继续，没有落到 `audit/replay.md`。
5. **进程退出后的续跑缺口**：session 已经创建、stage 已分发，但主进程退出后没有单独入口把工作流接着推进。

### 2.2 001 阶段目标

`recovery 001` 建立一个保守但可用的恢复协议：

1. 正式提供 `openoctopus recover --session <session_id|session_dir>` 命令。
2. 对恢复前置条件做严格校验：`effective_config.yaml`、`events.md` 哈希链、关键目录存在性。
3. 引入首版增量 checkpoint，在阶段开始、阶段完成、等待人工、恢复续跑时落盘。
4. 恢复时按 `events.md -> latest checkpoint -> master_schedule.md` 的顺序重建恢复视图，并修复最小必要文件。
5. 能对 `INITIAL` / `READY` / `RUNNING` 的单机会话继续推进；`WAITING_HUMAN` 只报告需人工恢复，不自动越权继续。
6. 把恢复检查与修复摘要写入 `audit/replay.md`，形成可审计证据链。

---

## 3. 范围定义

### 3.1 001 范围内

- `internal/recovery` 服务层与正式 CLI `recover` 命令。
- `events.md` 哈希链与基础 session 结构校验。
- `session.state.md`、`planner/blockers.md`、`audit/replay.md` 的恢复与修复。
- 首版 checkpoint 追加写：阶段开始、阶段完成、阻塞等待、恢复启动。
- 对 `orchestrator` / `role-runtime` 的续跑驱动复用。
- recovery 模块对应的单元测试、命令测试、E2E、文档与时间线更新。

### 3.2 001 范围外

- 不做跨主机、跨进程分布式恢复。
- 不实现完整的“从零只靠 WAL 重建整个 planner/roles 子树”。
- 不自动恢复 `WAITING_HUMAN`，这类会话仍由 `resume` 处理。
- 不自动恢复业务含义上的 `FAILED` 终态；失败会话只做校验与报告，不擅自改写为可执行态。
- 不引入 daemon、数据库、后台任务或 Web 恢复面板。
- 不补齐 `runtime.recovery` / `runtime.checkpoint` 的新配置字段；`001` 固定采用严格模式。

---

## 4. 方案取舍

围绕 `recovery 001`，当前有三条可选路径：

### 方案 A：只加一个 `recover` 命令，内部直接再跑一遍 orchestrator / role-runtime

代码最少，但它本质只是“再 tick 一次”，没有 checkpoint、没有修复、没有 replay 报告，无法真正成为 recovery 模块。

### 方案 B：增加 `internal/recovery` 服务层，负责校验、checkpoint、修复与续跑

CLI 只负责解析参数，真正的恢复判断、状态修复、checkpoint 追加与 replay 审计统一放进服务层。这条路径最符合当前仓库 “cmd -> internal service -> Markdown 协议” 的分层方式。

### 方案 C：直接实现完整 WAL replay 引擎，完全忽略现有 `master_schedule.md` / `roles/*`

理论上最纯粹，但对首版来说明显过重。当前 orchestrator / role-runtime 已有大量现成状态文件，完全绕开它们重建一套引擎，会把 `001` 推成过度设计。

### 结论

`001` 明确选择 **方案 B**。

原因如下：

1. 当前已经有 `orchestrator`、`role-runtime`、`human-gate` 的稳定入口，recovery 更适合做“修复 + 续跑”的组合门面。
2. 增量 checkpoint 可以先做成保守快照，不必一次性上完整回放引擎。
3. 后续 `tmux` / `web` / `daemon` 若要接 recovery，也能直接复用同一服务层。

---

## 5. 核心用户故事

### 5.1 命令行用户

- 作为用户，我希望 session 已分发但进程退出后，可以通过正式命令把工作流继续推完，而不是重新 `run`。
- 作为用户，我希望恢复命令能明确告诉我“做了哪些检查、修了哪些文件、是否真的继续执行了”。
- 作为用户，我希望遇到 `WAITING_HUMAN` 时 recovery 不要擅自越权继续，而是明确提示我去走 `resume`。

### 5.2 下游模块

- 作为 `session`，我希望 recovery 能复用 `effective_config.yaml` 与 checkpoint 目录，不重新发明配置来源。
- 作为 `orchestrator`，我希望 recovery 先修复 `session.state.md` / `blockers.md` 再继续 tick，避免拿脏状态推进。
- 作为 `role-runtime`，我希望 recovery 只在已有 `context.md` / `inbox.md` 协议上继续执行，不绕过我已有的回合文件协议。

### 5.3 审计与排障

- 作为排障者，我希望通过 `audit/replay.md` 就能看到本次恢复读了哪个 checkpoint、校验了哪些文件、有没有继续运行。
- 作为复盘者，我希望 `state/checkpoints/` 至少能看到“初始化”“阶段开始”“阶段完成/阻塞/恢复”这些首版锚点。

---

## 6. 首版命令与协议

### 6.1 `recover`

`recover` 是 `recovery 001` 的正式 CLI 入口：

```bash
openoctopus recover --session <session_id|session_dir> [--format text|json]
```

执行顺序固定如下：

1. 解析 `--session` 到真实 session 目录。
2. 校验关键文件是否存在：
   - `metadata.md`
   - `session.state.md`（允许缺失，由 recovery 修复）
   - `state/effective_config.yaml`
   - `bus/events.md`
   - `planner/master_schedule.md`
3. 校验 `events.md` 哈希链合法。
4. 读取最新 checkpoint（如果只有 `0000-init.md` 也视为合法）。
5. 结合 `master_schedule.md`、`interrupts.md`、`conclusion.md` 推导恢复视图。
6. 修复最小必要文件：
   - `session.state.md`
   - `planner/blockers.md`
   - `audit/replay.md`
7. 如果恢复后状态属于 `INITIAL` / `READY` / `RUNNING`，则复用 orchestrator + role-runtime 继续推进。
8. 如果恢复后状态属于 `WAITING_HUMAN` / `FAILED` / `COMPLETED`，则只写 replay 报告并返回，不越权执行。

### 6.2 输出协议

`recover --format json` 的最小输出形状建议如下：

```json
{
  "ok": true,
  "command": "recover",
  "data": {
    "session_dir": ".octopus/sessions/sess_xxx",
    "recovered_status": "COMPLETED",
    "continued": true,
    "repaired_files": ["session.state.md", "planner/blockers.md", "audit/replay.md"],
    "checkpoint_ref": "state/checkpoints/0002-recover-start.md",
    "replay_ref": "audit/replay.md"
  }
}
```

文本模式至少要返回：

- session 路径
- 恢复后的 workflow status
- 是否继续执行
- replay 报告位置

---

## 7. 核心状态与恢复策略

### 7.1 可恢复状态判定

`recovery 001` 只自动继续以下状态：

| 状态 | 处理策略 |
| --- | --- |
| `INITIAL` | 允许继续 bootstrap/tick |
| `READY` | 允许继续调度 |
| `RUNNING` | 允许继续调度与 role tick |

以下状态只做校验和报告，不自动继续：

| 状态 | 处理策略 |
| --- | --- |
| `WAITING_HUMAN` | 返回 `needs_human_resume`，提示使用 `resume` |
| `FAILED` | 返回 `terminal_failed`，不擅自清错 |
| `COMPLETED` | 返回 `already_completed`，保持幂等 |

### 7.2 恢复视图推导顺序

首版恢复视图按以下顺序生成：

1. **事件层**：读取并校验 `bus/events.md`，确认 WAL 未损坏。
2. **快照层**：读取最新 checkpoint，拿到最后一个可信恢复锚点。
3. **当前态层**：读取 `planner/master_schedule.md`、`bus/interrupts.md`、角色 `conclusion.md` / `state.md`。
4. **归约层**：推导 workflow status、current stage、current role、blocker summary。
5. **修复层**：只重写当前态文件和 replay 报告，不重建整个 session 子树。

### 7.3 `session.state.md` 修复规则

首版修复遵循以下规则：

1. `WAITING_HUMAN` 优先级最高：只要任一 stage 为 `BLOCKED` 或存在未清理人工等待 blocker，就将 session 写为 `WAITING_HUMAN`。
2. `FAILED` 次高：只要任一 stage 为 `FAILED`，就写为 `FAILED`。
3. `COMPLETED`：当全部 stage 为 `COMPLETED` 时写为 `COMPLETED`。
4. `RUNNING`：当至少一个 stage 为 `DISPATCHED` 时写为 `RUNNING`。
5. 其他情况写为 `READY`。

`current_stage_id` 与 `current_role_id` 优先取第一个 `DISPATCHED` stage；若没有，则取最近一个 `BLOCKED` / `FAILED` stage；再没有则为空。

### 7.4 `planner/blockers.md` 修复规则

- 如果 workflow 为 `WAITING_HUMAN`，则从首个 `BLOCKED` stage 的结论摘要或 `interrupt-all` 原因中恢复 summary。
- 如果 workflow 不是 `WAITING_HUMAN`，则固定写回 `clear`。

---

## 8. Checkpoint 方案

### 8.1 命名规则

`001` 阶段延续 `0000-init.md` 作为初始化快照，后续 checkpoint 统一采用：

```text
state/checkpoints/{seq:04d}-{kind}.md
```

示例：

- `0001-stage-stage_a-dispatched.md`
- `0002-stage-stage_a-completed.md`
- `0003-session-waiting-human.md`
- `0004-recover-start.md`

### 8.2 落盘时机

首版只覆盖以下边界：

1. orchestrator 分发 stage 之后。
2. orchestrator 消费结论、把 stage 置为 `COMPLETED` / `BLOCKED` / `FAILED` 之后。
3. `interrupt-all` 把 session 拉到 `WAITING_HUMAN` 之后。
4. `resume` 清理人工等待并重新入队之后。
5. `recover` 正式开始恢复之前。

### 8.3 checkpoint 内容

每个 checkpoint 至少包含：

- `checkpoint_seq`
- `kind`
- `session_id`
- `workflow_status`
- `current_stage_id`
- `current_role_id`
- `schedule_version`
- `last_event`
- `source`
- `created_at`

并尽量引用：

- `planner/master_schedule.md`
- `session.state.md`
- `planner/blockers.md`

---

## 9. 与现有模块的分层关系

### 9.1 recovery 负责什么

- 校验 session 是否具备恢复前置条件。
- 提供 checkpoint 追加写能力。
- 归约恢复视图并修复当前态文件。
- 调用 orchestrator / role-runtime 继续推进。
- 写 replay 审计报告。

### 9.2 recovery 不负责什么

- 不重新实现 orchestrator 的图构建与 dispatch 逻辑。
- 不替代 role-runtime 的 turn 写入、结论解析和 interrupt 闸门。
- 不直接吸收人工输入，也不替代 human-gate 的 `resume` 语义。

---

## 10. 测试要求

`recovery 001` 至少覆盖以下测试层次：

### 10.1 单元测试

- checkpoint 渲染与序号递增。
- `events.md` 哈希链校验通过/失败。
- `session.state.md` / `blockers.md` 的恢复归约逻辑。

### 10.2 命令测试

- `recover` 成功恢复并输出稳定文本/JSON。
- `recover` 面对 `WAITING_HUMAN` 时返回成功但不继续执行。
- `recover` 面对损坏的 `events.md` 返回稳定失败。

### 10.3 E2E

- session 已分发但未继续执行，`recover` 能续跑到 `COMPLETED`。
- `session.state.md` 丢失后，`recover` 能修复并继续推进。
- `events.md` 哈希链被篡改后，`recover` 必须失败。

---

## 11. 通过标准

以下条件同时满足，才视为 `recovery 001` 达标：

1. `recover` 成为正式 CLI 命令，可稳定消费 session id / session dir。
2. `state/checkpoints/` 不再只有 `0000-init.md`，而是能稳定追加阶段边界快照。
3. 恢复过程会把 replay 审计写入 `audit/replay.md`。
4. 核心 E2E 可以稳定证明“进程退出后的 session 能继续完成”。
5. `WAITING_HUMAN` 会被明确拒绝自动恢复，不破坏 human-gate 边界。

