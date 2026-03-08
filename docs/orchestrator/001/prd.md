# Orchestrator 模块 001 阶段 PRD

## 1. 阶段定位

`orchestrator 001` 的目标，不是马上把主 Agent 做成一个复杂常驻调度器，而是先把 **Master 单次编排闭环** 做实：基于已经落盘的 session 目录和 event-bus，总结当前需求、生成排程、分发首批任务、消费角色结论并推进到下一状态。

当前仓库里，`session 001` 已经能创建 `planner/` 占位文件，`event-bus 001` 也正在把 `bus/*.md` 升级为真实协议；但主控编排本身仍然缺席：`requirement.snapshot.md` 还没有真实内容，`master_schedule.md` 还不是可推进的排程，`roles/{role_id}/context.md` 和 `inbox.md` 也还没有任何分发能力。

`orchestrator 001` 要解决的核心问题是：把现有 session 骨架和 event-bus 能力，升级为后续 `role-runtime`、`human-gate`、`recovery` 都能围绕其工作的 **首版 Master 编排协议**。

这意味着本阶段要从“目录已存在”升级到“主控能稳定 bootstrap planner 文件、能按 YAML 阶段图生成首版排程、能写角色任务包、能消费结论并更新全局当前态”。

`001` 的目标不是把 orchestrator 做复杂，而是先把 **单 session、显式状态机、单次 tick 可解释** 这条最小主链路跑通，避免后续模块各自发明自己的调度协议。

## 2. 本阶段要解决的问题

### 2.1 当前问题

当前仓库已经具备配置加载、session 目录初始化和 event-bus 基座，但 orchestrator 还存在以下缺口：

1. **planner 文件仍是占位文档**：`requirement.snapshot.md`、`master_schedule.md`、`global_progress.md`、`blockers.md` 还没有真实业务结构。
2. **缺少完整 planner 视图**：`task_board.md`、`task_graph.mmd`、`dispatch_log.md`、`decision_log.md` 还未建立，主控决策链无法解释。
3. **没有阶段图落地能力**：虽然 YAML 已有 `stages` 和 `transitions`，但还没有真正把它们渲染成可推进的主控排程。
4. **没有人工输入吸收协议**：`planner/human_messages.md` 还没有消费游标、需求快照版本和重编排标记。
5. **没有角色任务分发协议**：`roles/` 根目录虽已存在，但 `context.md`、`inbox.md` 还没有被主控生成。
6. **没有结论收口与重试规则**：`conclusion.md` 还没有被主控消费，因此 `retry` / `blocked` / `completed` 都无法推进到全局状态。
7. **没有黑盒验证**：当前没有任何命令级或 E2E 证明 `run` 已经从“建骨架”升级到“完成首轮主控编排”。

### 2.2 001 阶段目标

`orchestrator 001` 的目标是建立一个简单但稳定的主控编排首版协议：

1. 将 `planner/` 占位文件升级为真实业务文档，并补齐缺失的 planner 文件。
2. 基于 `state/effective_config.yaml` 生成首版阶段图、主排程和任务看板。
3. 定义 `human_messages.md` 的消费规则，以及 `requirement.snapshot.md` 的版本与游标协议。
4. 为每个将被分发的角色生成 `roles/{role_id}/context.md` 和 `inbox.md`。
5. 消费角色 `conclusion.md`，支持 `SUCCESS / NEEDS_RETRY / BLOCKED / FAILED` 四类首版结果收口。
6. 将主控写操作统一收口到 `internal/orchestrator`，通过 event-bus 锁和事件保障顺序与可追溯性。
7. 在 `run` 成功创建 session 且 bootstrap event-bus 后，立即完成 orchestrator bootstrap 与首轮 tick。

## 3. 范围定义

### 3.1 001 范围内

- 基于已创建好的 session 目录 bootstrap orchestrator。
- 将 `planner/requirement.snapshot.md`、`planner/master_schedule.md`、`planner/global_progress.md`、`planner/blockers.md` 升级为真实协议文档。
- 新增并维护以下 planner 文件：
  - `planner/task_board.md`
  - `planner/task_graph.mmd`
  - `planner/dispatch_log.md`
  - `planner/decision_log.md`
