# Event Bus 模块 001 阶段 E2E 方案

## 1. 目标

`event-bus 001` 的 E2E 不验证完整的多角色协作，而是聚焦一个问题：**当 `run` 创建 session 成功后，bus 目录是否会被稳定升级为真实可用的事件总线；并且在真实文件系统上，锁、事件、offset、interrupt 协议是否能以黑盒方式被验证。**

这批测试要证明五件事：

1. `run` 成功后，`bus/*.md` 不再是 session 占位文本。
2. `bus/events.md` 中会存在首条 `SESSION_CREATED` 事件。
3. event-bus 的追加写、链校验、锁冲突、offset 提交、interrupt 投影，在真实 session 目录上可以稳定运行。
4. 事件或投影写失败时，不会破坏已有文件内容。
5. 这些验证默认在宿主机执行，不依赖 Docker，也不需要真正启动 orchestrator / role runtime。

## 2. 验证范围

### 2.1 本阶段必须覆盖

- `run` 成功创建 session 后自动 bootstrap event-bus。
- `bus/events.md` 首条事件是 `SESSION_CREATED`。
- `bus/lock.md` 初始状态是可解析的 `FREE` 锁，而不是占位文案。
- 在已 bootstrap 的 session 上追加两条事件，序号和哈希链正确。
- stale lease 获取 / 续租 / 释放被拒绝。
- `offsets.md` 支持前进式 upsert，回退会失败。
- `interrupts.md` 支持请求、确认、清理三类状态推进。
- 手工破坏 `events.md` 的哈希链后，读取接口会稳定报错。

### 2.2 本阶段暂不覆盖

- `orchestrator` 真正的调度主循环。
- `role-runtime` 真正的被动触发与 `turns/*.md` 写入。
- 用户态 `interrupt` / `interrupt-all` / `reset-session` 正式命令。
- `recovery` 的完整回放重建。
- `roles/{role_id}/heartbeat.md` 的真实刷新与超时判定。

## 3. 环境与目录约定

首版继续遵守仓库统一 E2E 规范：默认直接在宿主机执行，不额外引入 Docker。

建议目录结构：

```text
e2e/
├── conftest.py
├── requirements.txt
├── config/
│   └── ...
├── session/
│   └── ...
└── eventbus/
    ├── fixtures/
    │   ├── valid-minimal/
    │   │   └── octopus.yaml
    │   ├── valid-repeat-bootstrap/
    │   │   └── octopus.yaml
    │   └── valid-lock-conflict/
    │       └── octopus.yaml
    ├── harness/
    │   └── main.go
    ├── test_run_bootstrap.py
    └── test_bus_mutations.py
```

额外约束：

- 真实测试工作目录统一使用仓库根目录 `e2e-test/`
- 每次执行前先清理 `e2e-test/`
- 首版 E2E 允许额外构建一个 **测试专用 harness 二进制** 来调用 `internal/eventbus`，但它不是正式用户命令
- 黑盒断言仍然只关注 CLI / harness 退出码、标准输出和文件系统副作用，不从 Python 直接调用 Go 私有函数

### 3.1 宿主机执行约定

执行流程建议如下：

1. 清理 `./e2e-test`
2. 构建当前仓库 `openoctopus` 二进制
3. 构建 `e2e/eventbus/harness` 测试二进制
4. 复制 fixture 到 `e2e-test/{case_name}/`
5. 先执行 `openoctopus run --config ./octopus.yaml`
6. 再通过 harness 对已创建的 session 执行 append / lock / offset / interrupt 黑盒操作
7. 通过 stdout、退出码和 `.octopus/` 文件内容做断言

执行命令推荐：

```bash
python3 -m pytest e2e/eventbus -v
```

## 4. 测试夹具设计

### 4.1 `valid-minimal`

目标：证明最小合法配置经过 `run` 后，可以完成 session + event-bus bootstrap，并允许后续总线操作。

特点：

- 只保留首版必填字段
- 依赖 `config 001` 默认值与 `session 001` 目录骨架
- 不要求任何 role runtime 真正执行

