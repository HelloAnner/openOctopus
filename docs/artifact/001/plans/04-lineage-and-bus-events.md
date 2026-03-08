# Plan 04 - lineage 与 bus 事件

## 目标

把 artifact 发布的审计链路补齐：不仅有 index/version，还要把 lineage 和 event-bus 证据一并留下。

## 产出

1. `audit/lineage.md` 记录追加
2. `ARTIFACT_PUBLISHED` 事件接入
3. 稳定错误类型与失败路径
4. 相关单测

## 关键点

- 先成功快照，再写 lineage 和 bus
- 记录要能反查 stage / role / task / conclusion / turn output
- 错误要能区分 missing source、invalid input ref、broken index
