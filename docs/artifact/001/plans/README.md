# Artifact 001 拆分实现计划

## 概述

本文档将 `artifact 001` 拆成 6 个可串行执行的小计划，目标是先把 session 内 artifact 协议升级成真实版本化存储，再逐步接入 orchestrator / role-runtime / E2E。

与当前“只有 `artifacts/index.md` placeholder、stage 只回传 `output_refs` 字符串”的状态不同，这里的拆分聚焦 bootstrap、版本快照、输入解析、lineage/diff、真实 Codex 兼容和黑盒 E2E。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-bootstrap-and-index-layout.md` | 落地 artifact bootstrap、索引模型与目录协议 | - |
| 02 | `02-publish-version-and-manifest.md` | 落地 source 快照、version 递增、manifest 与 diff 摘要 | 01 |
| 03 | `03-input-resolution-and-context-projection.md` | 落地 artifact 输入解析、suggested ref 与 `context.md` 投影 | 01, 02 |
| 04 | `04-lineage-and-bus-events.md` | 落地 lineage 记录、发布事件与稳定错误模型 | 01-03 |
| 05 | `05-runtime-and-run-integration.md` | 接入 orchestrator / role-runtime / run，并补齐真实 Codex 执行兼容 | 01-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 建立 deterministic + 真实 Codex 的 artifact 黑盒 E2E，并接入 `make check` | 01-05 |

## 推荐执行顺序

```text
01 bootstrap 与索引协议
  ↓
02 版本发布与 manifest/diff
  ↓
03 输入解析与 context 投影
  ↓
04 lineage 与 bus 事件
  ↓
05 orchestrator / runtime / run 集成
  ↓
06 E2E 夹具与真实 Codex 验证
```

## 统一约束

- 代码实现以 Go 为主，优先使用标准库和当前仓库已有写法。
- artifact 只负责 session 内产物版本与索引，不负责内容生成本身。
- 文档、实现与测试都必须和 `docs/artifact/001/prd.md`、`docs/artifact/001/e2e.md` 保持一致。
- 不允许 `git add`、`git commit`。
- 未来若新增 `artifact 002`，不得覆盖本目录文件，应新建 `docs/artifact/002/`。
