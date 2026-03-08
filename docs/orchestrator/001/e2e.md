# Orchestrator 模块 001 阶段 E2E 方案

## 1. 目标

`orchestrator 001` 的 E2E 不验证真实子 Agent 执行，而是聚焦一个问题：**当 `run` 完成 config 校验、session 初始化和 event-bus bootstrap 后，系统是否会真实生成 planner 当前态、首批角色任务包，并能在后续 tick 中根据 `conclusion.md` 推进工作流。**

这批测试要证明六件事：

1. `run` 成功时一定会 bootstrap orchestrator，而不是只创建 session。
2. `planner/` 中关键业务文件会被真实生成并保持一致。
3. ready stage 会被分发为角色 `context.md` 与 `inbox.md`。
4. 写入 synthetic `conclusion.md` 后，主控下一轮 tick 能推进完成、重试或阻塞状态。
5. 新的 `human_messages.md` 输入会被 requirement snapshot 吸收并推进游标。
6. 全链路仍然遵守宿主机黑盒测试原则，只通过 CLI / harness / 文件系统副作用验证。

## 2. 验证范围

### 2.1 本阶段必须覆盖

- `run` 成功后 orchestrator 自动 bootstrap。
- `master_schedule.md`、`task_board.md`、`task_graph.mmd`、`global_progress.md`、`dispatch_log.md`、`decision_log.md` 真实存在。
- entry stage 会被分发，`roles/{role_id}/context.md` 与 `roles/{role_id}/inbox.md` 内容合法。
- synthetic `conclusion.md` 写为 `SUCCESS` 后，下一轮 tick 能推进到 `COMPLETED` 或下一个 stage。
- synthetic `conclusion.md` 写为 `NEEDS_RETRY` 后，下一轮 tick 能递增 attempt 并重发任务。
- synthetic `conclusion.md` 写为 `BLOCKED` 后，会写 `blockers.md` 并把 session 状态推进到 `WAITING_HUMAN`。
- 新增 `human_messages.md` 输入后，`requirement.snapshot.md` 的 `snapshot_version` 与 `human_message_cursor` 会更新。
- `max_parallel_roles` 生效，多个 entry stage 不会超出并发限制。

### 2.2 本阶段暂不覆盖

- 真实 `role-runtime` CLI 执行器启动。
- `outbox.md`、`turns/*.md`、`heartbeat.md` 的真实写入。
- `interrupt-all`、`approval`、`reroute`、`reset-session` 等用户态命令。
- 复杂条件表达式、聚合流转、join stage。
- recovery 重建和跨进程并发 orchestrator 竞争。

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
├── eventbus/
│   └── ...
└── orchestrator/
    ├── fixtures/
    │   ├── valid-minimal/
    │   │   └── octopus.yaml
    │   ├── valid-two-entry/
    │   │   └── octopus.yaml
    │   └── valid-human-message/
    │       └── octopus.yaml
    ├── harness/
    │   └── main.go
    ├── test_run_bootstrap.py
    ├── test_orchestrator_tick.py
    └── test_orchestrator_guards.py
