# Human Gate 模块 001 阶段 PRD

## 1. 目标

`human-gate 001` 的目标不是一次性做完完整的人审平台，而是先把 **CLI 触发人工介入 -> Markdown 留痕 -> 运行时暂停 -> 人工补充 -> 恢复续跑** 这条最小闭环跑通。

在当前仓库里，`event-bus 001` 已经提供了 `interrupts.md` 与中断事件协议，`orchestrator 001` 已经能在角色 `BLOCKED` 后把工作流置为 `WAITING_HUMAN`，`role-runtime 001` 也已经能在中断请求到达时写出 `INTERRUPTED` 状态。但系统仍缺一层统一的人工 gate 门面，导致以下问题没有被真正解决：

1. 用户还不能直接通过正式 CLI 写入中断请求。
2. 用户补充意见只能靠测试 harness 直接改 `planner/human_messages.md`。
3. `WAITING_HUMAN` 后没有正式恢复入口，阻塞阶段不能重新入队。
4. `ACKNOWLEDGED` 的 interrupt 还没有成为真正的“暂停闸门”。

所以 `001` 的职责非常明确：**把现有 event-bus / orchestrator / role-runtime 之间已经具备雏形的人工介入协议，收口成一个真实可操作的模块与 CLI 闭环。**

---

## 2. 范围

### 2.1 本期必须完成

`human-gate 001` 只落地 4 个首版入口：

1. `openoctopus interrupt --session <...> --role <role_id> --reason "..."`
2. `openoctopus interrupt-all --session <...> --reason "..."`
3. `openoctopus inject --session <...> [--role <role_id>] (--message "..." | --input ./note.md)`
4. `openoctopus resume --session <...> [--role <role_id>]`

并围绕这 4 个入口补齐以下协议能力：

- 正式 CLI 写 `bus/events.md` + `bus/interrupts.md`
- 正式 CLI 追加 `planner/human_messages.md`
- `ACKNOWLEDGED` 的 interrupt 在 clear 前必须阻止角色继续执行
- `resume` 能清理已确认 interrupt，并把 `WAITING_HUMAN` 的阻塞阶段重新入队
- `resume` 在恢复后直接复用已有 orchestrator / role-runtime 同步 loop，把会话继续推进

### 2.2 本期明确不做

以下能力仍然留给后续版本：

- 多人审批流、投票流、审批链
- Web 人工审批页面
- `reroute` 的任意阶段重排
- `reset-session` 的正式 CLI
- 复杂审批策略，比如“只允许某角色审批某阶段”
- 细粒度的人工 diff merge / patch 审批工作台

`001` 的原则是：**先把最关键的人类接管闭环稳定跑通，再继续扩展复杂协作。**

---

## 3. 方案取舍

围绕 `human-gate 001`，当前有三种实现路径：

### 方案 A：命令直接改 Markdown 文件

最省代码，但会把中断、注入、恢复的业务判断散落在多个 Cobra 命令里，后续很难复用，也不利于 E2E 和 recovery 继续接管。

### 方案 B：增加 `internal/humangate` 服务层

CLI 只负责参数解析，真正的 session 读写、中断申请、人工消息追加、阻塞恢复和 loop 驱动统一落到服务层。这样最符合当前仓库“CLI -> internal service -> 文件协议”的分层方式。

### 方案 C：直接让 orchestrator 吞掉 human-gate 职责

短期代码最少，但会把“主控调度”和“人工入口”混成一层。后续 `recovery`、`web`、`tmux` 如果也要调用人工 gate，会继续扩大 orchestrator 责任面。

### 结论

`001` 明确选择 **方案 B**。

原因很简单：

1. 当前仓库已经有 `event-bus`、`orchestrator`、`role-runtime` 三层边界，`human-gate` 应该成为它们之上的“人工协作门面”，而不是继续把逻辑挤进 `cmd/` 或 `orchestrator/`。
2. `resume` 天然是组合动作：清 interrupt、重排阻塞阶段、驱动 orchestrator tick、驱动 role-runtime tick。把这些组合步骤收进一个服务层，后续 CLI / Web / Recovery 都能复用。
3. 这样做能保证首版依旧简单：复用现有 Markdown 文件协议，不引入新数据库、不引入守护进程。

---

## 4. 首版命令与协议

### 4.1 `interrupt`

`interrupt` 只处理“定向暂停某个角色”这一件事：

1. 解析 `--session` 到目标 session 目录。
2. 通过 `event-bus` 获取 lease。
3. 写入 `INTERRUPT_REQUESTED` 事件，并更新 `bus/interrupts.md`。
4. 不直接执行 role-runtime，只返回“请求已受理”。

角色运行时在下一次 tick 时必须：

1. 读到 `REQUESTED` interrupt。
2. 把 `roles/{role_id}/state.md` 写成 `INTERRUPTED`。
3. 通过 `INTERRUPT_ACKNOWLEDGED` 把中断标记成 `ACKNOWLEDGED`。
4. 在 clear 前不允许继续执行 turn。

### 4.2 `interrupt-all`

`interrupt-all` 是 `interrupt` 的批量版，但它还要把会话立即拉到人工等待态：

