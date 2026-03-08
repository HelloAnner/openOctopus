# TMUX 模块 001 PRD

## 1. 目标

`tmux 001` 的目标不是马上把 OpenOctopus 做成完整的交互式终端控制台，而是先把 **真实 tmux 会话编排闭环** 做实：`run` 能按 session 创建独立 tmux server / session，生成“左主右子”的最小 pane 布局，给每个 role 分配稳定 pane，并把布局元数据落到 session 目录，供 CLI `switch` 与后续调试能力复用。

这一版要解决的是“有没有稳定、可追踪、可切换的 tmux 运行载体”，而不是“tmux 内是否已经跑着完整常驻子进程”。

## 2. 首版定位

### 2.1 要解决的问题

当前仓库已经具备 `run`、`status`、`interrupt`、`resume`、`recover` 等 CLI 主链路，也已经把 session、event-bus、orchestrator、role-runtime、人机介入等模块跑通了最小闭环；但 `tmux` 仍然停留在模块总纲，没有真正落地。结果是：

1. session 虽然已经存在，但用户缺少一个稳定的多 pane 观察入口。
2. role 与 pane 之间没有正式绑定协议，`switch` 无法落地。
3. 黑盒 E2E 无法真实验证“tmux 会话是否被创建、pane 是否按 role 编排、pane 是否可捕获”。

`tmux 001` 就是为了解决这三个问题。

### 2.2 本阶段成功标准

满足以下条件，即认为 `tmux 001` 达标：

1. `runtime.tmux` 配置可以被加载、默认值注入和校验。
2. `openoctopus run` 在 `runtime.tmux.enabled=true` 时，能真实创建独立 tmux socket / session。
3. tmux 会话具备最小“左主右子”布局，右侧 role pane 数量与 role 数量一致。
4. session 内存在正式 `state/tmux/layout.md`，记录 socket、session、main pane、role pane 映射。
5. 新增 `openoctopus switch` 命令，能基于 session + role 解析目标 pane；若当前命令运行在 tmux 客户端内，则直接切换；若不在 tmux 内，则稳定输出目标信息。
6. E2E 能黑盒验证 tmux 布局、pane 标题、pane 内容捕获与 `switch` 返回结果。

## 3. 设计原则

### 3.1 第一性原则

- 直接复用系统已安装的 `tmux`，不自建终端仿真层。
- 保持 tmux 只负责“运行视图与 pane 绑定”，不让它反向侵入 orchestrator / role-runtime 的状态决策。
- 所有关键运行结果继续落 Markdown 文件，tmux 只是显示和定位载体。

### 3.2 001 的保守范围

001 只实现以下最小能力：

1. `run` 时创建 detached tmux session。
2. 基于 role 数量编排 pane。
3. 为主 pane 与 role pane 设置稳定标题。
4. 把 layout 元数据写入 `state/tmux/layout.md`。
5. 提供 `switch` 命令与 pane 捕获底层能力。
6. 在交互式 `run` 场景下，为支持的 role pane 自动拉起对应的交互式 CLI 待命页面，并默认聚焦首个 role pane。

### 3.3 001 明确不做

本阶段不做：

1. 不实现 tmux 内常驻 watcher / daemon。
2. 不实现 Bubble Tea / Lip Gloss 交互式状态面板。
3. 不把 orchestrator 或 role-runtime 的执行流重定向进 tmux pane；role pane 中的交互式 CLI 仅作为人工协作入口，不承担正式状态推进。
4. 不实现复杂的 pane 动态重排、拖拽、窗口组同步。
5. 不实现 Web / tmux 跨端联动。

## 4. 范围与边界

### 4.1 输入

- `runtime.tmux.enabled`
- `runtime.tmux.socket_name`
- `runtime.tmux.main_pane_ratio`
- `runtime.tmux.role_layout`
- session id
- roles 列表

### 4.2 输出

- 独立 tmux socket
- 独立 tmux session
- 主 pane 与 role pane 布局
- `state/tmux/layout.md`
- `switch` 命令的可消费结果

### 4.3 模块边界

`tmux` 模块负责：

