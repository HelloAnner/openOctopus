# Role Runtime 模块 001 阶段 PRD

## 1. 阶段定位

`orchestrator 001` 已经能把 `roles/{role_id}/context.md` 与 `inbox.md` 分发出来，并在后续 tick 中消费 `conclusion.md`；但当前链路仍停在“主控能发任务、测试用 harness 能伪造结论”的阶段，真正的角色执行壳层还不存在。

`role-runtime 001` 要解决的核心问题，不是马上做成常驻 watcher、tmux 多 pane 或复杂流式代理，而是先把 **单角色被动执行闭环** 做实：稳定读取 `context.md` / `inbox.md`、生成回合输入、调用执行器、落盘 `turns/*.md`、写 `state.md` / `outbox.md` / `conclusion.md` / `heartbeat.md`，并把关键信号同步回 event-bus。

本阶段的重点是让角色运行时从“概念职责”变成“可解释协议”。也就是说，后续无论是 `human-gate` 的打断、人类注入，还是 `recovery` 的恢复、`artifact` 的产物追踪，都不再依赖某个 CLI 进程的临时内存态，而是围绕 `roles/{role_id}/` 下的 Markdown 文件协作。

首版依然坚持第一性原理：**优先做单次 tick + 有界运行闭环，不做常驻守护**。这样既能和当前 Go CLI 原型保持一致，也能避免在协议还未稳定前过早引入复杂并发模型。

## 2. 本阶段要解决的问题

### 2.1 当前问题

当前仓库已经具备配置、session、event-bus、orchestrator 的首版能力，但角色运行时仍存在以下缺口：

1. **角色目录协议缺失**：`roles/` 根目录已经存在，`context.md` / `inbox.md` 也会被主控写出，但 `state.md`、`outbox.md`、`heartbeat.md`、`session.reset.md`、`turns/*.md` 还没有稳定协议。
2. **没有真实执行器壳层**：目前没有任何模块负责根据 `llm_profiles` 调起 `codex` CLI，也没有统一的 deterministic executor 供测试和守卫场景使用。
3. **没有回合级落盘链路**：角色每次执行前读了什么、执行时拿到什么、执行后返回什么，还没有被固化到 `turns/NNNN-input.md` / `turns/NNNN-output.md`。
4. **没有稳定结论产出协议**：虽然 orchestrator 已经能消费 `conclusion.md`，但真实 runtime 还没有定义如何从执行结果渲染出兼容 orchestrator 的结论文件。
5. **没有打断、位点与幂等规则**：event-bus 已经支持 offsets 与 interrupts，但 role-runtime 还没有消费位点、避免重复执行、按安全边界响应打断的能力。
6. **没有真实黑盒验证**：当前 orchestrator E2E 仍靠 synthetic `conclusion.md` 推进，无法证明真实角色执行链路能闭环，更无法证明真实 `codex` 环境可用。

### 2.2 001 阶段目标

`role-runtime 001` 的目标，是把角色执行器收敛成一个简单但稳定的首版协议层：

1. 定义并创建 `roles/{role_id}/` 下的首版运行时文件协议。
2. 提供 `TickRole` / `TickAll` 这类显式角色执行入口，完成“扫描 -> 执行 -> 落盘 -> 回执”的单次闭环。
3. 基于 `context.md`、`inbox.md`、`state/effective_config.yaml`、event-bus offsets / interrupts 生成可重放的回合输入。
4. 首版支持两类执行器：真实 `codex` CLI 执行器，以及用于测试和守卫场景的 deterministic executor。
5. 将执行结果稳定渲染为 `turns/*.md`、`outbox.md`、`conclusion.md`、`heartbeat.md`、`state.md`，并保证与 `orchestrator 001` 的结论消费契约兼容。
6. 在 `run` 主链路中补齐一个**有界同步角色运行循环**，形成“orchestrator 分发 -> role-runtime 执行 -> orchestrator 收口”的最小闭环。

## 3. 范围定义

### 3.1 001 范围内

- 新增 `internal/roleruntime` 首版服务，负责角色 bootstrap、单次 tick、执行器调用和结果落盘。
- 定义并维护以下角色目录文件：
  - `roles/{role_id}/state.md`
  - `roles/{role_id}/session.reset.md`
  - `roles/{role_id}/outbox.md`
  - `roles/{role_id}/conclusion.md`
  - `roles/{role_id}/heartbeat.md`
  - `roles/{role_id}/events.md`
  - `roles/{role_id}/turns/*.md`
