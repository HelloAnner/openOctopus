# Event Bus 模块 001 阶段 PRD

## 1. 阶段定位

`event-bus 001` 的目标，不是马上把主 Agent 调度、角色执行、人工打断和恢复链路全部跑通，而是先把 **事件先落盘、状态后推进** 的文件协议基线做实。

在当前仓库里，`session 001` 已经能够创建 `.octopus/sessions/{session_id}/bus/` 目录，并生成 `events.md`、`offsets.md`、`interrupts.md`、`lock.md` 这些占位文件；但这些文件目前还只是“位置存在”，并没有成为真正可读、可写、可校验、可恢复的总线协议。

`event-bus 001` 要解决的核心问题是：把 session 初始化出来的总线占位文件，升级为后续 `orchestrator`、`role-runtime`、`human-gate`、`recovery` 都能复用的 **首版 Markdown WAL 总线**。

这意味着本阶段要从“bus 目录已经存在”升级到“有首条事件、有可追加的事件日志、有可比较版本的锁、有可推进但不可回退的消费位点、有可追踪的打断队列”的首版总线基座。

`001` 的目标不是把 event-bus 做复杂，而是把未来所有状态推进都收口到一个统一入口，避免后续模块各自绕过总线直接改写状态文件。

## 2. 本阶段要解决的问题

### 2.1 当前问题

当前仓库已经有了 `bus/` 目录骨架，但距离“可用的总线协议”还有明显缺口：

1. **事件日志还是占位文件**：`bus/events.md` 目前没有事件 ID、序号、哈希链，也没有首条 `SESSION_CREATED` 事件。
2. **当前态和 WAL 不一致**：`session.state.md` 里已经有 `last_event: SESSION_CREATED`，但总线里并没有对应事件证据。
3. **没有统一并发控制**：`bus/lock.md` 还没有真正的租约字段、版本比较和过期判定规则。
4. **没有消费位点协议**：`bus/offsets.md` 还不能记录 orchestrator、role runtime、recovery 各自消费到哪里。
5. **没有打断投递协议**：`bus/interrupts.md` 还不能表达“谁在什么时候对谁发起了什么打断请求、当前状态是什么”。
6. **缺少读写边界**：还没有一个统一的 event-bus 服务接口，把事件追加、锁、offset、interrupt 的读写收口起来。
7. **缺少黑盒验证**：当前没有任何命令级或 E2E 证明 bus 文件已从占位文档升级为真实协议文件。

### 2.2 001 阶段目标

`event-bus 001` 的目标是建立一个简单但稳定的文件总线首版协议：

1. 将 session 初始化出的 bus 占位文件升级为可用协议文件。
2. 在 `bus/events.md` 中写入首条 `SESSION_CREATED` 事件，补齐 session 当前态与 WAL 的一致性。
3. 提供事件追加、事件读取、尾事件查询与哈希链校验能力。
4. 提供 `lock.md` 的租约获取、续租、释放和冲突检测能力。
5. 提供 `offsets.md` 的消费位点提交能力，并阻止 offset 回退。
6. 提供 `interrupts.md` 的打断请求 / 确认 / 清理投影能力。
7. 对外暴露清晰的 `internal/eventbus` 服务接口，让后续模块只依赖总线服务，不直接拼写 Markdown 文件。

## 3. 范围定义

### 3.1 001 范围内

- 基于已创建好的 session 目录 bootstrap event-bus。
- 将 `bus/events.md`、`bus/offsets.md`、`bus/interrupts.md`、`bus/lock.md` 初始化为首版协议内容。
- 在 bootstrap 时写入首条 `SESSION_CREATED` 事件。
- 提供事件追加、按顺序读取、按 `after_event_id` 增量读取、尾事件读取。
- 对 `events.md` 执行序号连续性与 `prev_event_hash -> event_hash` 链校验。
- 提供 `lock.md` 的获取、续租、释放与版本冲突判断。
- 提供 `offsets.md` 的原子 upsert 与不可回退约束。
- 提供 `interrupts.md` 的原子投影更新。
- 定义稳定错误类型，例如 `lease conflict`、`offset regression`、`event chain broken`。
- 在 `run` 成功创建 session 后，立即完成 bus bootstrap。