1. 创建 tmux session 与 pane。
2. 为 pane 命名并维护 role -> pane 映射。
3. 暴露 `switch` 与 pane capture 所需底层能力。

`tmux` 模块不负责：

1. session 生命周期创建。
2. role 的真实执行。
3. 业务状态推进。
4. 人工打断协议。

## 5. 配置模型

## 5.1 新增 `runtime.tmux`

```yaml
runtime:
  tmux:
    enabled: false
    socket_name: "octopus-{session_id}"
    main_pane_ratio: 0.5
    role_layout: "adaptive_grid"
```

### 5.2 配置语义

| 字段 | 含义 | 001 规则 |
| --- | --- | --- |
| `enabled` | 是否启用 tmux 编排 | 默认 `false`，只有显式开启才创建 tmux |
| `socket_name` | tmux socket 名称模板 | 支持 `{session_id}` 占位符 |
| `main_pane_ratio` | 主 pane 宽度占比 | 必须大于 `0`，建议小于 `1` |
| `role_layout` | 角色区布局策略 | 001 只支持 `adaptive_grid`、`tiled` |

### 5.3 默认值策略

采用保守默认值：

1. `enabled=false`：避免把当前所有 `run` 默认拖进对 tmux 的宿主机依赖。
2. `socket_name=octopus-{session_id}`：保证同一 session 的 tmux socket 可追踪且天然隔离。
3. `main_pane_ratio=0.5`：与总 PRD 保持一致。
4. `role_layout=adaptive_grid`：作为首版推荐值。

## 6. 启动与编排流程

### 6.1 `run` 接入点

当 `openoctopus run` 成功通过配置加载与校验，并完成 session 工作目录初始化后，`tmux 001` 在 orchestrator bootstrap 之前接入：

```text
加载配置
  -> 注入默认值与校验
  -> 创建 session 目录
  -> 如果 tmux.enabled=true，则创建 tmux session 与 layout.md
  -> bootstrap event-bus / artifact / orchestrator
  -> 返回 session created
```

选择这个位置的原因：

1. session id 已经产生，可以安全展开 `socket_name`。
2. 如果 tmux 创建失败，可以和当前 `run` 的失败清理逻辑一起回滚。
3. tmux 不需要提前感知 orchestrator 内部状态，只需要会话和 role 列表。

### 6.2 pane 布局策略

001 采用 **tmux 原生 `main-vertical` 为主、右侧 role pane 自动平铺** 的策略，而不是自己计算复杂几何坐标。

实际步骤：

1. 先创建 detached session，得到主 pane。
2. 第一个 role 先通过水平拆分创建在右侧。
3. 其余 role 继续在角色区补 pane。
4. 最终统一应用 `main-vertical` 布局，让主 pane 固定在左侧，角色区在右侧自动铺开。
5. 用 `main_pane_ratio` 二次调整主 pane 宽度。

这样做的原因是：

- 能稳定满足“左主右子”。
- 直接复用 tmux 内置布局算法，减少自己维护 pane 坐标的复杂度。
- role 数量变化时仍保持可读。

### 6.3 role_layout 语义

001 阶段两种布局值的差别收敛到“创建 pane 的顺序策略”上：

1. `adaptive_grid`：优先按右侧自动平铺阅读性布局，适合作为默认值。
2. `tiled`：仍保留左主右子大结构，但角色区更偏平均分配。

首版不追求把两者做成完全不同的复杂几何算法，重点是让配置字段、布局过程和 tmux 原生命令先稳定下来。

## 7. 文件协议

### 7.1 新增目录与文件

`tmux 001` 在 session 内新增：

```text
state/
  tmux/
    layout.md
```

### 7.2 `layout.md` 内容要求

至少要记录：

- `session_id`
- `socket_name`
- `session_name`
- `window_name`
- `role_layout`
- `main_pane_ratio`
- `main_pane_id`
- 每个 role 对应的 `pane_id`
- `updated_at`

### 7.3 pane 标题规范

为了让 `switch` 与捕获能力不依赖内存态，pane 标题必须稳定：

- 主 pane 标题固定为 `main`
- role pane 标题固定为 `role:{role_id}`

