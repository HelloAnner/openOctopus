# Event Bus 001 拆分实现计划

## 概述

本文档将 `event-bus 001` 的实现拆成 6 个可串行执行的小计划，目标是先把 OpenOctopus 的 bus 占位文件升级为真实 WAL 总线，再逐步接入 `run` 与黑盒 E2E。

与当前“只有 `bus/*.md` 占位文件”的状态不同，这里的拆分聚焦 bootstrap、事件追加、锁协议、offset / interrupt 投影、服务接入与 E2E 验证。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-bootstrap-and-store-types.md` | 落地 event-bus 输入输出模型、错误类型与 placeholder 升级入口 | - |
| 02 | `02-event-wal-and-hash-chain.md` | 落地 `events.md` 追加写、读取、序号与哈希链校验 | 01 |
| 03 | `03-lock-lease-and-conflict-control.md` | 落地 `lock.md` 的租约获取、续租、释放与版本冲突控制 | 01 |
| 04 | `04-offsets-and-interrupt-projections.md` | 落地 `offsets.md` 与 `interrupts.md` 的原子投影更新 | 01-03 |
| 05 | `05-service-facade-and-run-integration.md` | 组装 `internal/eventbus` 服务入口，并接入 `openoctopus run` | 01-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 建立 event-bus 首版黑盒 E2E、测试 harness 与 `make check` 接入 | 01-05 |

## 推荐执行顺序

```text
01 类型与 bootstrap 入口
  ↓
02 事件 WAL 与链校验
  ↓
03 锁租约与冲突控制
  ↓
04 offsets / interrupts 投影
  ↓
05 服务封装与 run 接入
  ↓
06 E2E 夹具与黑盒验证
```

## 统一约束

- 代码实现以 Go 为主，优先使用标准库和当前仓库已有写法。
- `event-bus` 只负责总线文件协议与读写服务，不负责调度算法、角色执行和人工决策本身。
- 文档、实现与测试都必须和 `docs/event-bus/001/prd.md`、`docs/event-bus/001/e2e.md` 保持一致。
- 不允许 `git add`、`git commit`。
- 未来若新增 `event-bus 002`，不得覆盖本目录文件，应新建 `docs/event-bus/002/`。
