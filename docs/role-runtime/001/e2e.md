# Role Runtime 模块 001 阶段 E2E 设计

## 1. 目标

`role-runtime 001` 的 E2E 要回答的问题只有一个：**当 `run` 已经完成 config 校验、session 初始化、event-bus bootstrap 与 orchestrator 首轮分发后，系统是否会真实驱动角色执行、写出 turn 文件、生成结论，并把工作流继续推进到下一状态。**

这组 E2E 不验证内部函数实现细节，只通过真实 CLI、真实文件系统副作用、真实宿主机 `codex` 环境与测试专用 harness 做黑盒断言。

E2E 的价值不只是证明“角色会跑”，更要证明以下协议已经闭环：

1. 角色目录文件会被正确 bootstrap。
2. 回合输入输出会被完整落盘。
3. 结论文件与 orchestrator 协议兼容。
4. interrupt / reset / offset 幂等规则成立。
5. 真实 `codex` CLI 至少能跑通一个最小 smoke 用例。

## 2. 验证范围

### 2.1 范围内

- `run` 触发 orchestrator 分发后，role-runtime 会真实执行已分发角色。
- `roles/{role_id}/state.md`、`session.reset.md`、`outbox.md`、`conclusion.md`、`heartbeat.md`、`events.md`、`turns/` 会按协议写出。
- `turns/NNNN-input.md` / `turns/NNNN-output.md` 会记录关键元信息、执行命令、退出码和结构化结果。
- deterministic executor 能稳定覆盖 `SUCCESS / NEEDS_RETRY / BLOCKED / FAILED` 及重复 tick、reset、interrupt 等守卫场景。
- 真实 `codex` executor 至少覆盖一个 smoke case，直接复用宿主机 `~/.codex/`。
- role-runtime 会消费 / 提交 event-bus offset，并能避免重复执行同一任务包。
- role-runtime 处理 interrupt / reset 后，orchestrator 下一轮 tick 能观察到正确结论或等待态。

### 2.2 范围外

- `claude_code` 真实执行链路。
- tmux pane、常驻 watcher、实时流式日志展示。
- artifact 索引、版本化和差异检查。
- 多角色真正并发子进程执行。
- Docker 化 E2E；本阶段默认宿主机直跑。

## 3. 环境与目录约定

### 3.1 宿主机执行约定

所有 `role-runtime 001` E2E 默认直接在宿主机运行，原因如下：

1. 真实 `codex` CLI 已依赖当前用户本机登录态。
2. 本阶段验证重点是文件协议与 CLI 适配，不是容器编排。
3. 复用仓库现有 `e2e/conftest.py` 可以保持各模块测试风格一致。

执行流程建议如下：

1. 清理 `./e2e-test`
2. 构建当前仓库 `openoctopus` 二进制
3. 构建 `e2e/role-runtime/harness` 测试二进制
4. 视场景复用 `e2e/eventbus/harness` 注入 interrupt
5. 复制 fixture 到 `e2e-test/role-runtime/{case_name}/`
6. 先执行 `openoctopus run --config ./octopus.yaml`
7. 如需额外驱动 role-runtime 边界场景，再调用 role-runtime harness / eventbus harness
8. 通过 stdout、退出码和 session 目录文件做断言

执行命令推荐：

```bash
python3 -m pytest e2e/role-runtime -v
```

### 3.2 推荐测试目录

后续实现建议采用以下目录结构：

```text
e2e/role-runtime/
├── fixtures/
│   ├── valid-deterministic-success/
│   │   └── octopus.yaml
│   ├── valid-retry-once/
│   │   └── octopus.yaml
│   ├── valid-blocked/
│   │   └── octopus.yaml
│   ├── valid-interrupt-before-start/
│   │   └── octopus.yaml
│   ├── valid-reset-generation/
│   │   └── octopus.yaml
│   └── valid-codex-smoke/
│       └── octopus.yaml
├── harness/
│   └── main.go
├── test_role_runtime_bootstrap.py
├── test_role_runtime_tick.py
└── test_role_runtime_guards.py
```

