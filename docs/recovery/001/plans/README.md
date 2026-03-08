# Recovery 001 拆分实现计划

## 概述

本文档将 `recovery 001` 的实现拆成 6 个可串行执行的小计划，目标是先把 session 恢复基线点亮，再逐步补齐 checkpoint、状态修复、CLI 与黑盒 E2E。

与当前“只有 `docs/recovery/prd.md`、没有正式实现”的状态不同，这里的拆分聚焦 recovery 服务、checkpoint、恢复视图修复、正式命令与 E2E。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-types-and-session-validation.md` | 落地 recovery 基础类型、关键文件校验与事件哈希链校验 | - |
| 02 | `02-checkpoint-writer-and-rendering.md` | 落地增量 checkpoint 渲染、序号分配与首版落盘入口 | 01 |
| 03 | `03-state-reducer-and-replay-report.md` | 落地恢复视图归约、`session.state.md` / `blockers.md` 修复与 `audit/replay.md` | 01-02 |
| 04 | `04-recover-service-and-command.md` | 组装 `internal/recovery` 服务并接入正式 `recover` CLI | 01-03 |
| 05 | `05-orchestrator-humangate-checkpoint-hooks.md` | 把 checkpoint 接入 orchestrator / human-gate 的关键边界 | 02-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 建立 recovery 黑盒 E2E、fixture 与 `make check` 接入 | 01-05 |

## 推荐执行顺序

```text
01 类型与会话校验
  ↓
02 checkpoint 渲染与写入
  ↓
03 状态归约与 replay 报告
  ↓
04 recover 服务与 CLI
  ↓
05 orchestrator / human-gate 接入 checkpoint
  ↓
06 E2E 夹具与黑盒验证
```

## 统一约束

- 代码实现以 Go 为主，优先使用标准库与当前仓库已有写法。
- recovery 只负责校验、修复、checkpoint 与续跑，不重新实现 orchestrator / role-runtime 的主业务逻辑。
- 文档、实现与测试必须和 `docs/recovery/001/prd.md`、`docs/recovery/001/e2e.md` 保持一致。
- 不允许 `git add`、`git commit`。
- 未来若新增 `recovery 002`，不得覆盖本目录文件，应新建 `docs/recovery/002/`。