- 消费 orchestrator 写出的 `context.md` / `inbox.md`，并把它们当作只读输入，不直接改写。
- 通过 event-bus 的 `ListAfter`、`CommitOffset`、`ReadInterrupts` 能力识别新分发、打断与去重条件。
- 生成标准化 `turns/NNNN-input.md`，并记录本轮读取顺序、上下文版本、任务版本和约束信息。
- 执行器抽象首版支持：
  - `provider: codex` + `mode: cli`
  - `provider: deterministic` + `mode: cli`
- 生成与 orchestrator 兼容的 `conclusion.md`，支持 `SUCCESS / NEEDS_RETRY / BLOCKED / FAILED` 四类状态。
- 更新角色级 `state.md`、`heartbeat.md`、`outbox.md`、`events.md`，并把关键节点同步写入 event-bus。
- 在 `run` 中接入有界同步运行循环：串行执行当前 ready/dispatched 角色，直到工作流进入终态、等待人工或检测到无进展。

### 3.2 001 范围外

- 不实现 tmux pane 常驻角色会话，也不实现 watcher daemon。
- 不实现 `claude_code` 的真实 CLI 适配器；该适配器留到 `role-runtime 002`。
- 不实现 token 级流式转发、SSE 输出、实时前端可视化。
- 不实现多角色真正并发的子进程调度；`001` 只保证串行有界执行，真正并行留给 tmux / runtime 002。
- 不负责 artifact 版本索引、产物血缘和只读文件锁明细；首版只允许在 `conclusion.md` / `outbox.md` 中记录 `output_refs`。
- 不实现用户态 `reset-session` / `interrupt-all` CLI 命令，只定义角色侧消费与响应协议。
- 不实现恢复重放逻辑本身，只为 `recovery 001` 提供稳定可重放文件。

## 4. 核心用户故事

### 4.1 命令行用户

- 作为用户，我希望 `run` 不只是把任务包写进 `roles/` 目录，而是真的能让角色执行至少一轮，并生成可读的回合文件和结论文件。
- 作为用户，我希望角色执行失败、阻塞或被打断时，有明确的 `state.md`、`heartbeat.md` 和 `conclusion.md` 可以追查，而不是只看到一个进程报错。

### 4.2 orchestrator 与后续模块

- 作为 `orchestrator`，我希望 `conclusion.md` 来自真实执行器而不是测试桩，且字段稳定、可重复消费。
- 作为 `human-gate`，我希望角色对 interrupt 的响应有清晰安全边界和回执文件，后续 CLI 可以直接接入，而不用改角色内核。
- 作为 `recovery`，我希望 `turns/*.md`、`state.md`、`session.reset.md`、`events.md` 足够表达“这个角色上一次到底做到哪里”。

### 4.3 E2E 与宿主机真实环境

- 作为 E2E，我希望至少有一个场景真的调用宿主机 `codex` CLI，证明 `~/.codex/` 配置可以被 role-runtime 首版链路真实复用。
- 作为测试作者，我也希望有 deterministic executor，保证 retry / blocked / interrupt / reset 这类协议测试不依赖大模型不稳定输出。

## 5. 核心方案

### 5.1 首版执行模型取舍

围绕角色运行时，首版存在三种可选实现路径：

1. **常驻 watcher + tmux pane**：最贴近产品远景，但需要文件监听、进程托管、终端 UI 和复杂并发控制，明显超出当前仓库成熟度。
2. **单次 tick + 有界同步运行循环**：以显式 `TickRole` / `TickAll` 作为内核，再由 `run` 调用一个有限轮数的 runtime loop。这条路径最贴合当前 Go CLI 架构，也是 `001` 的推荐方案。
3. **把执行器逻辑直接塞进 orchestrator**：短期看实现最少，但会把 `context/inbox` 与 `state/turns/conclusion` 的边界打碎，后续 `human-gate`、`recovery` 都会变难。

`001` 明确选择方案 2。原则很简单：**orchestrator 负责决定“谁该做什么”，role-runtime 负责决定“角色这一轮如何执行并如何落盘”**。二者通过 Markdown 文件与 bus 事件解耦，而不是共享内存或相互调用私有状态。

