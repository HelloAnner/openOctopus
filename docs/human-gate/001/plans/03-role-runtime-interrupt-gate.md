# Human Gate 001 / Plan 03：Role Runtime 中断闸门

## 目标

修正当前“interrupt 被 ACK 后，下次 tick 仍可能继续执行”的缺口，让 `ACKNOWLEDGED` 真正成为恢复前的暂停闸门。

## 范围

- role-runtime 识别 `REQUESTED` / `ACKNOWLEDGED` / `CLEARED` 三种状态
- `REQUESTED` 时写 `INTERRUPTED` 并 ACK
- `ACKNOWLEDGED` 且未 clear 时，不允许生成新 turn

## 涉及文件

- `internal/roleruntime/interrupts.go`
- `internal/roleruntime/engine.go`
- `internal/roleruntime/engine_test.go`

## 验收点

1. 第一次 tick 命中 `REQUESTED` 时角色写成 `INTERRUPTED`。
2. 第二次 tick 在 interrupt 未 clear 时不执行 turn。
3. clear 后同一任务可继续执行。
