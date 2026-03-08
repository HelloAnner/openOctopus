# Orchestrator 001 拆分实现计划

## 概述

本文档将 `orchestrator 001` 的实现拆成 6 个可串行执行的小计划，目标是先把 OpenOctopus 的 planner 占位文件升级为真实主控协议，再逐步接入 `run`、角色任务分发和黑盒 E2E。

与当前“只有 session 骨架和 event-bus 基座”的状态不同，这里的拆分聚焦 planner bootstrap、阶段图投影、人工输入吸收、角色任务包、结论收口与 E2E 验证。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-bootstrap-and-planner-layout.md` | 落地 orchestrator 输入输出模型、错误类型与 planner 文件 bootstrap | - |
| 02 | `02-stage-graph-and-schedule-projections.md` | 落地阶段图构建、`master_schedule.md`、`task_board.md` 与 `task_graph.mmd` 投影 | 01 |
| 03 | `03-requirement-snapshot-and-human-message-cursor.md` | 落地 `human_messages.md` 解析、`requirement.snapshot.md` 与消费游标推进 | 01 |
| 04 | `04-dispatch-packages-and-role-context.md` | 落地 ready stage 分发、`context.md` / `inbox.md` 渲染与 dispatch 事件 | 01-03 |
| 05 | `05-decision-loop-and-run-integration.md` | 落地结论收口、重试/阻塞/完成判定，并接入 `openoctopus run` | 01-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 建立 orchestrator 首版黑盒 E2E、测试 harness 与 `make check` 接入 | 01-05 |

## 推荐执行顺序

```text
01 planner bootstrap 与类型基线
  ↓
02 阶段图与排程投影
  ↓
03 requirement snapshot 与消息游标
  ↓
04 角色任务分发与 context/inbox
  ↓
05 决策循环与 run 接入
  ↓
06 E2E 夹具与黑盒验证
```

## 统一约束

- 代码实现以 Go 为主，优先使用标准库和当前仓库已有写法。
- `orchestrator` 只负责主控编排与 planner 业务文件，不负责角色真实执行。
- 文档、实现与测试都必须和 `docs/orchestrator/001/prd.md`、`docs/orchestrator/001/e2e.md` 保持一致。
- 不允许 `git add`、`git commit`。
- 未来若新增 `orchestrator 002`，不得覆盖本目录文件，应新建 `docs/orchestrator/002/`。