- 读取 `state/effective_config.yaml`，构建首版阶段图与 entry stage 集合。
- 定义并消费 `planner/human_messages.md` 的首版消息块格式。
- 在首版中支持 **线性阶段流转** 与 **多入口并行起步**，并用 `runtime.scheduler.max_parallel_roles` 控制首批分发数。
- 为 ready 阶段生成 `task_id`，按角色写入 `roles/{role_id}/context.md` 与 `roles/{role_id}/inbox.md`。
- 读取 `roles/{role_id}/conclusion.md`，执行完成、重试、阻塞、失败四类状态推进。
- 更新 `session.state.md` 中与主控相关的当前态字段，例如 `status`、`current_stage_id`、`current_role_id`、`updated_at`、`human_message_cursor`。
- 通过 event-bus 追加 orchestrator 相关事件，并提交消费者位点 `orchestrator/master`。
- 在 `run` 成功路径中完成 orchestrator bootstrap + 首轮调度。

### 3.2 001 范围外

- 不负责 `role-runtime` 的真实执行壳层、`turns/*.md`、`state.md`、`outbox.md`、`heartbeat.md` 写入。
- 不负责 `human-gate` 的 `interrupt` / `interrupt-all` / `approval` / `reroute` CLI 入口实现。
- 不负责 `recovery` 的完整事件回放与崩溃恢复，只为它提供稳定 planner 当前态和决策记录。
- 不负责复杂条件表达式求值、`aggregate` / `role_aggregate` 聚合、join stage、多分支收敛；这些留到 `orchestrator 002`。
- 不负责常驻 watch daemon、tmux 面板、HTTP/SSE 可视化。
- 不引入数据库、外部队列、后台守护进程或额外调度框架。

## 4. 核心用户故事

### 4.1 命令行用户

- 作为用户，我希望 `run` 成功后，session 目录里不仅有 bus 文件，也已经有真实的 `master_schedule.md` 和首批角色任务包，方便我立刻看到系统下一步准备做什么。
- 作为用户，我希望主控阻塞时会把原因写进 `planner/blockers.md`，而不是只在终端里给我一句无法追溯的报错。

### 4.2 下游模块

- 作为 `role-runtime`，我希望 `context.md` 和 `inbox.md` 是稳定、可重读、带版本号的任务包，而不是临时字符串拼接。
- 作为 `human-gate`，我希望人工输入写进 `human_messages.md` 后，主控能明确知道自己已经消费到了哪一条，而不是反复全量扫描并重复吸收。
- 作为 `recovery`，我希望崩溃后可以从 `master_schedule.md`、`decision_log.md`、`dispatch_log.md` 看出上一次主控到底推进到了哪一步。

### 4.3 审计与排障

- 作为排障者，我希望知道一次 stage 失败后，主控到底选择了重试、阻塞还是终止。
- 作为复盘者，我希望通过 `decision_log.md`、`dispatch_log.md` 与 bus 事件能清楚看到“为什么发这个任务、为什么走这条路径、为什么最终结束”。

## 5. 核心方案

### 5.1 启动入口与模块接口

`orchestrator 001` 的输入，不再直接依赖原始 `--config` 路径，而是以 session 中已落盘的 `state/effective_config.yaml` 为主配置来源。这样后续 recovery、重跑和 E2E harness 都能基于 session 自身完成一致读取。

建议对外提供一个显式服务入口，例如：

- `NewEngine(sessionDir string)`
- `Bootstrap() error`
- `Tick() (TickResult, error)`

执行顺序固定为：

1. `config 001` 生成有效配置
2. `session 001` 创建 session 骨架
3. `event-bus 001` bootstrap 总线
4. `orchestrator 001` bootstrap planner 文件
5. `orchestrator 001` 执行首轮 `Tick()`

### 5.2 planner 文件集与写入边界

`orchestrator 001` 是以下文件的唯一业务写入方：

| 文件 | 职责 | 写入策略 |
| --- | --- | --- |
| `planner/requirement.snapshot.md` | 当前需求快照与人工输入消费位点 | 原子替换 |
| `planner/master_schedule.md` | 主控唯一排程真相 | 原子替换 |
| `planner/task_board.md` | 面向阅读的状态投影 | 原子替换 |
| `planner/task_graph.mmd` | 阶段依赖图 | 原子替换 |
| `planner/global_progress.md` | 全局进度摘要 | 原子替换 |
| `planner/blockers.md` | 阻塞与人工提示 | 原子替换 |
| `planner/dispatch_log.md` | 任务分发记录 | 追加写 |
| `planner/decision_log.md` | 主控决策记录 | 追加写 |

边界约束如下：

