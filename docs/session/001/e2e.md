# Session 模块 001 阶段 E2E 方案

## 1. 目标

`session 001` 的 E2E 不验证复杂调度，而是聚焦一个问题：**当 `run` 在合法配置下通过 config 校验后，系统是否会稳定创建出完整、可追溯、无脏数据的 session 目录骨架。**

这批测试要证明四件事：

1. `run` 成功时一定会创建 session。
2. 创建出来的 session 结构、关键文件和初始化内容符合 `docs/session/001/prd.md`。
3. 有效配置快照和初始 checkpoint 会真实落盘。
4. 创建失败时不会留下半初始化目录。

## 2. 验证范围

### 2.1 本阶段必须覆盖

- `run` 成功创建 session 目录。
- session 目录中关键目录、关键文件存在。
- `metadata.md`、`session.state.md`、`timeline.md`、`0000-init.md` 内容一致。
- `state/effective_config.yaml` 落盘成功。
- 连续两次 `run` 生成不同 `session_id`，不覆盖旧目录。
- 自定义 `runtime.workspace.sessions_dir` 时创建路径正确。
- 创建失败时不留下半初始化 session。

### 2.2 本阶段暂不覆盖

- `status` / `resume` / `stop` / `interrupt` / `reset-session` 命令。
- `planner/master_schedule.md` 的业务填充。
- `bus/events.md` 的持续追加与 offset 推进。
- `roles/{role_id}` 子目录与 `turns/*.md` 回合写入。
- checkpoint 增量保存、崩溃恢复、event replay。

## 3. 环境与目录约定

首版仍遵守仓库统一 E2E 规范：默认直接在宿主机执行，不额外引入 Docker。

建议目录结构：

```text
e2e/
├── conftest.py
├── requirements.txt
├── config/
│   └── ...
└── session/
    ├── fixtures/
    │   ├── valid-minimal/
    │   │   └── octopus.yaml
    │   ├── valid-custom-sessions-dir/
    │   │   └── octopus.yaml
    │   └── valid-path-collision/
    │       └── octopus.yaml
    ├── test_run_session_init.py
    └── test_run_session_failure.py
```

额外约束：

- 真实测试工作目录统一使用仓库根目录 `e2e-test/`
- 每次执行前先清理 `e2e-test/`
- `session 001` 的 E2E 不要求真实触发 `codex` CLI 执行链路，但 fixture 仍保持与真实配置模型兼容

### 3.1 宿主机执行约定

执行流程建议如下：

1. 清理 `./e2e-test`
2. 构建当前仓库的 `openoctopus` 二进制
3. 将对应 fixture 复制到 `e2e-test/{case_name}/`
4. 在该目录内执行 `openoctopus run --config ./octopus.yaml`
5. 通过 stdout、退出码和 `.octopus/` 目录内容做黑盒断言

执行命令推荐：

```bash
pytest e2e/session -v
```

## 4. 测试夹具设计

### 4.1 `valid-minimal`

目标：证明最小合法配置可以成功创建 session。

特点：

- 只保留首版必填字段
- 依赖 `config 001` 默认值生成工作目录配置
- 不要求任何 role runtime 真正执行

### 4.2 `valid-custom-sessions-dir`

目标：证明 `runtime.workspace.sessions_dir` 自定义时，session 会创建到指定目录。

特点：

- YAML 内显式设置非默认 `sessions_dir`
- 用例只验证 session 模块对路径解析的尊重，不重新验证 config 模块语义

### 4.3 `valid-path-collision`

目标：证明当 `sessions_dir` 目标路径已被普通文件占用时，`run` 会失败且不留下半初始化目录。

设计方式：

- YAML 中指定一个简单相对路径作为 `sessions_dir`
- 测试启动前在该路径创建同名普通文件
- 断言 `run` 失败，且没有新建任何 session 子目录

## 5. 核心用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| SES-E2E-001 | 最小合法配置创建 session | `valid-minimal/octopus.yaml` | `run` 退出码为 `0`，stdout 含 `session created:` |
| SES-E2E-002 | 创建默认 session 骨架 | `valid-minimal/octopus.yaml` | 创建 `metadata.md`、`session.state.md`、`timeline.md`、`state/effective_config.yaml`、`state/checkpoints/0000-init.md` |
| SES-E2E-003 | 初始化文件内容一致 | `valid-minimal/octopus.yaml` | 多个文件中的 `session_id`、`status`、`workflow_id` 等字段一致 |
| SES-E2E-004 | 有效配置快照落盘 | `valid-minimal/octopus.yaml` | `effective_config.yaml` 存在，且包含默认工作目录相关字段 |
| SES-E2E-005 | 自定义 sessions 目录 | `valid-custom-sessions-dir/octopus.yaml` | session 创建到 fixture 指定目录 |
| SES-E2E-006 | 连续运行不覆盖旧 session | `valid-minimal/octopus.yaml` 连续执行两次 | 生成两个不同 `session_id` 目录 |
| SES-E2E-007 | 路径冲突时回滚 | `valid-path-collision/octopus.yaml` + 预创建同名文件 | `run` 非 `0`，无半初始化 session 目录 |