### 4.2 `valid-repeat-bootstrap`

目标：证明重复打开同一个 session 执行 bootstrap，不会重复写第二条 `SESSION_CREATED` 事件。

设计方式：

- 先执行一次 `run`
- 再通过 harness 显式调用 `bootstrap`
- 断言 `events.md` 仍只有一条 `SESSION_CREATED` 事件

### 4.3 `valid-lock-conflict`

目标：证明 lock 协议能拒绝 stale version 或错误 token。

设计方式：

- 先通过 harness 成功 acquire 一次
- 再用旧版本或旧 token 尝试 renew / release
- 断言操作失败，且 `lock.md` 保持前一个合法状态

## 5. 核心用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| BUS-E2E-001 | `run` 触发 bus bootstrap | `valid-minimal/octopus.yaml` | `run` 退出码为 `0`，stdout 含 `session created:` |
| BUS-E2E-002 | 首条事件写入 | `valid-minimal/octopus.yaml` | `bus/events.md` 含 `SESSION_CREATED`、`event-000001` |
| BUS-E2E-003 | 初始锁状态合法 | `valid-minimal/octopus.yaml` | `bus/lock.md` 含 `status: FREE`、`lease_version: 0` |
| BUS-E2E-004 | 事件顺序追加 | harness append 两次 | `event-000002`、`event-000003` 序号连续且哈希链正确 |
| BUS-E2E-005 | 重复 bootstrap 幂等 | `valid-repeat-bootstrap` + harness bootstrap | 不产生第二条 `SESSION_CREATED` |
| BUS-E2E-006 | 锁冲突拒绝 | `valid-lock-conflict` + stale version | harness 非 `0`，`lock.md` 状态不被破坏 |
| BUS-E2E-007 | offset 前进提交 | commit `event-000003` | `offsets.md` 更新成功 |
| BUS-E2E-008 | offset 回退拒绝 | commit 更小 sequence | harness 非 `0`，`offsets.md` 保持原值 |
| BUS-E2E-009 | interrupt 请求与确认 | request + ack + clear | `interrupts.md` 状态按顺序推进 |
| BUS-E2E-010 | 事件链损坏可检测 | 手工破坏 `prev_event_hash` | read/list 非 `0`，输出 `event chain broken` |

## 6. 关键断言方式

### 6.1 `run` 成功断言

- 进程退出码为 `0`
- stdout 至少包含 `session created:`
- 能从 stdout 解析出真实 `session_dir`
- `session_dir/bus/` 真实存在

### 6.2 bootstrap 断言

至少验证：

- `bus/events.md` 不再包含 `Initialized by session 001.`
- `bus/events.md` 包含 `event-000001`
- `bus/events.md` 包含 `SESSION_CREATED`
- `bus/lock.md` 包含 `status: FREE`
- `bus/offsets.md` 和 `bus/interrupts.md` 有合法标题，而不是空文件

### 6.3 事件链断言

至少验证：

- `event_id` 连续递增
- `sequence` 连续递增
- 第二条事件的 `prev_event_hash` 等于第一条事件的 `event_hash`
- 读取接口在链损坏时返回稳定错误，而不是忽略错误继续读

### 6.4 锁断言

- acquire 成功后 `lock.md` 中 `holder`、`lease_token`、`lease_version`、`expire_at` 都有值
- stale version renew / release 返回非 `0`
- 失败后 `lock.md` 不会被覆盖成错误状态

### 6.5 offset 与 interrupt 断言

- `offsets.md` 同一 `consumer_id` 只能前进，不能回退
- `interrupts.md` 中同一 `interrupt_id` 的状态只能沿 `REQUESTED -> ACKNOWLEDGED -> CLEARED` 前进
- interrupt 投影更新后，对应事件也真实存在于 `events.md`

## 7. 推荐测试实现

### 7.1 复用现有 `pytest` 基础设施

建议继续复用 `e2e/conftest.py` 的基础能力：