### 3.2 001 范围外

- 不负责 `orchestrator` 的调度算法、主循环与 `master_schedule.md` 业务推进。
- 不负责 `role-runtime` 的 `roles/{role_id}/turns/*.md`、`context.md`、`outbox.md` 写入。
- 不负责 `recovery` 的完整状态重建，只提供读取顺序与链校验基础能力。
- 不负责真正的用户态 `interrupt` / `interrupt-all` / `reset-session` 命令入口实现。
- 不负责 `roles/{role_id}/heartbeat.md` 的刷新写入；该文件的生产者仍属于 `role-runtime`。
- 不引入数据库、消息队列、外部 WAL 服务或后台守护进程。

## 4. 核心用户故事

### 4.1 命令行用户

- 作为用户，我希望 `run` 成功后，session 目录里不是只有 bus 占位文件，而是已经有首条真实事件，方便我立即排查系统到底初始化到了哪一步。
- 作为用户，我希望总线写入失败时能明确报错，而不是留下格式不完整、后续无法恢复的半条事件。

### 4.2 下游模块

- 作为 `orchestrator`，我希望可以先拿锁，再安全地向总线追加事件并推进 offset，而不是和别的模块互相覆盖文件。
- 作为 `role-runtime`，我希望可以稳定读取“自上次消费以来的新事件”，并在完成处理后提交自己的消费位点。
- 作为 `human-gate`，我希望未来写入打断请求时，有稳定的 `interrupts.md` 投影文件可读，而不是只能去扫一整份事件日志。
- 作为 `recovery`，我希望崩溃后可以从 `events.md` 头到尾重放，并快速发现日志是否被破坏或截断。

### 4.3 审计与排障

- 作为排障者，我希望知道某次写失败是锁冲突、offset 回退，还是事件链损坏。
- 作为复盘者，我希望通过 `event_id`、`event_hash`、`payload_ref` 明确知道某条事件对应哪份文档或哪个角色动作。

## 5. 核心方案

### 5.1 启动入口与模块接口

`event-bus 001` 的入口，不直接读取 YAML；它只接收已经创建好的 session 路径和必要的 bootstrap 元信息。

推荐接口形态：

```go
type BootstrapOptions struct {
    SessionID   string
    SessionDir  string
    WorkflowID  string
    MetadataRef string
}

type Lease struct {
    Holder       string
    LeaseToken   string
    LeaseVersion int64
    ExpireAt     time.Time
}

type AppendEvent struct {
    EventType        string
    Producer         string
    SessionID        string
    RoleID           string
    CorrelationID    string
    CausationEventID string
    PayloadRef       string
    Summary          string
}

type OffsetCommit struct {
    ConsumerID   string
    LastEventID  string
    LastSequence int64
    Note         string
}

type InterruptRecord struct {
    InterruptID   string
    Scope         string
    TargetRoleID  string
    Source        string
    Reason        string
    Status        string
    RequestEventID string
}
```

约束如下：

1. `event-bus` 只接收 session 路径和结构化输入，不直接读取原始配置文件。
2. 除 bootstrap 外，所有会修改 `events.md` / `offsets.md` / `interrupts.md` 的操作，都必须带有效 lease。
3. 业务模块不得直接改写 `bus/*.md`，统一通过 `internal/eventbus` 服务。

### 5.2 Bootstrap 协议

`session 001` 已经创建了 bus 目录和占位文件，`event-bus 001` 要做的是把这些占位文件升级为真正的协议文件。

bootstrap 完成后，`bus/` 应呈现如下状态：

```text
bus/
├── events.md
├── offsets.md
├── interrupts.md
└── lock.md
```

初始化规则如下：

1. **`events.md` 必须写入首条 `SESSION_CREATED` 事件**，补齐 session 当前态与 WAL 的证据链。
2. **`lock.md` 必须写成可解析的空闲锁状态**，而不是占位文本。
3. **`offsets.md` 必须写成合法但为空的消费位点文档**。
4. **`interrupts.md` 必须写成合法但为空的打断投影文档**。
5. **bootstrap 必须幂等**：如果 bus 文件已经是合法协议内容，则再次 bootstrap 不应重复写入第二条 `SESSION_CREATED`。

