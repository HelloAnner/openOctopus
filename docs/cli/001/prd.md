# CLI 模块 001 阶段 PRD

## 1. 阶段定位

`cli 001` 不追求一次把 `init`、`stop`、`report`、`debug`、`status --watch`、Bubble Tea 交互面板全部做完，而是先把 **当前已经逐步成型的命令入口，收敛成一个稳定、可脚本化、可观测的正式 CLI 基线**。

到目前为止，仓库已经围绕 `config`、`session`、`event-bus`、`orchestrator`、`role-runtime`、`human-gate` 落地了一批真实能力，`cmd/openoctopus/` 也已经有 `validate`、`run`、`interrupt`、`interrupt-all`、`inject`、`resume` 这些命令。但这些命令目前仍然存在明显的首版缺口：输出格式不统一、缺少正式 `status` 命令、脚本侧很难稳定消费结果、会话定位逻辑分散、错误退出码也还没有被收口成明确协议。

因此，`cli 001` 的核心目标不是新增更多命令数量，而是把“已经存在且真实可用”的 CLI 能力，升级为一个 **可供用户手工使用，也可供 shell / CI / E2E 稳定消费的正式门面层**。

---

## 2. 本阶段要解决的问题

### 2.1 当前问题

当前 CLI 已经能工作，但离“稳定正式入口”还有几个关键差距：

1. **输出协议不统一**：`validate`、`run`、`interrupt`、`inject`、`resume` 都是纯文本输出，字段形状不固定，脚本侧只能靠字符串截取。
2. **缺少正式观测入口**：用户虽然能手动去翻 `.octopus/sessions/{id}`，但还没有 `status` 命令把 session 当前态、blocker、调度状态做成稳定输出。
3. **错误退出码不清晰**：当前除了成功 / 失败，没有为“配置校验失败”“session 不存在”这些高频可识别错误建立稳定退出码。
4. **session 解析逻辑分散**：`interrupt`、`inject`、`resume` 等命令都依赖 session 参数，但解析逻辑目前收在 `human-gate` 模块里，不利于 CLI 层统一复用。
5. **命令展示层职责还不清楚**：现在很多命令“能调服务”，但没有把“展示给用户什么、JSON 输出长什么样、stderr 如何表达错误”沉淀成协议。

### 2.2 001 阶段目标

`cli 001` 的目标是建立一个简单但稳定的首版 CLI 协议：

1. 正式支持 `validate`、`run`、`status`、`interrupt`、`interrupt-all`、`inject`、`resume` 这组首版命令。
2. 为上述命令统一提供 `--format text|json` 输出协议。
3. 提供首版 `status` 命令，稳定输出 session 当前态、调度摘要和 blocker 摘要。
4. 将高频错误至少收口到一组最小可用退出码：成功、配置校验失败、session 不存在、通用执行失败。
5. 把 session 引用解析和状态读取抽到 CLI 可复用支撑层，避免继续散落在单个业务模块里。

---

## 3. 范围定义

### 3.1 001 范围内

- `validate` / `run` / `status` / `interrupt` / `interrupt-all` / `inject` / `resume` 的正式命令协议。
- `--format text|json` 的统一实现。
- CLI 输出模型、错误输出模型与最小退出码约定。
- `status` 所需的 session 读取服务与 Markdown 读模型。
- 通用 session 引用解析逻辑（支持 session id 与 session dir）。
- 对现有命令测试、E2E 与 `Makefile` 的补齐。

### 3.2 001 范围外

- 不实现 `init` / `stop` / `report` / `debug` 的正式业务能力。
- 不实现 `status --watch`、Bubble Tea、交互式审批面板。
- 不新增 daemon、HTTP API 或 Web 控制台。
- 不把 orchestrator / role-runtime / human-gate 的业务判断重新搬回 CLI。
- 不改变现有 session、event-bus、orchestrator、role-runtime 的底层协议格式。

---

## 4. 核心用户故事

### 4.1 脚本用户

- 作为脚本调用方，我希望 `validate --format json` 能直接给出成功 / 失败和 defaults 数量，而不是再自己解析文本。
- 作为 CI，我希望 `run --format json` 能稳定拿到 `session_id` 和 `session_dir`，便于后续继续调用 `status` / `interrupt` / `resume`。
- 作为自动化测试，我希望“配置错误”和“session 不存在”能有稳定退出码，而不是所有错误都只有 `1`。

### 4.2 命令行用户

- 作为用户，我希望 `status` 可以快速告诉我 session 当前是 `RUNNING`、`WAITING_HUMAN` 还是 `COMPLETED`，而不是我自己翻多个 Markdown 文件。
- 作为用户，我希望 `interrupt`、`inject`、`resume` 这些命令既能有好读的文本输出，也能切换成 JSON 供脚本复用。
- 作为排障者，我希望错误信息继续保留 stderr，可读且带上下文，而不是只有模糊失败。

### 4.3 模块协作边界

- 作为 `human-gate`，我希望 CLI 只负责参数解析和结果展示，真正的 interrupt / inject / resume 逻辑仍留在服务层。
- 作为 `orchestrator` 与 `session`，我希望 `status` 仅读取现有协议文件，不反向修改任何业务文件。

---

## 5. 核心方案

### 5.1 命令集合

`cli 001` 的正式命令集合定义如下：

| 命令 | 职责 | 依赖模块 |
| --- | --- | --- |
| `validate` | 校验配置并输出 defaults / 错误摘要 | `config` |
| `run` | 以正式 CLI 入口启动一次 session | `config` / `session` / `event-bus` / `orchestrator` / `role-runtime` / `artifact` |
| `status` | 读取 session 当前态和调度摘要 | `session` / `orchestrator` / `human-gate` |
| `interrupt` | 对单角色发起人工中断 | `human-gate` |
| `interrupt-all` | 对全部未完成角色发起中断并进入等待人工 | `human-gate` |
| `inject` | 追加人工消息 | `human-gate` |
| `resume` | 清理等待人工并继续推进 | `human-gate` / `orchestrator` / `role-runtime` |