1. `human_messages.md` 是 append-only 输入文件，orchestrator 只读，不回写。
2. `roles/{role_id}/context.md` 和 `roles/{role_id}/inbox.md` 由 orchestrator 写；`state.md` / `outbox.md` / `conclusion.md` / `turns/*.md` 属于 `role-runtime`。
3. 任何 planner 当前态文件覆盖写都必须采用 `*.tmp + rename`。
4. 业务模块不得绕过 orchestrator 直接修改 `master_schedule.md` 或 `task_board.md`。

### 5.3 `human_messages.md` 协议

`session 001` 只创建占位文件，`orchestrator 001` 需要定义首版可消费格式。建议按消息块追加：

```text
# Human Messages

## message: msg-000001
- message_id: msg-000001
- source: user
- created_at: 2026-03-08T09:00:00Z

### content
请继续设计 orchestrator 001 的文档。
```

首版规则：

1. `message_id` 由写入方保证单调递增且唯一，推荐 `msg-000001` 形式。
2. orchestrator 不修改消息块本身，消费位点通过 `requirement.snapshot.md` 与 `session.state.md` 记录。
3. 占位文档或空文档都视为“0 条消息”，不能报错。
4. 首版只支持纯文本 `content`，不引入多附件或结构化表单。

### 5.4 `requirement.snapshot.md` 协议

`requirement.snapshot.md` 用于表达“主控当前理解到的需求输入快照”，首版至少包含：

- `snapshot_version`
- `workflow_id`
- `workflow_name`
- `human_message_cursor`
- `source_message_count`
- `planner_status`
- `updated_at`

推荐正文最少包含三段：

1. `## workflow_summary`：从 `effective_config.yaml` 直接抽取的工作流元信息、阶段数、角色数。
2. `## latest_messages`：最近一次吸收的新人工输入原文拼接结果。
3. `## current_dispatch_brief`：当前准备投递或刚刚投递给角色的任务摘要。

首版不引入额外 master LLM 来总结用户输入，而是采用**确定性文本合并**：保留配置摘要、按顺序拼接新增人工消息，并显式记录游标。这样更易验证，也更利于 recovery。

### 5.5 `master_schedule.md`、`task_board.md` 与 `task_graph.mmd`

`master_schedule.md` 是唯一排程真相，至少需要包含：

- `schedule_version`
- `workflow_status`
- `active_dispatch_count`
- `updated_at`

每个阶段块至少记录：

- `stage_id`
- `stage_name`
- `role_id`
- `status`
- `attempt`
- `last_task_id`
- `last_conclusion_ref`
- `next_stage_id`
- `updated_at`

首版 stage 状态集合建议固定为：

- `PENDING`
- `READY`
- `DISPATCHED`
- `COMPLETED`
- `RETRY_PENDING`
- `BLOCKED`
- `FAILED`

`task_board.md` 是阅读投影，不新增新事实，只把 `master_schedule.md` 投影为 `Todo / Doing / Done / Blocked` 四栏。

`task_graph.mmd` 则把当前支持的 `to` 关系渲染为 Mermaid 图，帮助人工快速核对阶段流转是否符合配置。

### 5.6 `context.md`、`inbox.md` 与 `conclusion.md` 协议边界

`orchestrator 001` 负责写入以下角色任务包：

#### A. `roles/{role_id}/context.md`

至少包含：

- `context_version`
- `task_id`
- `stage_id`
- `role_id`
- `requirement_snapshot_ref`
- `master_schedule_ref`
- `attempt`
- `updated_at`

正文应包含：

- 当前阶段目标
- 输入 artifact / 输出 artifact 约束
- 需要遵守的只读或禁止写规则摘要
- 最近一次人工输入摘要

#### B. `roles/{role_id}/inbox.md`

至少包含：

- `inbox_version`
- `task_id`
- `stage_id`
- `role_id`
- `status: DISPATCHED`
- `dispatch_event_id`
- `context_version`
- `updated_at`

#### C. `roles/{role_id}/conclusion.md`

虽然该文件由 `role-runtime` 生产，但 `orchestrator 001` 必须定义首版消费契约，至少需要：

- `role_id`
- `stage_id`
- `task_id`
- `status: SUCCESS | NEEDS_RETRY | BLOCKED | FAILED`
- `summary`
- `output_refs`
- `updated_at`

主控只基于 `conclusion.md` 做下一步决策，不依赖临时会话上下文。

### 5.7 Tick 主循环

`orchestrator 001` 不是常驻后台，而是一次调用执行一轮确定性 `Tick()`。建议每轮执行顺序固定如下：