推荐由 `run` 在 `session.Create(...)` 成功后立即调用 `eventbus.Bootstrap(...)`。这样用户第一次看到 session 目录时，总线已经可用，而不是还停留在“后续模块再补”的中间态。

### 5.3 `events.md` 协议

`events.md` 是首版唯一的事件 WAL，采用 **追加语义 + 全文件原子替换写**。

之所以不用真正的文件尾部直接 append，而是“读全量 -> 内存追加 -> 写 `*.tmp` -> rename 替换”，是因为 `001` 更关心原子性与实现简单性，而不是大规模吞吐。当前仓库的 session 规模和事件量足以接受这一策略。

单条事件建议格式如下：

```markdown
### event-000001

- event_id: event-000001
- sequence: 1
- ts: 2026-03-08T10:00:00Z
- event_type: SESSION_CREATED
- producer: session
- session_id: sess_1741428000_ab12cd34
- role_id:
- correlation_id: bootstrap
- causation_event_id:
- payload_ref: metadata.md
- summary: session skeleton promoted to event bus
- prev_event_hash:
- event_hash: sha256:9f4c...
```

约束如下：

1. `event_id` 固定为 `event-%06d`，首条从 `event-000001` 开始。
2. `sequence` 必须严格单调递增，且与 `event_id` 数字部分一致。
3. `event_type` 采用大写蛇形命名，例如 `SESSION_CREATED`、`INTERRUPT_REQUESTED`。
4. `event-bus 001` 不在模块内硬编码完整业务事件枚举，只校验 `event_type` 非空且命名合法；不同模块的详细事件目录由各自模块文档维护。
5. `payload_ref` 优先指向已存在的 Markdown / YAML 文件，避免把复杂嵌套载荷直接塞进事件条目。
6. `event_hash` 通过“去掉 `event_hash` 字段后的稳定 JSON 序列化结果”计算 `sha256`。
7. `prev_event_hash` 指向上一条事件的 `event_hash`，首条事件留空。

### 5.4 事件写入与读取规则

事件写入规则：

1. 先读出当前 `events.md` 尾事件。
2. 校验序号连续性与上一条 `event_hash`。
3. 生成新的 `event_id`、`sequence`、`event_hash`。
4. 将新条目拼接到内存缓冲区。
5. 写入 `events.md.tmp`。
6. 使用 `rename` 原子替换原文件。

事件读取规则：

- 支持按全量顺序读取。
- 支持按 `after_event_id` 增量读取。
- 支持只读尾事件。
- 每次读取时都要做最小链路校验：序号不能跳号，`prev_event_hash` 必须匹配上一条 `event_hash`。
- 如果发现链损坏，直接返回 `event chain broken` 错误，不做静默修复。

### 5.5 `lock.md` 租约协议

`lock.md` 是 event-bus 的并发控制文件，用于保证多个写入方不会同时改写总线状态。

建议格式：

```markdown
# Bus Lock

- status: FREE
- holder:
- lease_token:
- lease_version: 0
- acquired_at:
- renewed_at:
- expire_at:
- last_operation: bootstrap
```

首版约束：

1. `lease_version` 是单调递增版本号，每次成功 acquire / renew / release 都要递增。
2. acquire 成功条件为：当前锁是 `FREE`，或已 `EXPIRED`。
3. renew / release 必须同时匹配 `lease_token` 与 `lease_version`。
4. 若版本不匹配，返回稳定冲突错误，不做覆盖写。
5. 锁状态文件采用原子替换写。
6. `expire_at` 以 UTC RFC3339 保存，由调用方传入 TTL 计算得到。

这里有一个首版取舍：**锁本身是总线控制平面，不强制把 acquire / renew / release 也写进 `events.md`**。这样可以避免“还没拿到锁就需要先写锁事件”的自举悖论，保持实现直接。未来如果确实需要锁审计，再在后续版本中补 `lock.audit.md` 或对应事件类型。

### 5.6 `offsets.md` 消费位点协议

`offsets.md` 记录各个消费者已处理到哪条事件，属于当前态投影文件。

建议格式：

