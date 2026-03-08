# Session 001 拆分实现计划

## 概述

本文档将 `session 001` 的实现拆成 6 个可串行执行的小计划，目标是先把 OpenOctopus 的 session 文件系统载体做稳，再逐步接入 CLI 与黑盒 E2E。

与旧的“只创建一个 `metadata.md`”的轻量实现不同，这里的拆分聚焦标准目录骨架、初始化文件协议、有效配置快照、原子写入与失败回滚。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-session-layout-and-types.md` | 落地 session 创建输入输出模型与标准目录骨架常量 | - |
| 02 | `02-path-resolution-and-session-id.md` | 落地 `sessions_dir` 解析、`session_id` 生成与结果路径组装 | 01 |
| 03 | `03-initial-files-and-config-snapshot.md` | 落地 `metadata/state/timeline/checkpoint` 模板与有效配置快照 | 01, 02 |
| 04 | `04-atomic-write-and-rollback.md` | 落地原子写文件、创建顺序与失败回滚 | 01-03 |
| 05 | `05-run-command-integration.md` | 将 session 创建正式接入 `openoctopus run` 与命令级测试 | 01-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 建立 session 首版黑盒 E2E、夹具与 `make check` 接入 | 01-05 |

## 推荐执行顺序

```text
01 目录骨架与类型
  ↓
02 路径解析与 session_id
  ↓
03 初始化文件与配置快照
  ↓
04 原子写入与失败回滚
  ↓
05 run 命令接入
  ↓
06 E2E 夹具与黑盒验证
```

## 统一约束

- 代码实现以 Go 为主，优先使用标准库和当前仓库已有写法。
- `session` 只负责会话载体，不负责调度推进、事件消费和角色执行。
- 文档、实现与测试都必须和 `docs/session/001/prd.md`、`docs/session/001/e2e.md` 保持一致。
- 不允许 `git add`、`git commit`。
- 未来若新增 `session 002`，不得覆盖本目录文件，应新建 `docs/session/002/`。
