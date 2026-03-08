# Human Gate 001 / Plan 04：Resume 重排与恢复 Loop

## 目标

落地 `resume` 的首版核心能力：清理 `ACKNOWLEDGED` interrupt，把 `BLOCKED` 阶段重新入队，并直接复用现有 orchestrator / role-runtime loop 继续推进 session。

## 范围

- 新增 `resume` 命令与服务
- 让 `BLOCKED` 阶段变为 `RETRY_PENDING`
- 递增 attempt，保证生成新的 `task_id`
- 驱动同步恢复 loop

## 涉及文件

- `internal/humangate/schedule.go`
- `internal/humangate/service.go`
- `cmd/openoctopus/resume.go`
- `cmd/openoctopus/run.go`
- `cmd/openoctopus/human_gate_command_test.go`

## 验收点

1. `resume` 后 `ACKNOWLEDGED` interrupt 变为 `CLEARED`。
2. `BLOCKED` 阶段恢复后重新分发，得到新的 `task_id`。
3. 恢复后工作流能继续推进，而不是只改文件不跑逻辑。