### 5.2 角色目录与文件归属

`role-runtime 001` 采用“主控只写任务包，角色只写执行结果”的边界：

| 文件 | 生产者 | 用途 |
| --- | --- | --- |
| `context.md` | `orchestrator` | 当前任务上下文、角色 system prompt、任务版本 |
| `inbox.md` | `orchestrator` | 当前待执行任务包与分发事件引用 |
| `state.md` | `role-runtime` | 当前角色最新运行态 |
| `session.reset.md` | `role-runtime` / 后续 `human-gate` | 角色会话代际与重置请求/执行结果 |
| `outbox.md` | `role-runtime` | 角色回执摘要，供主控/恢复读取 |
| `conclusion.md` | `role-runtime` | 与 orchestrator 对接的阶段结论 |
| `heartbeat.md` | `role-runtime` | 当前存活探测与超时判断依据 |
| `events.md` | `role-runtime` | 角色本地追加式审计日志 |
| `turns/*.md` | `role-runtime` | 每轮输入/输出与执行原始记录 |

`001` 中，角色目录建议在**第一次角色 tick** 或 **首次准备执行任务**时统一 bootstrap；不要等文件缺失时临时补一半，以免产生不一致状态。

### 5.3 单次角色 tick 流程

`TickRole(sessionDir, roleID)` 的建议流程如下：

1. 读取 `state/effective_config.yaml`，拿到 role、llm profile、重试/超时/心跳策略。
2. 读取 `roles/{role_id}/state.md`、`context.md`、`inbox.md`、`session.reset.md`，并从 event-bus 读取该角色上次 offset 之后的增量事件。
3. 判断本轮是：
   - 无新任务/无新打断：直接 no-op
   - 存在 reset 请求：先处理 reset
   - 存在待确认 interrupt：按安全边界处理 interrupt
   - 存在新的 `TASK_DISPATCHED`：进入执行流程
4. 将角色状态写为 `RUNNING`，记录本轮 `turn_seq`、`task_id`、`context_version`、`inbox_version`。
5. 生成 `turns/NNNN-input.md`，按配置或回退顺序把 requirement snapshot / context / inbox / 必读文件 / 约束信息编入输入。
6. 调起执行器，捕获 stdout / stderr / exit code / duration，持续刷新 `heartbeat.md`。
7. 将原始输出写入 `turns/NNNN-output.md`，从输出中提取标准化 `role_result` 块。
8. 由 runtime 渲染 `conclusion.md` 与 `outbox.md`，同时更新 `state.md`、`events.md` 和 bus offset。
9. 必要时向 event-bus 追加关键事件，再返回“本轮是否产生进展”。

整个 tick 必须支持重复调用而不重复消费同一任务：如果 `task_id + inbox_version + context_version + session_generation` 没有变化，且没有新的 interrupt / reset，则再次 tick 必须是幂等 no-op。

### 5.4 `state.md`、`heartbeat.md`、`outbox.md`、`conclusion.md` 协议

#### A. `roles/{role_id}/state.md`

至少包含以下字段：

- `role_id`
- `session_generation`
- `status`：`IDLE / RUNNING / COMPLETED / BLOCKED / FAILED / INTERRUPTED / RESET_PENDING`
- `current_stage_id`
- `current_task_id`
- `current_turn_seq`
- `context_version`
- `inbox_version`
- `last_consumed_event_id`
- `last_conclusion_status`
- `executor_provider`
- `updated_at`

#### B. `roles/{role_id}/heartbeat.md`

至少包含以下字段：

- `heartbeat_version`
- `role_id`
- `status`
- `current_task_id`
- `current_turn_seq`
- `last_seen_at`
- `expire_at`
- `session_generation`
- `updated_at`

`expire_at` 必须基于 `policies.timeout.role_heartbeat_timeout_seconds` 计算。`001` 即使不做复杂 watchdog，也必须把这个过期点落盘，供后续 orchestrator / recovery 判断角色是否失联。

#### C. `roles/{role_id}/outbox.md`

至少包含以下字段：

- `outbox_version`
- `role_id`
- `stage_id`
- `task_id`
- `turn_seq`
- `status`
- `conclusion_ref`
- `turn_output_ref`
- `updated_at`