这是**推荐实现目录**，不是当前仓库已存在目录。实际开始实现时，目录结构必须与仓库真实情况同步创建。

### 3.3 真实 Codex 环境约束

只要 fixture 里出现以下任一条件，就必须把该用例视为真实 Codex E2E：

- `llm_profiles.*.provider: codex`
- `llm_profiles.*.cli_path: codex`
- role-runtime 真实调用宿主机 `codex` CLI

这类测试必须：

- 复用当前用户真实 `~/.codex/`
- 通过 `e2e/conftest.py` 的 `real_codex_env` 注入 `CODEX_HOME`
- 禁止伪造 token
- 禁止用假的 `CODEX_HOME` 目录替代真实目录

## 4. 测试夹具设计

### 4.1 `valid-deterministic-success`

目标：证明最小合法配置在 deterministic executor 下，可以从 orchestrator 分发继续推进到角色真实执行，并最终完成工作流。

特点：

- 只有一个 stage、一个 role
- role 使用 deterministic executor 返回 `SUCCESS`
- `run` 结束后应同时具备 `turns/0001-input.md`、`turns/0001-output.md`、`conclusion.md` 和 `session.state.md = COMPLETED`

### 4.2 `valid-retry-once`

目标：证明 role-runtime 产出 `NEEDS_RETRY` 后，orchestrator 能重发同一 stage，runtime 第二轮能继续执行并成功收口。

特点：

- deterministic executor 第一次返回 `NEEDS_RETRY`
- orchestrator 下一轮会重写 `inbox.md`
- runtime 第二次执行后返回 `SUCCESS`
- 最终应生成 `turns/0001-*` 与 `turns/0002-*`

### 4.3 `valid-blocked`

目标：证明 role-runtime 产出 `BLOCKED` 后，系统会正确进入等待人工状态，而不是继续盲目运行。

特点：

- deterministic executor 返回 `BLOCKED`
- `conclusion.md.status = BLOCKED`
- `planner/blockers.md` 被更新
- `session.state.md.status = WAITING_HUMAN`

### 4.4 `valid-interrupt-before-start`

目标：证明当 interrupt 在 runtime 真正启动 turn 之前到达时，角色不会误开新 turn，而是先确认 interrupt 并停住。

特点：

- 先通过 `run` 完成 session 创建与任务分发
- 再用 `eventbus-harness` 请求一个命中该角色的 interrupt
- 最后调用 role-runtime harness 的 `tick-role`
- 断言没有生成新的 `turns/0001-output.md`，且 `state.md.status = INTERRUPTED`

### 4.5 `valid-reset-generation`

目标：证明 role-runtime 应用 reset 请求时，会递增 `session_generation`，但不会删除历史回合文件。

特点：

- 先执行至少一轮成功 turn
- 再用 role-runtime harness 写入 reset 请求
- 下一次 tick 应应用 reset，`session_generation` 递增
- `turns/0001-*` 仍保留

### 4.6 `valid-codex-smoke`

目标：证明真实宿主机 `codex` CLI 能在最小合法任务中跑通 role-runtime 首版闭环。

特点：

- role 使用 `provider: codex`
- prompt 尽量简短，只要求输出标准化 `role_result` 块和一句极简摘要
- 断言不依赖自然语言细节，只验证：退出码、turn 文件存在、`conclusion.md.status = SUCCESS`、workflow 能完成

