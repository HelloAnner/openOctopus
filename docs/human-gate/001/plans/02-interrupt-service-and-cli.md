# Human Gate 001 / Plan 02：Interrupt 服务与 CLI

## 目标

把单角色中断从“只能由测试代码直接写 bus 文件”升级为正式 `interrupt` CLI，并统一走 `internal/humangate` 服务层。

## 范围

- 新增 `interrupt` 命令
- 基于 `event-bus` lease 申请 `INTERRUPT_REQUESTED`
- 返回稳定命令输出与错误

## 涉及文件

- `internal/humangate/service.go`
- `cmd/openoctopus/interrupt.go`
- `cmd/openoctopus/root.go`
- `cmd/openoctopus/human_gate_command_test.go`

## 验收点

1. `interrupt --role agent_a --reason ...` 能成功写入 `bus/events.md`。
2. `bus/interrupts.md` 中新增目标角色记录。
3. 不指定 `--role` 或 `--reason` 时命令失败。