- `project_root`
- `binary_path`
- `workspace_dir`
- `run_cli(command: list[str], env: dict[str, str] | None = None)`
- `prepare_case(...)`

对 `eventbus` 模块额外补充：

- `eventbus_harness_path`
- `parse_session_dir(stdout: str) -> Path`
- `run_harness(args: list[str], cwd: Path)`
- `read_text(path: Path) -> str`

### 7.2 Harness 子命令建议

为避免过早引入正式用户态 CLI，本阶段允许一个测试专用 harness，建议支持以下子命令：

- `bootstrap --session-dir <dir>`
- `append --session-dir <dir> --holder <holder> --lease-token <token> --lease-version <n> --event-type <type> --producer <producer> --payload-ref <ref>`
- `acquire-lock --session-dir <dir> --holder <holder> --ttl-seconds <n>`
- `renew-lock --session-dir <dir> --holder <holder> --lease-token <token> --lease-version <n> --ttl-seconds <n>`
- `release-lock --session-dir <dir> --holder <holder> --lease-token <token> --lease-version <n>`
- `commit-offset --session-dir <dir> --consumer-id <id> --last-event-id <id> --last-sequence <n>`
- `request-interrupt --session-dir <dir> --scope <scope> --target-role-id <id> --source <src> --reason <text>`
- `ack-interrupt --session-dir <dir> --interrupt-id <id>`
- `clear-interrupt --session-dir <dir> --interrupt-id <id>`
- `list --session-dir <dir>`

### 7.3 黑盒调用方式

推荐统一通过 Python `subprocess.run` 调用 CLI 与 harness：

```python
result = run_cli([
    "run",
    "--config",
    str(case_dir / "octopus.yaml"),
], cwd=case_dir)
```

```python
harness = run_harness([
    "acquire-lock",
    "--session-dir",
    str(session_dir),
    "--holder",
    "orchestrator",
    "--ttl-seconds",
    "30",
], cwd=case_dir)
```

断言只关注：

- `returncode`
- `stdout`
- `stderr`
- 文件系统副作用

## 8. 分阶段执行策略

### 8.1 第一批：bootstrap 黑盒

`event-bus 001` 初版先把最关键的“总线被真正初始化”验证住，聚焦：

- `run` 后 bus 文件不再是占位文本
- `SESSION_CREATED` 事件存在
- `lock.md` / `offsets.md` / `interrupts.md` 有合法初始内容

### 8.2 第二批：harness 驱动的总线行为

在第一批通过后，再补：

- 事件追加与链校验
- lock acquire / renew / release
- offset 提交与回退阻断
- interrupt 请求 / ack / clear

### 8.3 第三批：跨模块联动增强

待 `orchestrator 001`、`role-runtime 001`、`human-gate 001` 落地后，再补以下场景：

1. orchestrator 是否能真实持锁推进调度事件。
2. role runtime 是否能按 offset 增量消费并提交位点。
3. `interrupt-all` 是否能通过正式 CLI 写入 interrupt 事件并驱动角色停下。
4. recovery 是否能在链完整时重放，在链损坏时稳定拒绝恢复。

## 9. 通过标准

以下条件同时满足，视为 `event-bus 001` E2E 达标：

1. `pytest e2e/eventbus -v` 全部通过。
2. `run` 创建出来的 session 目录中，bus 文件全部是可解析协议内容。
3. 事件、锁、offset、interrupt 的关键黑盒场景全部有对应用例。
4. `make check` 已纳入 event-bus 相关测试，不破坏已有检查链路。

## 10. 与后续模块的衔接

- `orchestrator 001` 将直接复用 `lock + append + offset` 这组黑盒验证思路。
- `role-runtime 001` 将直接复用 `list + commit-offset + interrupt projection` 的基础设施。
- `human-gate 001` 将把测试专用 harness 中的 interrupt 动作，替换为正式 CLI 命令。
- `recovery 001` 将基于本阶段的链校验规则补完整回放与 repair 策略。