```

额外约束：

- 真实测试工作目录统一使用仓库根目录 `e2e-test/`
- 每次执行前先清理 `e2e-test/`
- orchestrator harness 只允许调用 `internal/orchestrator` 的公开服务，不直接拼写 planner Markdown 协议细节

### 3.1 宿主机执行约定

执行流程建议如下：

1. 清理 `./e2e-test`
2. 构建当前仓库的 `openoctopus` 二进制
3. 构建测试专用 `orchestrator-harness` 二进制
4. 将 fixture 复制到 `e2e-test/orchestrator/{case_name}/`
5. 在该目录执行 `openoctopus run --config ./octopus.yaml`
6. 通过 stdout、退出码和 `.octopus/` 目录内容做黑盒断言
7. 如需后续推进，调用 `orchestrator-harness tick --session-dir <dir>` 或辅助子命令模拟人工输入 / 角色结论

执行命令推荐：

```bash
python3 -m pytest e2e/orchestrator -v
```

## 4. 测试夹具设计

### 4.1 `valid-minimal`

目标：证明最小合法配置经过 `run` 后，可以完成 orchestrator bootstrap，并分发首个 stage。

特点：

- 只有一个 stage，一个 role
- 依赖 `config 001` 默认值、`session 001` 骨架和 `event-bus 001` bootstrap
- 后续 success / retry / blocked 用例都可以复用这个 fixture，只通过 harness 写不同的 `conclusion.md`

### 4.2 `valid-two-entry`

目标：证明多个 entry stage 存在时，orchestrator 会受 `max_parallel_roles` 控制，按并发限制分发而不是一次性全发。

特点：

- 两个 entry stage 指向两个不同 role
- YAML 中显式设置 `runtime.scheduler.max_parallel_roles: 1`
- 用例只验证分发节流，不验证复杂流转

### 4.3 `valid-human-message`

目标：证明新的人工输入会被 requirement snapshot 吸收，而不是停留在原始日志里无人消费。

特点：

- 以最小单 stage 流程为基础
- `run` 后由 harness 追加一条人类消息
- 再执行一次 `tick`，验证 `snapshot_version` 和 `human_message_cursor` 前进

## 5. 核心用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| ORC-E2E-001 | `run` 触发 orchestrator bootstrap | `valid-minimal/octopus.yaml` | `run` 退出码为 `0`，stdout 含 `session created:` |
| ORC-E2E-002 | planner 文件生成 | `valid-minimal/octopus.yaml` | `master_schedule.md`、`task_board.md`、`task_graph.mmd`、`global_progress.md`、`dispatch_log.md`、`decision_log.md` 存在 |
| ORC-E2E-003 | 首批任务分发 | `valid-minimal/octopus.yaml` | `roles/agent_a/context.md` 与 `roles/agent_a/inbox.md` 存在，且 `task_id`、`stage_id` 一致 |
| ORC-E2E-004 | 成功结论推进完成 | `valid-minimal` + harness `write-conclusion SUCCESS` + `tick` | `master_schedule.md` 标记 `COMPLETED`，`session.state.md` 为 `COMPLETED` |
| ORC-E2E-005 | 重试结论触发重发 | `valid-minimal` + harness `write-conclusion NEEDS_RETRY` + `tick` | 同一 stage `attempt` 递增，`inbox.md` 任务版本更新 |
| ORC-E2E-006 | 阻塞结论触发等待人工 | `valid-minimal` + harness `write-conclusion BLOCKED` + `tick` | `blockers.md` 写入原因，`session.state.md` 为 `WAITING_HUMAN` |
| ORC-E2E-007 | 人工输入游标推进 | `valid-human-message` + harness `append-human-message` + `tick` | `requirement.snapshot.md` 的 `snapshot_version` 和 `human_message_cursor` 更新 |
| ORC-E2E-008 | 并发限制生效 | `valid-two-entry/octopus.yaml` | 两个 entry stage 中最多一个进入 `DISPATCHED` |

## 6. 关键断言方式

### 6.1 `run` 成功断言

- 进程退出码为 `0`
- stdout 至少包含 `session created:`
- 能从 stdout 解析出真实 `session_dir`
- `session_dir/planner/` 和 `session_dir/roles/` 真实存在

### 6.2 planner bootstrap 断言

至少验证：

- `planner/master_schedule.md` 不再包含 `Initialized by session 001.`
- `planner/requirement.snapshot.md` 含 `snapshot_version`
- `planner/global_progress.md` 含当前 workflow 状态摘要
- `planner/task_board.md` 与 `planner/task_graph.mmd` 已被创建
- `planner/dispatch_log.md` 与 `planner/decision_log.md` 是合法文档，而不是空白文件

### 6.3 任务分发断言

至少验证：

- `roles/{role_id}/context.md` 与 `inbox.md` 真实存在
- 两个文件中的 `task_id`、`stage_id`、`role_id` 一致
- `context_version` 与 `inbox_version` 是正整数或可递增值
- bus 中存在对应 `TASK_DISPATCHED` 事件

### 6.4 结论推进断言

- `SUCCESS` 后 stage 状态推进为 `COMPLETED`
- `NEEDS_RETRY` 后 `attempt` 递增且重新分发
- `BLOCKED` 后 `blockers.md` 包含 summary，`session.state.md` 为 `WAITING_HUMAN`
- `FAILED`（若补充测试）后工作流进入 `FAILED`

### 6.5 人工输入断言

- 新增消息后，`requirement.snapshot.md` 的 `source_message_count` 增加
- `human_message_cursor` 指向最新 `message_id`
- 若没有新增消息，再次 tick 不应重复增长 `snapshot_version`

## 7. 推荐测试实现

### 7.1 复用现有 `pytest` 基础设施

建议继续复用 `e2e/conftest.py` 的基础能力：

- `project_root`
- `binary_path`
- `workspace_dir`
- `run_cli(command: list[str], env: dict[str, str] | None = None)`
- `prepare_module_case(...)`
- `eventbus_harness_path`（若需要辅助验证 bus 事件）

对 `orchestrator` 模块额外补充：

- `orchestrator_harness_path`
- `run_orchestrator_harness(args: list[str], cwd: Path)`
- `parse_session_dir(stdout: str) -> Path`
- `read_text(path: Path) -> str`

### 7.2 Harness 子命令建议

为避免过早引入正式用户态 CLI，本阶段允许一个测试专用 harness，建议支持以下子命令：

- `tick --session-dir <dir>`
- `append-human-message --session-dir <dir> --source <src> --message <text>`
- `write-conclusion --session-dir <dir> --role-id <id> --stage-id <id> --task-id <id> --status <status> --summary <text>`
- `read-schedule --session-dir <dir>`
- `read-progress --session-dir <dir>`

这些子命令只允许调用 orchestrator 公开服务或写测试专用 synthetic 输入文件，不允许在 Python 侧直接手改 `master_schedule.md`。

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
harness = run_orchestrator_harness([
    "write-conclusion",
    "--session-dir",
    str(session_dir),
    "--role-id",
    "agent_a",
    "--stage-id",
    "stage_a",
    "--task-id",
    "task-stage_a-01",
    "--status",
    "SUCCESS",
    "--summary",
    "stage done",
], cwd=case_dir)
```