1. 获取 `event-bus` 租约锁，holder 固定为 `orchestrator/master`
2. 读取 `effective_config.yaml`
3. bootstrap 或刷新 planner 文件
4. 吸收 `human_messages.md` 的新增消息，更新 `requirement.snapshot.md`
5. 读取 `master_schedule.md` 与各角色 `conclusion.md`
6. 对已完成阶段执行状态推进：完成、重试、阻塞、失败
7. 选出新的 ready 阶段，按 `max_parallel_roles` 分发任务
8. 更新 `session.state.md`、`global_progress.md`、`blockers.md`
9. 追加 `dispatch_log.md`、`decision_log.md` 和 bus 事件
10. 提交 `orchestrator/master` offset 并释放锁

如果本轮没有任何实质进展，应仍然更新 `global_progress.md` 中的无进展计数，以便后续 deadlock guard 和人工介入模块复用。

### 5.8 阶段图与首版流转约束

为保证 `001` 可稳定实现，阶段图约束收紧如下：

1. 首版只支持 `transition.to` 的显式直接流转。
2. `on_true`、`on_false`、`condition.expr`、`aggregate`、`role_aggregate` 在 `001` 中不解释；若配置使用这些能力，应在 orchestrator bootstrap 时显式报“不支持”。
3. 首版允许多个 entry stage，但不支持多前驱 join stage。
4. 同一角色同一时刻只允许一个 active task。
5. 每轮最大新分发数不得超过 `runtime.scheduler.max_parallel_roles`。

这样做的原因是：当前仓库的配置校验已经能保证引用合法，但还没有提供完整条件求值器；`001` 先把线性主链路和多入口起步做稳，后续再演进复杂路由。

### 5.9 结果收口与守卫策略

`orchestrator 001` 首版仅收敛四类结论：

- `SUCCESS`：标记当前 stage 为 `COMPLETED`，若 `next_stage_id == __END__` 则工作流结束，否则推进下一个 stage 为 `READY`
- `NEEDS_RETRY`：若未超过 `policies.retry.max_retry_per_stage`，则置为 `RETRY_PENDING` 并重新生成任务包；否则置为 `FAILED`
- `BLOCKED`：将 session 状态置为 `WAITING_HUMAN`，写 `blockers.md` 并停止继续分发
- `FAILED`：将 stage 置为 `FAILED`，会话进入 `FAILED`

守卫策略：

1. 单 stage 重试次数受 `policies.retry.max_retry_per_stage` 控制。
2. 单任务无效循环受 `policies.loop_guard.max_rounds_per_task` 控制。
3. session 级无进展轮次受 `runtime.master_watch.max_no_progress_rounds` 控制。
4. `runtime.master_watch.auto_interrupt_all_on_deadlock` 在 `001` 中只负责落 blocker 与事件，不负责真正触发完整 `interrupt-all` 执行链。

### 5.10 event-bus 交互协议

`orchestrator 001` 不是总线协议设计者，但必须统一复用 `event-bus 001`：

- 获取租约：`AcquireLock("orchestrator/master", ttl)`
- 写入事件：`Append(...)`
- 提交位点：`CommitOffset(consumer_id="orchestrator/master")`

首版建议至少追加以下事件类型：

- `ORCHESTRATOR_BOOTSTRAPPED`
- `REQUIREMENT_SNAPSHOT_UPDATED`
- `STAGE_READY`
- `TASK_DISPATCHED`
- `STAGE_COMPLETED`
- `STAGE_RETRY_SCHEDULED`
- `STAGE_BLOCKED`
- `WORKFLOW_COMPLETED`
- `WORKFLOW_FAILED`

边界原则保持不变：**orchestrator 可以表达决策，但不能自定义 bus 文件格式。**

### 5.11 模块边界

| 模块 | 负责内容 | 不负责内容 |
| --- | --- | --- |
| `session` | 创建 `planner/` 根目录与占位文件、提供 session 根路径与有效配置快照 | 主控真实排程、角色任务分发 |
| `event-bus` | 锁、事件 WAL、offset、interrupt 协议 | 排程算法、角色上下文格式 |
| `orchestrator` | planner 业务文件、`context.md` / `inbox.md`、结论收口、全局当前态推进 | 角色执行壳层、回合文件写入 |
| `role-runtime` | `state.md`、`outbox.md`、`conclusion.md`、`turns/*.md`、`heartbeat.md` | 主控排程与跨角色决策 |
| `human-gate` | 向 `human_messages.md` 追加人工输入，未来接入审批与打断 | 直接改写 `master_schedule.md` |
| `recovery` | 基于 bus + planner 文件重建主控状态 | 首次创建排程与任务包 |

## 6. 分阶段交付物

### 6.1 必交付物

