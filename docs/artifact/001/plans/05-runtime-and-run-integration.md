# Plan 05 - orchestrator / runtime / run 集成

## 目标

让 artifact 能真实参与运行主链路：bootstrap、发布、输入解析，以及真实 Codex 简单任务的执行兼容。

## 产出

1. `run` 接入 artifact bootstrap
2. orchestrator 在 `SUCCESS` 时触发 artifact 发布
3. role-runtime prompt / deterministic / codex executor 兼容 artifact 输出契约
4. 命令级与集成级回归测试

## 关键点

- 不能破坏当前 session / bus / orchestrator / deterministic 基线
- 要给其他模块测试留默认关闭 runtime loop 的安全阀
- 真实 Codex 先围绕 session 内 artifact 任务闭环，不扩到 repo 级代码修改