```markdown
# Bus Offsets

## consumer: orchestrator/master
- consumer_id: orchestrator/master
- last_event_id: event-000005
- last_sequence: 5
- updated_at: 2026-03-08T10:05:00Z
- note: schedule updated
```

规则如下：

1. `consumer_id` 只要求非空且稳定，例如 `orchestrator/master`、`role/impl_agent`、`recovery/replayer`。
2. offset 提交前，必须持有有效 lease。
3. `last_sequence` 只能前进，不能回退。
4. 同一 `consumer_id` 再次提交时，使用全文件原子替换做 upsert。
5. `offsets.md` 内的消费者块按 `consumer_id` 排序，保证 diff 稳定。
6. 每次成功提交 offset 后，建议同步向 `events.md` 追加一条 `OFFSET_COMMITTED` 事件，再更新 `offsets.md` 当前态。

### 5.7 `interrupts.md` 打断投影协议

`interrupts.md` 不是完整 WAL，而是“打断请求的当前态投影”。完整历史仍以 `events.md` 为准。

建议格式：

```markdown
# Interrupts

## interrupt: event-000006
- interrupt_id: event-000006
- scope: role
- target_role_id: impl_agent
- source: cli
- reason: waiting for human approval
- status: REQUESTED
- request_event_id: event-000006
- ack_event_id:
- clear_event_id:
- created_at: 2026-03-08T10:06:00Z
- updated_at: 2026-03-08T10:06:00Z
```

规则如下：

1. 首次打断请求先追加 `INTERRUPT_REQUESTED` 事件，再更新 `interrupts.md`。
2. `interrupt_id` 直接复用请求事件的 `event_id`，避免再额外发明一套自增 ID。
3. 确认打断时追加 `INTERRUPT_ACKNOWLEDGED` 事件，并更新 `status` 与 `ack_event_id`。
4. 清理打断时追加 `INTERRUPT_CLEARED` 事件，并更新 `status` 与 `clear_event_id`。
5. 同一 `interrupt_id` 的状态只能沿 `REQUESTED -> ACKNOWLEDGED -> CLEARED` 前进，不能倒退。

### 5.8 心跳边界

虽然平台总纲里提到了 `heartbeat.md`，但在当前目录协议中，真正的心跳文件位于 `roles/{role_id}/heartbeat.md`，而不是 `bus/heartbeat.md`。

因此 `event-bus 001` 的边界保持为：

- **负责**：提供记录心跳相关事件的统一 WAL 能力，例如未来的 `ROLE_HEARTBEAT_REPORTED`、`ROLE_HEARTBEAT_TIMEOUT`。
- **不负责**：直接创建或刷新 `roles/{role_id}/heartbeat.md`；这属于 `role-runtime 001`。

### 5.9 错误模型与失败策略

首版至少要区分以下错误：

- `bus not initialized`
- `event chain broken`
- `lease conflict`
- `lease expired`
- `offset regression`
- `interrupt not found`
- `invalid event type`

失败策略如下：

1. 任何当前态文件重写都必须走 `*.tmp + rename`。
2. 事件追加失败时，原 `events.md` 必须保持不变。
3. 当前态投影写失败时，事件可以已经写入；调用方应得到明确错误，并允许后续通过重放或 repair 重新投影。
4. 不做静默跳过，不做自动截断修复，不做自动降级覆盖。

### 5.10 模块边界

`event-bus 001` 与其他模块的责任边界如下：

| 模块 | 负责内容 | 不负责内容 |
| --- | --- | --- |
| `session` | 创建 `bus/` 目录与占位文件，提供 session 根目录 | 真实 WAL、锁、offset、interrupt 协议 |
| `event-bus` | `events.md`、`lock.md`、`offsets.md`、`interrupts.md` 的真实读写协议与服务接口 | 调度算法、角色执行、人工决策 |
| `orchestrator` | 获取锁、追加调度相关事件、提交自己的 offset | 设计总线底层文件格式 |
| `role-runtime` | 读取新事件、写角色事件、提交角色 offset、刷新角色心跳 | 设计总线底层文件格式 |
| `human-gate` | 触发打断 / 注入请求并通过总线投递 | 直接改写 `bus/*.md` |
| `recovery` | 基于总线顺序读取与链校验做回放 | 负责首次创建 bus |

