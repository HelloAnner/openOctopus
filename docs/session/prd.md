# Session 模块 PRD

## 模块目标

为每次工作流执行建立独立的 session 工作目录、状态文件和 checkpoint 体系，保证可追溯、可中断、可恢复。

## 核心职责

- 生成 `session_id` 并初始化 `.octopus/sessions/{id}` 目录结构。
- 管理 `session.state.md`、`timeline.md`、`metadata.md` 与 checkpoint 文件。
- 负责原子写入、版本替换和 session 生命周期状态流转。

## 关键输入输出

- 输入：运行配置、当前阶段、事件结果、人工介入信号。
- 输出：session 目录、全局状态快照、时间线与恢复所需文件。

## 首版边界

- 首版主状态源固定为文件系统，不引入数据库作为主存储。
- session 模块只管理会话载体，不直接决定任务调度逻辑。