### 5.2 统一输出协议

所有首版命令统一支持：

```bash
--format text
--format json
```

默认 `text`，用于人类阅读；`json` 用于脚本消费。

建议 JSON 统一最小形状：

```json
{
  "ok": true,
  "command": "status",
  "data": {
    "session_id": "sess_xxx",
    "workflow_status": "RUNNING"
  }
}
```

失败时输出到 stderr，保持同样结构：

```json
{
  "ok": false,
  "command": "validate",
  "error": {
    "code": "config_validation_failed",
    "message": "config validation failed"
  }
}
```

原则：

1. stdout 只放成功结果。
2. stderr 只放错误结果。
3. `text` 输出允许自然语言，但关键词必须稳定。
4. `json` 输出字段必须稳定，可被 E2E 直接断言。

### 5.3 `status` 命令协议

`status` 是 `cli 001` 唯一新增的正式命令。它只读 session 文件，不写业务状态。

输入：

```bash
openoctopus status --session <session-id-or-dir> [--format text|json]
```

最小输出字段：

- `session_id`
- `session_dir`
- `workflow_id`
- `workflow_name`
- `workflow_status`
- `current_stage_id`
- `current_role_id`
- `schedule_version`
- `active_dispatch_count`
- `blocker_summary`
- `updated_at`

其中：

- `workflow_id` / `workflow_name` 来自 `metadata.md`
- `workflow_status` / `current_stage_id` / `current_role_id` / `updated_at` 来自 `session.state.md`
- `schedule_version` / `active_dispatch_count` 来自 `planner/master_schedule.md`
- `blocker_summary` 来自 `planner/blockers.md`

如果某个文件仍处于 placeholder 状态，CLI 不应 panic，而应回退为空值或默认文案。

### 5.4 session 引用解析

CLI 统一支持两种 session 输入：

1. session 目录绝对 / 相对路径
2. session id（自动在 `workingDir/.octopus/sessions/{id}` 下解析）

后续 `status`、`interrupt`、`interrupt-all`、`inject`、`resume` 全部复用同一套解析器。

### 5.5 最小退出码

`cli 001` 只建立最小必要退出码：

| 退出码 | 含义 |
| --- | --- |
| `0` | 成功 |
| `1` | 通用执行失败 |
| `2` | 配置校验失败 |
| `3` | session 不存在 |

说明：

- 不是所有业务错误都要在 `001` 细分退出码。
- 只优先覆盖脚本中最常见、最有稳定价值的两类错误：配置阻断和 session 找不到。

---

## 6. 分层与实现边界

### 6.1 `cmd/openoctopus`

负责：

- Cobra 命令装配
- flag 定义
- 调用服务
- 成功 / 失败输出渲染
- 错误与退出码映射

不负责：

- 真正修改 session、interrupt、schedule 的业务逻辑
- 自己手写业务文件内容

### 6.2 `internal/cli`

新增一个轻量 CLI 支撑层，负责：

- session 引用解析
- `metadata.md` / `session.state.md` / `master_schedule.md` / `blockers.md` 的只读解析
- `status` 聚合读模型
- 可复用的最小错误类型（如 `session not found`）

这个包只读现有协议文件，不承担业务写操作，保持“CLI 展示服务”定位。

### 6.3 现有业务模块

- `config` 继续负责配置加载、默认值注入、错误校验。
- `human-gate` 继续负责 interrupt / inject / resume 的业务写操作。
- `orchestrator` / `role-runtime` / `session` 继续维护各自文件协议。
- CLI 只读取或转调，不改写业务分层。

---

## 7. 测试与验证要求

### 7.1 命令级测试

至少覆盖：

1. `validate --format json` 成功输出。
2. `validate --format json` 失败输出。
3. `run --format json` 成功输出 session 信息。
4. `status` 在 `RUNNING` / `WAITING_HUMAN` / `COMPLETED` 其中至少两种状态下可正确读取。
5. `status` 在 session 不存在时返回稳定错误。
6. `interrupt` / `interrupt-all` / `inject` / `resume` 的 `--format json` 结果可稳定断言。

### 7.2 E2E

E2E 只做黑盒验证：

- `validate --format json`
- `run --format json`
- `status --format json`
- `interrupt-all` 后 `status` 反映 `WAITING_HUMAN`
- session 不存在的退出码与错误输出

---

## 8. 交付物

`cli 001` 完成时，至少应交付：

1. `docs/cli/001/prd.md`
2. `docs/cli/001/e2e.md`
3. `docs/cli/001/plans/`
4. `docs/plans/2026-03-08-cli-001-implementation.md`
5. `cmd/openoctopus/status.go`
6. `cmd/openoctopus` 下统一输出 / 错误处理代码
7. `internal/cli/` 只读支撑层
8. `cmd/openoctopus` 单测
9. `e2e/cli/` 黑盒测试
10. `Makefile` 与 `e2e/README.md` 同步更新

---

## 9. 完成标准

满足以下条件，才视为 `cli 001` 完成：

1. 现有正式命令都支持 `--format text|json`。
2. `status` 命令可稳定读取 session 当前态与 blocker 摘要。
3. 脚本可通过 JSON 输出稳定拿到 `session_id`、`session_dir`、`workflow_status` 等核心字段。
4. 配置校验失败和 session 不存在拥有稳定退出码。
5. 命令级测试、`e2e/cli` 和 `make check` 全部通过。