1. `docs/orchestrator/001/prd.md`
2. `docs/orchestrator/001/e2e.md`
3. `docs/orchestrator/001/plans/` 下的拆分任务文档
4. `internal/orchestrator` 的首版服务实现
5. `cmd/openoctopus/run.go` 对 orchestrator bootstrap + 首轮 tick 的接入
6. 至少一组单测 / 命令级测试，验证 `run` 后 planner 文件和首批任务包已落盘

### 6.2 建议代码落位

建议保持轻量分文件，不要把主控逻辑塞进一个大文件：

- `internal/orchestrator/types.go`：基础模型与状态枚举
- `internal/orchestrator/errors.go`：稳定错误类型
- `internal/orchestrator/bootstrap.go`：planner bootstrap 与 placeholder 升级
- `internal/orchestrator/graph.go`：阶段图构建与首版约束校验
- `internal/orchestrator/snapshot.go`：human message 消费与 requirement snapshot 合并
- `internal/orchestrator/dispatch.go`：任务分发、context / inbox 渲染
- `internal/orchestrator/engine.go`：`Tick()` 主循环与决策收口
- `internal/orchestrator/render.go`：Markdown 渲染工具
- `internal/orchestrator/*_test.go`：模块测试

CLI 侧保持最小变更：

- `cmd/openoctopus/run.go`
- `cmd/openoctopus/command_test.go`

## 7. 验收标准

以下条件同时满足，才视为 `orchestrator 001` 达标：

1. `run` 在合法配置下创建 session 并 bootstrap event-bus 后，能继续生成真实 planner 文件，而不是停留在 placeholder。
2. `master_schedule.md`、`task_board.md`、`task_graph.mmd`、`global_progress.md`、`decision_log.md`、`dispatch_log.md` 都存在且内容一致。
3. 至少一个 ready stage 会被分发，并写出对应角色的 `context.md` 与 `inbox.md`。
4. `conclusion.md` 写入 `SUCCESS / NEEDS_RETRY / BLOCKED / FAILED` 时，主控能稳定推进到正确状态。
5. orchestrator 通过 event-bus 租约与 offset 工作，不直接绕过总线做无锁调度。
6. `session.state.md` 与 `master_schedule.md` 的当前状态保持一致，可解释当前运行位置。
7. `make check` 能覆盖 orchestrator 新增测试，不破坏已有 `config`、`session`、`event-bus` 基线。

## 8. 依赖关系与风险

### 8.1 上游依赖

- `config 001`：提供合法的阶段、角色和基础策略配置
- `session 001`：提供 `planner/`、`roles/`、`state/effective_config.yaml` 等基础骨架
- `event-bus 001`：提供锁、事件写入和 offset 提交能力

### 8.2 下游影响

- `role-runtime 001` 将直接复用 `context.md` / `inbox.md` 协议与任务版本字段
- `human-gate 001` 将直接复用 `human_messages.md` 消息块格式与 `requirement.snapshot.md` 游标
- `recovery 001` 将直接复用 `master_schedule.md`、`decision_log.md` 和 `dispatch_log.md` 做状态重建

### 8.3 主要风险

1. **主控协议与角色协议边界易混淆**：若不提前明确 `context/inbox` 属于 orchestrator、`state/outbox/conclusion` 属于 role-runtime，后续实现会互相覆盖。
2. **阶段图能力过早扩张**：如果 `001` 试图一次支持条件表达式、聚合和 join，会显著拉高实现风险。
3. **当前态投影过多**：`master_schedule.md`、`task_board.md`、`global_progress.md` 都是投影文件，必须坚持“唯一事实源 + 其余只读投影”的思路，避免互相不一致。
4. **人机输入消费重复**：如果没有稳定的 `human_message_cursor`，主控容易把旧输入重复吸收，导致虚假重编排。
5. **无 role-runtime 场景下难以验证闭环**：需要测试专用 harness 人工写 `conclusion.md`，否则 orchestrator 很难在当前阶段做真实黑盒验证。

## 9. 与模块总纲的关系

- `docs/orchestrator/prd.md` 是模块总纲，定义“主 Agent 负责任务拆分、阶段流转、回路控制与终态判定”的大边界。
- 本文是该总纲的首个版本化落地稿，把“编排核心”细化为“首版到底有哪些 planner 文件、如何吸收人工输入、如何写角色任务包、如何消费结论、如何完成一次确定性 tick”。
- 后续如果进入 `orchestrator 002`，必须在新版本目录中继续演进，不能直接覆盖本目录文档。
