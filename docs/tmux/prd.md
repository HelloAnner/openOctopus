# TMUX 模块 PRD

## 模块目标

为主 Agent 与多个角色提供可观察、可切换、可持久的终端运行视图，支撑真实协作和远程调试。

## 核心职责

- 创建与维护 `octopus-{session_id}` 对应的 tmux server / session / pane 布局。
- 根据角色数量自动编排右侧 pane，并维护主控 pane 与角色 pane 绑定关系。
- 在交互式运行场景下，为支持的 role pane 提供待命式 CLI 页面入口。
- 为 CLI 的 `switch`、调试、pane 捕获和运行观察提供底层能力。

## 关键输入输出

- 输入：session id、角色列表、布局策略、人工切换指令。
- 输出：tmux 会话、pane id 映射、布局状态与可观测运行画面。

## 首版边界

- 首版以 tmux 原生命令编排为主，不自建终端仿真层。
- tmux 模块只负责运行视图，不负责工作流状态决策。
