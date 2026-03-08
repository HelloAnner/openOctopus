# Role Runtime 模块 PRD

## 模块目标

为各类角色提供统一执行壳层，负责接任务、跑回合、落文件、写结论，并屏蔽不同执行器差异。

## 核心职责

- 管理 `inbox.md`、`outbox.md`、`state.md`、`conclusion.md` 和 `turns/*.md`。
- 抽象 Claude Code、Codex、本地 deterministic executor 等执行器适配器。
- 统一处理回合编号、状态推进、错误归类和回执提交。

## 关键输入输出

- 输入：主 Agent 分发任务、上下文快照、角色配置、工具调用请求。
- 输出：角色最新状态、回合输入输出文件、结论文件和回执事件。

## 首版边界

- 首版先支持 CLI 型执行器，不在模块内耦合 WebSocket 或复杂流式协议。
- 角色只对本角色结果负责，不能跨角色直接修改对方状态。