断言只关注：

- `returncode`
- `stdout`
- `stderr`
- 文件系统副作用

## 8. 分阶段执行策略

### 8.1 第一批：bootstrap 与首批分发

先把最关键的“主控真的启动了”锁住，聚焦：

- `run` 后 planner 文件不再是 placeholder
- `master_schedule.md` 与 `task_board.md` 已生成
- 首个 role 的 `context.md` 与 `inbox.md` 已写出

### 8.2 第二批：结论驱动推进

在第一批通过后，再补：

- `SUCCESS` 推进到完成
- `NEEDS_RETRY` 触发重复分发
- `BLOCKED` 触发 `WAITING_HUMAN`

### 8.3 第三批：人工输入与守卫

最后补齐：

- `human_messages.md` 输入吸收
- `max_parallel_roles` 节流
- 无新增输入时 snapshot 不重复增长

## 9. 与上游模块的联动要求

- `config 001`：fixture 必须继续走真实 YAML 校验链路
- `session 001`：必须继续依赖真实 session 目录，不允许绕过 `run` 手工伪造目录
- `event-bus 001`：orchestrator E2E 应至少验证一次 bus 中确实存在 `TASK_DISPATCHED` 或同等事件

## 10. 完成标准

当以下条件同时满足时，才认为 `orchestrator 001` E2E 达标：

1. `pytest e2e/orchestrator -v` 可稳定运行
2. `make check` 已纳入 orchestrator E2E
3. 所有断言都通过 CLI / harness / 文件系统副作用完成，不直接调用 Go 私有函数
4. 测试结果能解释“主控已启动、已分发、能推进、能阻塞、能吸收人工输入”五类关键能力