边界原则仍然只有一句话：**业务模块可以“表达动作”，但不能“自定义总线协议”。**

## 6. 分阶段交付物

### 6.1 必交付物

1. `docs/event-bus/001/prd.md`
2. `docs/event-bus/001/e2e.md`
3. `docs/event-bus/001/plans/` 下的拆分任务文档
4. `internal/eventbus` 的首版服务实现
5. `cmd/openoctopus/run.go` 对 event-bus bootstrap 的接入
6. 至少一组单测 / 命令级测试，验证 `run` 后 bus 文件已从占位状态升级为真实协议状态

### 6.2 建议代码落位

建议保持轻量分文件，不要把总线所有逻辑塞进一个大文件：

- `internal/eventbus/types.go`：事件、lease、offset、interrupt 基础模型
- `internal/eventbus/errors.go`：稳定错误类型
- `internal/eventbus/bootstrap.go`：bus 初始化与 placeholder 升级
- `internal/eventbus/events.go`：事件序列化、追加、读取、链校验
- `internal/eventbus/lock.go`：锁获取、续租、释放
- `internal/eventbus/offsets.go`：消费位点 upsert
- `internal/eventbus/interrupts.go`：打断投影维护
- `internal/eventbus/store.go`：对外服务入口
- `internal/eventbus/*_test.go`：模块测试

CLI 侧保持最小变更：

- `cmd/openoctopus/run.go`
- `cmd/openoctopus/command_test.go`

## 7. 验收标准

以下条件同时满足，才视为 `event-bus 001` 达标：

1. `run` 在合法配置下创建 session 后，`bus/*.md` 都是可解析协议文件，而不是 session 占位文本。
2. `bus/events.md` 存在且首条事件为 `SESSION_CREATED`。
3. 连续追加两条事件时，`event_id`、`sequence`、`prev_event_hash`、`event_hash` 一致且可校验。
4. `lock.md` 支持 acquire / renew / release，且 stale version 会被拒绝。
5. `offsets.md` 支持 upsert，同一消费者不能回退 `last_sequence`。
6. `interrupts.md` 能稳定表示请求、确认、清理三种状态推进。
7. 任一写入失败不会留下半条事件或半写 Markdown。
8. `make check` 能覆盖 event-bus 新增测试，不破坏已有 `config` 与 `session` 基线。

## 8. 依赖关系与风险

### 8.1 上游依赖

- `config 001`：提供合法 `RuntimeConfig`，保证 `run` 能继续走到 session 创建阶段
- `session 001`：提供标准 `bus/` 目录与占位文件
- 当前 `run` 入口：负责在 session 创建成功后调用 bus bootstrap

### 8.2 下游影响

- `orchestrator 001` 将直接复用 lease、append、offset 能力
- `role-runtime 001` 将直接复用事件读取、增量消费和 interrupt 投影
- `human-gate 001` 将直接复用 interrupt 事件与投影协议
- `recovery 001` 将直接复用事件读取顺序与链校验能力

### 8.3 主要风险

1. **全文件替换写的性能边界**：首版为了简单采用整文件原子替换，后续事件量大时可能要优化，但当前阶段这是更稳的选择。
2. **锁协议与 WAL 的边界容易混淆**：如果不明确锁是控制平面，后续实现可能陷入“写锁前先写事件”的自举死结。
3. **事件类型膨胀**：如果 event-bus 试图维护全平台完整事件枚举，会过早绑死 orchestrator、role-runtime 的演进节奏。
4. **投影与 WAL 失配**：若事件已写入但 `offsets.md` / `interrupts.md` 投影失败，必须保留重放和 repair 的空间，不能靠静默覆盖掩盖问题。

## 9. 与模块总纲的关系

- `docs/event-bus/prd.md` 是模块总纲，定义“事件先落盘、状态后推进、支持恢复与审计”的大边界。
- 本文是该总纲的首个版本化落地稿，把“总线职责”细化为“首版到底有哪些 bus 文件、怎样写第一条事件、怎样拿锁、怎样推 offset、怎样记 interrupt”。
- 后续如果进入 `event-bus 002`，必须在新版本目录中继续演进，不能直接覆盖本目录文档。