## 5. 核心用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| RRT-E2E-001 | `run` 触发 role-runtime 最小闭环 | `valid-deterministic-success/octopus.yaml` | `run` 退出码为 `0`，workflow 最终为 `COMPLETED` |
| RRT-E2E-002 | 角色目录 bootstrap | `valid-deterministic-success/octopus.yaml` | `roles/agent_a/state.md`、`session.reset.md`、`outbox.md`、`conclusion.md`、`heartbeat.md`、`events.md`、`turns/` 存在 |
| RRT-E2E-003 | 首轮 turn 文件生成 | `valid-deterministic-success/octopus.yaml` | `turns/0001-input.md` 与 `turns/0001-output.md` 存在且字段一致 |
| RRT-E2E-004 | retry 二次执行 | `valid-retry-once/octopus.yaml` | 生成 `turns/0001-*` 与 `turns/0002-*`，最终 `conclusion.md.status = SUCCESS` |
| RRT-E2E-005 | blocked 进入等待人工 | `valid-blocked/octopus.yaml` | `conclusion.md.status = BLOCKED`，`session.state.md.status = WAITING_HUMAN` |
| RRT-E2E-006 | 重复 tick 幂等 | `valid-deterministic-success` + 再次 `tick-role` | 不新增 `turns/0002-*`，`outbox_version` / `turn_seq` 不重复增长 |
| RRT-E2E-007 | interrupt 先于执行到达 | `valid-interrupt-before-start` + harness request-interrupt + `tick-role` | 不启动新 turn，`state.md.status = INTERRUPTED` |
| RRT-E2E-008 | reset 递增代际且保留历史 | `valid-reset-generation` + harness write-reset + `tick-role` | `session_generation` 前进，`turns/0001-*` 仍保留 |
| RRT-E2E-009 | 真实 Codex smoke | `valid-codex-smoke/octopus.yaml` | 真实 `codex` 执行成功，workflow 进入 `COMPLETED` |

## 6. 关键断言方式

### 6.1 `run` 最小闭环断言

- 进程退出码为 `0`
- stdout 至少包含 `session created:`
- 能从 stdout 解析出真实 `session_dir`
- `session.state.md.status` 最终与用例预期一致（`COMPLETED` / `WAITING_HUMAN`）

### 6.2 角色目录 bootstrap 断言

至少验证：

- `roles/{role_id}/state.md` 存在且不再是 placeholder
- `roles/{role_id}/session.reset.md` 存在且包含 `session_generation`
- `roles/{role_id}/heartbeat.md` 含 `last_seen_at` / `expire_at`
- `roles/{role_id}/events.md` 至少包含一条角色级事件
- `roles/{role_id}/turns/` 目录真实存在

### 6.3 turn 文件断言

至少验证：

- `turns/0001-input.md` 包含 `task_id`、`stage_id`、`role_id`、`context_version`、`inbox_version`
- `turns/0001-output.md` 包含 `executor_provider`、`exit_code`、`duration_ms`
- 输入输出中的 `turn_seq` 一致
- 第二轮执行时文件名严格递增为 `0002-*`

### 6.4 结论与回执断言

- `conclusion.md` 中的 `task_id`、`stage_id`、`role_id` 与 `inbox.md` 一致
- `conclusion.md.status` 只允许为 `SUCCESS / NEEDS_RETRY / BLOCKED / FAILED`
- `outbox.md.conclusion_ref` 指向当前 `conclusion.md`
- `outbox.md.turn_output_ref` 指向当前 `turns/NNNN-output.md`

### 6.5 幂等与 offset 断言

- 对同一 `task_id + inbox_version + context_version + session_generation` 再次 `tick-role`，不会生成新的 turn 文件
- `events.md` 不会重复写同一条“accepted / started / completed”事件
- bus offsets 中 `consumer_id = role-runtime/{role_id}` 会向前推进，但不会回退

### 6.6 interrupt 与 reset 断言

- interrupt 在执行前到达时，不应创建新的输出 turn
- interrupt 被处理后，`state.md.status = INTERRUPTED`
- reset 应递增 `session_generation`
- reset 不删除历史 `turns/*.md`
- reset 之后若无新分发任务，再次 `tick-role` 不应误执行旧任务

### 6.7 真实 Codex smoke 断言

- 使用真实 `CODEX_HOME`
- `turns/0001-output.md` 中 `executor_provider = codex`
- `conclusion.md.status = SUCCESS`
- 不断言自然语言摘要的具体措辞，只断言结构和状态

## 7. 推荐测试实现

### 7.1 复用现有 `pytest` 基础设施

建议继续复用 `e2e/conftest.py` 的基础能力：