## 6. 关键断言方式

### 6.1 `run` 成功断言

- 进程退出码为 `0`
- stdout 至少包含 `session created:`
- 能从 stdout 解析出真实 `session_dir`
- `session_dir` 在文件系统真实存在

### 6.2 骨架断言

必须断言以下路径存在：

- `metadata.md`
- `session.state.md`
- `timeline.md`
- `planner/`
- `bus/`
- `roles/`
- `artifacts/index.md`
- `state/effective_config.yaml`
- `state/checkpoints/0000-init.md`
- `audit/lineage.md`
- `audit/replay.md`

### 6.3 内容一致性断言

至少验证：

- `metadata.md` 与 `session.state.md` 的 `session_id` 一致
- `metadata.md` 的 `status` 与 `session.state.md` 的 `status` 都是 `INITIAL`
- `timeline.md` 首条记录包含 `SESSION_CREATED`
- `0000-init.md` 引用了 `effective_config.yaml`

### 6.4 失败回滚断言

- 进程退出码非 `0`
- 目标 `sessions_dir` 下不存在新建 session 子目录
- 不存在只创建了一半的 `metadata.md` 或 `session.state.md`

## 7. 推荐测试实现

### 7.1 复用现有 `pytest` 基础设施

建议继续复用 `e2e/conftest.py` 的基础能力：

- `project_root`
- `binary_path`
- `workspace_dir`
- `run_cli(command: list[str], env: dict[str, str] | None = None)`
- `clean_environment()`

在 `session` 模块额外补充两个辅助函数即可：

- `parse_session_dir(stdout: str) -> Path`
- `assert_session_skeleton(session_dir: Path)`

### 7.2 黑盒调用方式

推荐统一通过 Python `subprocess.run` 黑盒执行：

```python
result = run_cli([
    binary_path,
    "run",
    "--config",
    str(case_dir / "octopus.yaml"),
])
```

断言只关注：

- `returncode`
- `stdout`
- `stderr`
- 文件系统副作用

不读取内部 Go 结构体，不直接调用 `internal/session` 私有函数。

## 8. 分阶段执行策略

### 8.1 第一批：session 单模块黑盒

`session 001` 完成后立即执行这批测试，聚焦：

- `run` 是否创建标准 session 骨架
- 有效配置快照是否落盘
- 失败回滚是否干净
- 连续运行是否能稳定生成新 session

### 8.2 第二批：与后续模块联动增强

待 `event-bus 001`、`orchestrator 001`、`role-runtime 001` 落地后，再补充以下联动场景：

1. `bus/events.md` 是否在 session 创建后继续追加真实事件。
2. `planner/master_schedule.md` 是否从占位文件变为真实调度文件。
3. `roles/{role_id}` 是否由运行时按角色创建并写入 turns。
4. `resume` / `interrupt-all` / `reset-session` 是否能复用 `session 001` 创建的基线目录。

## 9. 通过标准

以下条件同时满足，才视为 `session 001` 的 E2E 方案达标：

1. 核心成功场景可以稳定创建完整 session 骨架。
2. 有效配置快照与初始 checkpoint 可以稳定复现。
3. 连续两次执行不会覆盖已有 session。
4. 失败场景不会留下半初始化 session 目录。
5. 测试断言只依赖 CLI 输出与文件系统结果，不依赖内部实现细节。

## 10. 与后续模块的衔接

`session 001` 的 E2E 要证明的是“**会话载体已经可信**”，后续模块再在这个载体上叠加业务行为：

- `event-bus 001`：验证事件能否按 session 协议持续落盘。
- `orchestrator 001`：验证主控能否基于现有 `planner/` 与 `session.state.md` 推进调度。
- `role-runtime 001`：验证角色目录、turn 文件和结论回传能否挂到现有 session 结构上。
- `recovery 001`：验证 `effective_config.yaml + checkpoints` 能否支撑恢复。

因此，`session` 的 E2E 不追求现在就把全链路测完，而是先把“session 基座是否稳定”做实。
