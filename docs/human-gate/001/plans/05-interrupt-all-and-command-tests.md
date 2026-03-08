# Human Gate 001 / Plan 05：Interrupt-All 与命令测试

## 目标

补齐批量中断命令，并用命令级测试把 `interrupt` / `interrupt-all` / `inject` / `resume` 的主要参数约束和关键产物锁住。

## 范围

- 新增 `interrupt-all` 命令
- 统一命令级测试夹具
- 更新 CLI 帮助和输出

## 涉及文件

- `cmd/openoctopus/interrupt_all.go`
- `cmd/openoctopus/inject.go`
- `cmd/openoctopus/human_gate_command_test.go`

## 验收点

1. `interrupt-all` 能让 session 立即进入 `WAITING_HUMAN`。
2. `inject` 缺少 `--message` / `--input` 时返回错误。
3. `resume` 缺少 `--session` 时返回错误。