- `project_root`
- `binary_path`
- `workspace_dir`
- `real_codex_env`
- `run_cli(command: list[str], env: dict[str, str] | None = None)`
- `prepare_module_case(...)`
- `eventbus_harness_path` / `run_harness(...)`

对 `role-runtime` 模块额外补充：

- `role_runtime_harness_path`
- `run_role_runtime_harness(args: list[str], cwd: Path)`
- `parse_session_dir(stdout: str) -> Path`
- `read_text(path: Path) -> str`

### 7.2 Harness 子命令建议

为避免过早暴露正式用户态 CLI，本阶段允许一个测试专用 harness，建议支持以下子命令：

- `tick-role --session-dir <dir> --role-id <id>`
- `tick-all --session-dir <dir>`
- `write-reset --session-dir <dir> --role-id <id> --reason <text> --requested-by <src>`
- `read-role-state --session-dir <dir> --role-id <id>`
- `read-role-summary --session-dir <dir> --role-id <id>`

interrupt 注入可以直接复用现有 `eventbus-harness`，无需在 role-runtime harness 里重复造一套总线注入逻辑。

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
tick = run_role_runtime_harness([
    "tick-role",
    "--session-dir",
    str(session_dir),
    "--role-id",
    "agent_a",
], cwd=case_dir)
```

```python
interrupt = run_harness([
    "request-interrupt",
    "--session-dir",
    str(session_dir),
    "--scope",
    "role",
    "--target-role-id",
    "agent_a",
    "--source",
    "e2e",
    "--reason",
    "stop before execution",
], cwd=case_dir)
```

断言只关注：

- `returncode`
- `stdout`
- `stderr`
- 文件系统副作用

## 8. 分阶段执行策略

### 8.1 第一批：deterministic 最小闭环

先锁定最关键的协议面：

- role 目录文件真实创建
- turn 输入输出真实生成
- `conclusion.md` 与 orchestrator 收口真实联通
- 重复 tick 不会重复执行

这一批应优先覆盖 `valid-deterministic-success` 与 `valid-retry-once`。

### 8.2 第二批：interrupt / reset / 幂等守卫

在最小闭环稳定后，再补：

- interrupt 在执行前的停住能力
- reset 的 generation 递增与历史保留
- blocked / failed 路径的文件与 workflow 状态一致性

这一批建议覆盖 `valid-blocked`、`valid-interrupt-before-start`、`valid-reset-generation`。

### 8.3 第三批：真实 Codex smoke

最后再补真实 `codex` smoke，用最小任务验证：

- 宿主机 `~/.codex/` 被正确复用
- role-runtime 真实调用 `codex`
- 真实 turn 与 conclusion 文件按协议写出

这一步只做 smoke，不在 `001` 追求复杂提示词、多轮推理或多角色真实并行。

## 9. 与上游模块的联动要求

- `config 001`：需保证 `llm_profile`、超时、只读约束、must-read files 等字段可以被 role-runtime 消费。
- `session 001`：需保证 session 根目录、`roles/`、`artifacts/`、`state/effective_config.yaml` 已存在。
- `event-bus 001`：需保证 offsets / interrupts / event list 能被 role-runtime 直接复用。
- `orchestrator 001`：需保证 `context.md` / `inbox.md` 字段稳定，且 `conclusion.md` 消费契约不再频繁变更。

## 10. 完成标准

满足以下条件，才视为 `role-runtime 001` 的 E2E 设计达标：

1. 已覆盖 deterministic success / retry / blocked / interrupt / reset / idempotent / real codex smoke 这些关键链路。
2. 每个用例都只依赖 CLI、harness 与文件系统断言，不直接调用 Go 私有函数。
3. 至少一个用例明确复用宿主机真实 `~/.codex/` 并成功跑通。
4. 文档中的推荐测试目录、fixture 设计、命令方式与仓库现有 E2E 习惯保持一致。
5. 后续真正实现时，可以直接据本文创建 `e2e/role-runtime/` 目录并补测试代码，而不需要再次推翻协议。