首版 `outbox.md` 主要承担“角色回执摘要”的职责。虽然 `orchestrator 001` 还不直接消费它，但 `recovery`、`artifact` 和后续主控演进都可以直接复用这层摘要，而无需反复扫描完整 turn 输出。

#### D. `roles/{role_id}/conclusion.md`

必须兼容 `orchestrator 001` 当前消费契约，至少保留以下字段：

- `role_id`
- `stage_id`
- `task_id`
- `status`
- `summary`
- `output_refs`
- `updated_at`

允许在不破坏 leading values 的前提下追加额外字段，例如 `turn_seq`、`session_generation`、`executor_provider`，但 orchestrator 不得依赖这些扩展字段做首版决策。

### 5.5 `turns/*.md` 协议

`001` 要求每次真实执行都落两类文件：

1. `turns/0001-input.md`
2. `turns/0001-output.md`

输入文件至少包含：

- `turn_seq`
- `role_id`
- `stage_id`
- `task_id`
- `session_generation`
- `context_version`
- `inbox_version`
- `dispatch_event_id`
- `read_order`
- `must_read_files`
- `forbidden_writes`
- `system_prompt`
- `resolved_context_refs`

输出文件至少包含：

- `turn_seq`
- `executor_provider`
- `command`
- `exit_code`
- `duration_ms`
- `status`
- `summary`
- `raw_stdout`
- `raw_stderr`

为降低大模型输出解析的不稳定性，`001` 建议要求执行器在末尾产出一个标准块：

```markdown
## role_result
- status: SUCCESS
- summary: task finished
- output_refs: 
```

role-runtime 负责从该块中提取结构化结果；如果块缺失、格式错误或状态非法，runtime 应将本轮视为 `FAILED`，并把错误原因写入 `conclusion.md.summary` 与 `turns/NNNN-output.md`。

### 5.6 执行器抽象与 provider 支持

首版需要把“执行器能力”与“角色文件协议”解耦，避免把 `codex` 的命令行细节散落到核心流程里。建议抽象为：

- `ExecutorResolver`：根据 `role.llm_profile` 找到执行器。
- `Executor`：负责执行单轮任务，返回结构化 `ExecutorResult`。
- `PromptBuilder`：负责把 `context.md` / `inbox.md` / 附加约束拼成最终输入。

`001` 对 provider 的支持范围如下：

1. **`codex` + `cli`**：真实执行器，必须直接调用宿主机 `codex` CLI，并复用当前用户真实 `~/.codex/`。
2. **`deterministic` + `cli`**：测试专用执行器，通过 fixture 文件或环境变量返回预设结果，用于 retry / blocked / interrupt / reset 等稳定黑盒场景。

对其他 provider（例如 `claude_code`）的策略是：**首版不静默跳过，也不伪造成功**。一旦 role-runtime 实际要执行该角色，应返回明确的 unsupported 错误，并让 `run` 失败退出，而不是写出误导性的成功结论。

### 5.7 interrupt、heartbeat 与安全边界

首版打断协议要解决的不是“任意时刻立刻中断”，而是“在不破坏文件一致性的前提下可解释地停下来”。

因此 `001` 采用以下规则：

1. role-runtime 每次 tick 前都要读取 `bus/interrupts.md`，只处理 `target_role_id` 命中的记录。
2. 如果 interrupt 在执行前已存在，则 runtime 直接确认 interrupt，不启动新 turn，并把 `state.md.status` 写为 `INTERRUPTED`。
3. 如果 interrupt 在 CLI 子进程运行中到达，`001` 不保证 tool-call 粒度抢占；统一在**当前 turn 结束后**处理。这意味着 `runtime.role_runtime.safe_interrupt_boundary` 在 CLI 执行器首版里统一退化为 `turn_end` 语义。
4. interrupt 被处理后，需要：
   - 更新 `state.md`
   - 刷新 `heartbeat.md`
   - 在本地 `events.md` 追加审计条目
   - 通过 event-bus 记录 `ROLE_INTERRUPTED` 或等价事件

heartbeat 建议在执行前、执行中、执行后都刷新。执行中的刷新间隔建议取 `min(10s, timeout/3)`，既保证可见性，也避免产生过于密集的心跳写放大。

### 5.8 reset 协议与会话代际

虽然 `reset-session` 用户态 CLI 不在 `001` 范围内，但角色侧文件协议必须先稳定下来。

