# CLI 001 拆分实现计划

## 概述

本文档将 `cli 001` 的实现拆成 6 个可串行执行的小计划，目标是先把 CLI 输出协议、退出码和状态读取基线做实，再把现有命令收口到统一的正式门面，并补齐黑盒 E2E。

与当前“命令已经可用，但输出零散、缺少 status、脚本消费不稳定”的状态不同，这里的拆分聚焦输出模型、状态读模型、`status` 命令、现有命令格式统一和 E2E 收口。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-output-model-and-exit-codes.md` | 建立统一输出模型、错误模型和最小退出码协议 | - |
| 02 | `02-session-resolution-and-status-reader.md` | 抽取 session 解析与状态聚合只读服务 | 01 |
| 03 | `03-validate-run-format-unification.md` | 为 `validate` / `run` 接入 `--format text|json` | 01-02 |
| 04 | `04-human-gate-command-format-unification.md` | 为 `interrupt` / `interrupt-all` / `inject` / `resume` 接入统一输出 | 01-02 |
| 05 | `05-status-command-and-root-wiring.md` | 实现 `status` 命令并接入 root / main | 01-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 补齐 `e2e/cli`、README、Makefile 与最终验证 | 01-05 |

## 推荐执行顺序

```text
01 输出模型与退出码
  ↓
02 session 解析与状态读取
  ↓
03 validate / run 统一输出
  ↓
04 human-gate 命令统一输出
  ↓
05 status 命令与 root 接线
  ↓
06 E2E 与 make check 收尾
```

## 统一约束

- CLI 只负责参数解析、输出展示与错误映射，不承载复杂业务写逻辑。
- `status` 只读 session 协议文件，不修改业务文件。
- 文档、实现与测试必须和 `docs/cli/001/prd.md`、`docs/cli/001/e2e.md` 保持一致。
- 不允许 `git add`、`git commit`。
- 未来若新增 `cli 002`，不得覆盖本目录文件，应新建 `docs/cli/002/`。

