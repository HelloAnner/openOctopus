# TMUX 模块 001 阶段 E2E 方案

## 1. 目标

`tmux 001` 的 E2E 只验证一个黑盒事实：**当配置开启 tmux 后，OpenOctopus 是否真的在宿主机创建了可观察、可捕获、可切换目标的 tmux 会话。**

本阶段重点不是验证 role-runtime 是否真的在 pane 内长期运行，而是验证 tmux 基础设施已经真实落地。

## 2. 环境原则

1. 必须使用宿主机真实 `tmux`，禁止 mock tmux 行为。
2. 继续复用仓库现有 E2E 基建：`go build`、`run_cli(...)`、`e2e-test/`。
3. 默认关闭 role-runtime loop，避免把 tmux E2E 和真实执行链路耦在一起。
4. 每个用例执行后要清理自己创建的 tmux session，避免污染宿主机环境。
5. 黑盒断言只看 CLI 结果、session 文件和真实 tmux 命令输出。

## 3. 验证范围

### 3.1 本阶段必须覆盖

1. `runtime.tmux` 配置能被 `run` 消费。
2. `run` 后真实存在独立 tmux socket / session。
3. `state/tmux/layout.md` 正式落盘。
4. pane 标题包含 `main` 和 `role:{role_id}`。
5. pane 内可捕获 bootstrap banner。
6. `switch --format json` 能稳定返回目标 pane 信息。
7. 非 tmux 客户端环境下执行 `switch` 不会阻塞 attach。

### 3.2 本阶段暂不覆盖

1. tmux 内实际运行主 Agent / 子 Agent 常驻进程。
2. `status --watch` 一类实时终端面板。
3. 多窗口编排。
4. pane 动态热重排。

## 4. 测试夹具设计

### 4.1 `valid-basic-layout`

目标：验证最小双 role 场景会创建左主右子布局。

特点：

- 两个 role
- `runtime.tmux.enabled=true`
- 默认关闭 role-runtime loop
- 断言 layout 文件与 tmux pane 标题

### 4.2 `valid-switch-role`

目标：验证 `switch` 命令能解析目标 role pane，并在非 tmux 客户端环境返回稳定 JSON。

特点：

- 复用 tmux enabled fixture
- 调用 `switch --role agent_b --format json`
- 断言 `target_pane_id`、`socket_name`、`session_name`、`switched=false`

### 4.3 `invalid-layout-config`

目标：验证非法 tmux 配置会在 `validate` 阶段被阻断。

特点：

- `main_pane_ratio=-1`
- 或 `role_layout=unknown`
- 断言退出码非 0，stderr 含稳定错误路径

## 5. 核心用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| TMUX-E2E-001 | run 创建 tmux session | `valid-basic-layout/octopus.yaml` | 退出码 `0`，真实 tmux session 存在 |
| TMUX-E2E-002 | layout 文件落盘 | `valid-basic-layout` | `state/tmux/layout.md` 存在，含 socket / pane map |
| TMUX-E2E-003 | pane 标题正确 | `valid-basic-layout` | `list-panes` 输出含 `main`、`role:agent_a`、`role:agent_b` |
| TMUX-E2E-004 | pane 可捕获 | `valid-basic-layout` | `capture-pane` 输出含 bootstrap banner |
| TMUX-E2E-005 | switch 返回稳定 JSON | `valid-switch-role` | 退出码 `0`，JSON 含 `target_pane_id` 且 `switched=false` |
| TMUX-E2E-006 | 非法配置被 validate 阻断 | `invalid-layout-config/octopus.yaml` | 退出码非 `0`，错误定位到 `runtime.tmux.*` |

## 6. 关键断言方式

### 6.1 tmux session 断言

至少验证：

- `tmux -L <socket_name> has-session -t <session_name>` 返回成功。
- `layout.md` 中的 `socket_name` 与 `session_name` 可被真实命令消费。

### 6.2 pane 标题断言

通过：

```bash
tmux -L <socket_name> list-panes -t <session_name> -F '#{pane_title}'
```

至少看到：

- `main`
- `role:agent_a`
- `role:agent_b`

### 6.3 pane 捕获断言

通过：

```bash
tmux -L <socket_name> capture-pane -pt <pane_id>
```

至少看到 bootstrap banner，例如：

- `[openoctopus] main session=<session_id>`
- `[openoctopus] role=agent_a`

### 6.4 switch 断言

至少断言 JSON 中存在：

- `session_dir`
- `socket_name`
- `session_name`
- `target_role`
- `target_pane_id`
- `switched`

并且在 pytest 子进程默认不在 tmux 客户端内时，`switched` 必须为 `false`。

## 7. 推荐测试实现

### 7.1 目录结构

建议新增：

```text
e2e/
  tmux/
    fixtures/
      valid-basic-layout/
        octopus.yaml
      invalid-layout-config/
        octopus.yaml
    test_tmux_layout.py
```

### 7.2 Python 辅助函数

建议在测试文件内封装：

1. 解析 `session created: ...` 得到 `session_dir`。
2. 解析 `layout.md` 中的 `socket_name`、`session_name`、`pane_id`。
3. 调用 `tmux` 原生命令。
4. 在 `finally` 中清理 tmux session。

## 8. 通过标准

以下条件同时满足，`tmux 001` E2E 才算通过：

1. `python3 -m pytest e2e/tmux -v` 全通过。
2. 新增用例已接入 `e2e/README.md` 与 `Makefile e2e` 目标。
3. `make check` 通过。