`roles/{role_id}/session.reset.md` 建议至少包含：

- `session_generation`
- `status`：`IDLE / REQUESTED / APPLIED`
- `requested_by`
- `reason`
- `requested_at`
- `applied_at`
- `last_cleared_task_id`

处理规则：

1. 初始 generation 为 `1`。
2. 收到 reset 请求后，runtime 在安全边界内停止旧上下文，generation 自增。
3. reset **不删除历史 `turns/*.md`、`conclusion.md`、`outbox.md` 的历史语义**；它只表示“后续 turn 不得继续沿用旧 CLI 上下文”。
4. reset 完成后，`state.md.current_task_id` 清空，`status` 回到 `IDLE` 或 `INTERRUPTED`，并等待 orchestrator 重发任务包。

### 5.9 `run` 集成与有界同步运行循环

为了尽快形成最小闭环，`role-runtime 001` 不引入常驻后台服务，而是在 `run` 命令中接入一个有界同步运行循环：

1. `run` 完成 config 校验、session 初始化、event-bus bootstrap。
2. `run` 执行 orchestrator bootstrap + 首轮 tick，写出首批 `context.md` / `inbox.md`。
3. `run` 调用 `role-runtime TickAll(sessionDir)`，串行执行当前已分发角色。
4. 角色产生 `conclusion.md` 后，`run` 再次触发 orchestrator tick 做收口。
5. 如仍有新分发任务，继续进入下一轮；直到：
   - workflow `COMPLETED`
   - workflow `FAILED`
   - workflow `WAITING_HUMAN`
   - 或达到“无新任务、无新 turn、无新结论”的无进展退出条件

这条链路的关键点不是追求并发，而是先把**主链路可运行**做实。真正的多角色并发、tmux pane 常驻和文件监听，都可以复用这个 tick 内核在 `002` 再扩展。

### 5.10 event-bus 交互协议

`role-runtime 001` 必须复用 `event-bus 001`，但不能重新定义 bus 文件格式。建议每个角色使用独立消费者：`role-runtime/{role_id}`。

首版至少要使用以下能力：

- `ListAfter(last_event_id)`：读取本角色上次消费之后的增量事件
- `CommitOffset(consumer_id=role-runtime/{role_id})`：提交消费位点
- `ReadInterrupts()`：读取 interrupt 投影
- `Append(...)`：追加角色关键事件

建议新增或复用的事件类型：

- `ROLE_RUNTIME_BOOTSTRAPPED`
- `ROLE_TASK_ACCEPTED`
- `ROLE_TURN_STARTED`
- `ROLE_TURN_COMPLETED`
- `ROLE_CONCLUSION_WRITTEN`
- `ROLE_INTERRUPTED`
- `ROLE_RESET_APPLIED`
- `ROLE_RUNTIME_FAILED`

原则保持不变：bus 只承载可追溯事件，角色当前态仍以 `roles/{role_id}/` 下的 Markdown 文件为准。

### 5.11 模块边界

| 模块 | 负责内容 | 不负责内容 |
| --- | --- | --- |
| `session` | 创建 `roles/` 根目录和 session 基础骨架 | 创建具体角色文件、执行 turn |
| `event-bus` | offsets / interrupts / WAL / lock 协议 | 角色 turn 输入输出格式 |
| `orchestrator` | `context.md` / `inbox.md` 写入、结论消费、全局状态推进 | 角色 CLI 执行、心跳刷新 |
| `role-runtime` | `state.md`、`heartbeat.md`、`outbox.md`、`conclusion.md`、`session.reset.md`、`turns/*.md`、角色 bus 位点 | 跨角色调度、工作流排程 |
| `human-gate` | 后续写 interrupt / reset / inject 请求 | 直接修改角色运行态文件 |
| `artifact` | 后续管理 output_refs 对应产物版本与索引 | 决定角色结论状态 |
| `recovery` | 后续基于 bus + role files 恢复角色状态 | 首次创建 turn 文件 |

## 6. 分阶段交付物

### 6.1 必交付物

1. `docs/role-runtime/001/prd.md`
2. `docs/role-runtime/001/e2e.md`
3. `internal/roleruntime` 首版服务实现
4. `cmd/openoctopus/run.go` 对有界同步 runtime loop 的接入
5. 至少一组单测 / 命令级测试，覆盖 turn 渲染、结论解析、幂等与打断守卫
6. `e2e/role-runtime/` 黑盒验证目录与测试 harness