1. 对当前 schedule 中仍未完成的角色逐个申请 interrupt。
2. 将 `planner/blockers.md` 改写为本次人工等待原因。
3. 将 `session.state.md` 改写为 `WAITING_HUMAN`。
4. 记录一条人类 gate 决策日志，便于后续审计。

`001` 不做复杂“只中断正在 RUNNING 的角色”优化；只要阶段仍未完成，就允许进入人工等待态。

### 4.3 `inject`

`inject` 的职责是把人工补充意见标准化写入 `planner/human_messages.md`。

消息块首版约定：

```md
## message: msg-000001
- message_id: msg-000001
- source: human-gate
- kind: inject
- target_role_id: agent_a
- created_at: 2026-03-08T00:00:00Z

### content
请继续推进，但先修正测试失败原因。
```

其中：

- `kind` / `target_role_id` 是 `human-gate 001` 新增的可选扩展字段。
- orchestrator 首版继续只消费 `message_id` / `source` / `created_at` / `content`，因此不会破坏现有解析器。
- `--message` 与 `--input` 二选一；`--input` 的内容按原文写入 `### content`。

### 4.4 `resume`

`resume` 是首版里最关键的组合动作。它必须同时处理两类“等待人工”来源：

#### A. 来自 interrupt 的等待

如果存在目标角色（或全部角色）的 `ACKNOWLEDGED` interrupt：

1. 将它们推进为 `CLEARED`
2. 保留事件链与 `interrupts.md` 当前态投影
3. 允许 role-runtime 下一次 tick 继续执行原任务

#### B. 来自 `BLOCKED` 结论的等待

如果 `planner/master_schedule.md` 中存在 `BLOCKED` 阶段：

1. 将其推进为 `RETRY_PENDING`
2. `attempt` 加一，确保下一次重新分发得到新的 `task_id`
3. 清理 `planner/blockers.md` 的阻塞摘要
4. 将 `session.state.md` 从 `WAITING_HUMAN` 拉回 `READY`

最后，`resume` 直接复用现有同步 loop：

1. 先执行一次 orchestrator tick，吸收最新 `human_messages.md`
2. 再驱动 role-runtime / orchestrator 的有界循环
3. 直到无进展或进入终态

这意味着在当前无 daemon 的前提下，用户可以通过 `inject` + `resume` 直接把会话继续跑下去。

---

## 5. 文件影响

`human-gate 001` 只在现有 session 协议内工作，不新建新的顶层目录。

### 5.1 直接写入的文件

- `bus/events.md`
- `bus/interrupts.md`
- `planner/human_messages.md`
- `planner/master_schedule.md`
- `planner/blockers.md`
- `planner/decision_log.md`
- `session.state.md`

### 5.2 间接驱动更新的文件

- `planner/requirement.snapshot.md`
- `roles/{role_id}/context.md`
- `roles/{role_id}/inbox.md`
- `roles/{role_id}/state.md`
- `roles/{role_id}/turns/*.md`
- `roles/{role_id}/conclusion.md`
- `roles/{role_id}/outbox.md`

---

## 6. 模块边界

| 模块 | `human-gate 001` 依赖它做什么 | 不在本模块做什么 |
| --- | --- | --- |
| `event-bus` | 申请 lease、追加事件、维护 interrupt 投影 | 自己手改 `bus/*.md` 原始协议 |
| `orchestrator` | 消费 `human_messages.md`、重新分发阶段、推进工作流状态 | 理解 CLI 参数和人工输入来源 |
| `role-runtime` | ACK 中断、在 clear 前阻止继续执行、恢复后继续执行 | 决定是否 requeue 阻塞阶段 |
| `cmd/openoctopus` | 暴露正式 CLI 命令 | 承担复杂业务判断 |

---

## 7. 验收标准

满足以下条件，视为 `human-gate 001` 完成：

1. `interrupt` 能稳定写出 `INTERRUPT_REQUESTED` 事件与 `interrupts.md` 投影。
2. `role-runtime` 在读到 `REQUESTED` 后会 ACK，并在 clear 前不产生新的 turn。
3. `inject` 能稳定把人工消息追加到 `planner/human_messages.md`，并支持 `--message` / `--input`。
4. `resume` 能清理 `ACKNOWLEDGED` interrupt，并继续推进被暂停的角色。
5. `resume` 能把 `BLOCKED` 阶段重新入队，重新分发后获得新的 `task_id`。
6. `interrupt-all` 后会话立刻进入 `WAITING_HUMAN`，并留下明确 blockers 原因。
7. 新增命令、单测、E2E、`make check` 全部通过。

---

## 8. 对比当前实现的关键差异

与当前仓库相比，`human-gate 001` 的关键变化是：

1. **从测试专用写文件** 升级为 **正式 CLI + internal service**。
2. **从“中断被 ACK 后仍可能继续执行”** 升级为 **ACK 即暂停，必须 clear 后才能恢复**。
3. **从“BLOCKED 只能停住”** 升级为 **人工补充后可正式 `resume` 重跑**。
4. **从“只能靠 harness 注入 human message”** 升级为 **正式 `inject` 命令落盘并被 orchestrator 消费**。