这意味着后续任何命令都可以通过 `tmux list-panes` + 标题找到目标 pane，而不是依赖进程内缓存。

## 8. CLI 设计

### 8.1 新增 `switch` 命令

```bash
openoctopus switch --session <id> --role <role_id>
openoctopus switch --session <id> --main
openoctopus switch --session <id> --role <role_id> --format json
```

### 8.2 行为规则

1. `--session` 必填。
2. `--role` 与 `--main` 二选一。
3. 命令先读取 `state/tmux/layout.md`，再校验目标 pane 是否存在。
4. 如果当前进程位于 tmux 客户端内，则执行真实切换。
5. 如果当前进程不在 tmux 客户端内，则不做 attach，只返回目标信息。

### 8.3 交互式 pane 启动规则

当 `run` 运行在交互式终端、且 `runtime.tmux.enabled=true` 时：

1. 主 pane 仍保留 session banner 与总览入口。
2. 支持的 role profile（当前为 `provider: codex` + `mode: cli`）会在各自 pane 内自动拉起交互式 CLI。
3. 如 `llm_profiles.{id}.tmux_command` 已声明，则优先使用该命令；支持 `{session_dir}`、`{role_id}`、`{prompt}` 占位符。
4. 交互式 CLI 的初始 prompt 只负责读取 `planner/requirement.snapshot.md`、对应 `context.md`、`inbox.md` 并待命，不直接替代 role-runtime 正式执行链路。
5. tmux 默认焦点落到首个 role pane，避免进入后停在 main banner。

### 8.4 返回字段

建议输出：

- `session_dir`
- `socket_name`
- `session_name`
- `target_role`
- `target_pane_id`
- `switched`

## 9. 错误处理

### 9.1 运行期错误

需要显式处理：

1. 本机未安装 `tmux`，但配置开启了 `runtime.tmux.enabled=true`。
2. socket 名称模板无法展开。
3. pane 创建失败。
4. `switch` 读取到的 role 在 layout 中不存在。
5. tmux session 已被外部删除，导致 `switch` 或 capture 失败。

### 9.2 清理策略

如果 `run` 过程中 tmux bootstrap 成功，但后续 event-bus / artifact / orchestrator bootstrap 失败，必须同步清理刚创建的 tmux session，避免留下孤儿 session。

## 10. 测试策略

### 10.1 Go 单测

Go 单测重点验证：

1. 配置默认值与校验。
2. socket 模板展开。
3. `layout.md` 渲染与解析。
4. `switch` 目标解析与错误分支。

### 10.2 E2E

E2E 必须使用宿主机真实 `tmux`，黑盒验证：

1. `run` 后真实存在 tmux session。
2. `state/tmux/layout.md` 存在且内容正确。
3. `tmux list-panes` 可以读到 `main` 与 `role:{role_id}` 标题。
4. `tmux capture-pane` 可以捕获 bootstrap banner。
5. `switch --format json` 返回稳定 target 信息。

## 11. 实现取舍

## 11.1 为什么不把 role-runtime 直接塞进 tmux pane

因为当前 role-runtime 还是同步有界 tick 模型，不是常驻会话模型。此时强行把执行链路改成“每个 role 常驻跑在 pane 里”，会把 role-runtime、interrupt、安全边界、恢复路径一起拖进大改，违背“快速直接解决问题”的工程文化。

### 11.2 为什么先做 detached session

detached session 最适合当前阶段：

1. 对 CLI `run` 没有交互阻塞。
2. E2E 可以直接用 `tmux list-panes` 和 `capture-pane` 做黑盒验证。
3. 后续无论是 `attach`、`switch-client`、还是 `status --watch`，都能建立在同一个 session 基础上演进。

## 12. 与后续版本的边界

`tmux 002` 再考虑：

1. 把主 Agent / role-runtime 的输出真正接进 pane。
2. 状态 watch 与 blocker 高亮。
3. 多窗口 / 分页布局。
4. 人工审批操作面板。
5. Web / tmux 双端联动。

001 完成后，tmux 模块至少已经不再是“只有文档，没有实体实现”的状态，而是成为一个可以被 `run`、`switch`、E2E 和后续调试能力真实复用的基础设施模块。