### 6.2 建议代码落位

建议保持职责分拆，避免单文件过大：

- `internal/roleruntime/types.go`：状态枚举、渲染模型、执行器结果模型
- `internal/roleruntime/errors.go`：稳定错误类型
- `internal/roleruntime/bootstrap.go`：角色目录与初始文件 bootstrap
- `internal/roleruntime/state.go`：`state.md` / `heartbeat.md` / `session.reset.md` 读写
- `internal/roleruntime/turns.go`：turn 输入输出渲染与编号
- `internal/roleruntime/executor.go`：执行器接口与 resolver
- `internal/roleruntime/executor_codex.go`：真实 Codex CLI 执行器
- `internal/roleruntime/executor_deterministic.go`：测试执行器
- `internal/roleruntime/engine.go`：`TickRole` / `TickAll` 主流程
- `internal/roleruntime/*_test.go`：模块单测

CLI 侧保持最小变更：

- `cmd/openoctopus/run.go`
- `cmd/openoctopus/command_test.go`

## 7. 验收标准

以下条件同时满足，才视为 `role-runtime 001` 达标：

1. `run` 在合法单阶段配置下，能从 orchestrator 分发继续推进到角色真实执行，而不是停留在 synthetic `conclusion.md`。
2. `roles/{role_id}/state.md`、`session.reset.md`、`outbox.md`、`conclusion.md`、`heartbeat.md`、`events.md`、`turns/` 都会按协议创建并保持字段一致。
3. 每次真实执行都会生成成对的 `turns/NNNN-input.md` 与 `turns/NNNN-output.md`。
4. `conclusion.md` 字段满足 orchestrator 当前消费要求，且 `SUCCESS / NEEDS_RETRY / BLOCKED / FAILED` 都能被真实 runtime 产出。
5. interrupt、reset、重复 tick 都具备幂等与安全边界，不会重复生成 turn 或破坏现有结论。
6. 至少一个 E2E 场景复用真实宿主机 `codex` 环境成功跑通。
7. `make check` 能覆盖 role-runtime 新增实现与测试，不破坏已有模块基线。

## 8. 依赖关系与风险

### 8.1 上游依赖

- `config 001`：提供 role、llm profile、超时、约束和 runtime 基础配置
- `session 001`：提供 `roles/`、`artifacts/`、`state/effective_config.yaml` 与 session 根骨架
- `event-bus 001`：提供 offsets、interrupts、WAL 和租约写事件能力
- `orchestrator 001`：提供 `context.md` / `inbox.md` 分发协议和 `conclusion.md` 消费契约

### 8.2 下游影响

- `human-gate 001` 将直接复用 interrupt / reset 协议与角色安全边界
- `recovery 001` 将直接复用 `state.md`、`session.reset.md`、`turns/*.md`、`events.md`
- `artifact 001` 将直接复用 `conclusion.md.output_refs` 与 `outbox.md.turn_output_ref`

### 8.3 主要风险

1. **真实 Codex 输出稳定性不足**：必须让 runtime 自己渲染 `conclusion.md`，而不是直接信任 CLI 原始文本。
2. **模块边界被 run 集成打穿**：`run` 可以调 role-runtime，但不得让 orchestrator 越权去写 `state.md` / `turns/*.md`。
3. **并发语义被误解**：首版是“同步串行执行”，不是“真实多角色并行执行”，文档和测试必须明确这一点。
4. **interrupt 安全边界过粗**：CLI 首版只能保证 turn-end 打断，需要明确写出，不可暗示 tool-call 粒度抢占。
5. **reset 容易误删历史**：必须坚持 reset 只清 CLI 上下文，不清 turn 历史，不覆盖结论追溯链。

## 9. 与模块总纲的关系

- `docs/role-runtime/prd.md` 是模块总纲，定义“接任务、跑回合、落文件、写结论、屏蔽执行器差异”的总体职责边界。
- 本文是该总纲的首个版本化落地稿，把抽象职责细化为：角色目录协议、单次 tick 流程、执行器抽象、interrupt / reset 语义、`run` 有界同步闭环与黑盒验证边界。
- 后续如果进入 `role-runtime 002`，必须在新版本目录继续演进，不能直接覆盖本目录文档。
