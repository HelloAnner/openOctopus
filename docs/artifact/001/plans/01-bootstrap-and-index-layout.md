# Plan 01 - bootstrap 与索引协议

## 目标

把 session 创建时留下的 `artifacts/index.md` 与 `audit/lineage.md` placeholder 升级成真实协议，并提供 artifact 模块自己的基础类型与路径工具。

## 产出

1. `internal/artifact` 基础包结构
2. `Bootstrap()` 幂等实现
3. `index.md` / `lineage.md` 渲染与解析模型
4. 必要单测

## 关键点

- placeholder 与合法协议内容都要兼容
- 统一 session 相对路径渲染
- 目录协议先固定，再做后续发布逻辑
