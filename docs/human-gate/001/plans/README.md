# Human Gate 001 拆分实现计划

## 概述

本文档将 `human-gate 001` 拆成 6 个可串行执行的小计划，目标是先把“人工打断、人工补充、恢复续跑”从零散测试写文件升级成正式模块闭环。

与当前“只有 event-bus interrupt 投影、没有正式 CLI / service / resume 闭环”的状态不同，这里的拆分聚焦 session 解析、消息追加、中断命令、角色暂停闸门、恢复重排与 E2E 黑盒验证。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-session-resolution-and-message-append.md` | 落地 session 定位、人工消息 ID 生成与 `inject` 写盘协议 | - |
| 02 | `02-interrupt-service-and-cli.md` | 落地 `interrupt` 单角色中断服务和 CLI 命令 | 01 |
| 03 | `03-role-runtime-interrupt-gate.md` | 让 role-runtime 在 ACK 后真正暂停，clear 前不继续执行 | 02 |
| 04 | `04-resume-requeue-and-loop.md` | 落地 `resume`：clear interrupt、重排阻塞阶段并驱动恢复 loop | 01-03 |
| 05 | `05-interrupt-all-and-command-tests.md` | 落地 `interrupt-all`、命令级测试和文档同步 | 01-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 建立 human-gate 黑盒 E2E，并接入 `make check` | 01-05 |

## 推荐执行顺序

```text
01 session 解析与 inject 协议
  ↓
02 interrupt 服务与 CLI
  ↓
03 role-runtime 暂停闸门
  ↓
04 resume 重排与恢复 loop
  ↓
05 interrupt-all 与命令测试
  ↓
06 E2E 夹具与黑盒验证
```

## 统一约束

- 代码实现以 Go 为主，优先复用当前仓库已有 `event-bus` / `orchestrator` / `role-runtime`。
- `human-gate` 负责人工介入门面，不直接替代 orchestrator 或 role-runtime 的职责。
- 文档、实现与测试都必须和 `docs/human-gate/001/prd.md`、`docs/human-gate/001/e2e.md` 保持一致。
- 不允许 `git add`、`git commit`。
- 后续若新增 `human-gate 002`，必须新建 `docs/human-gate/002/`，不得覆盖本目录。
