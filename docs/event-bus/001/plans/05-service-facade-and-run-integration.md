# Event Bus 001 Plan 05: 服务封装与 run 接入

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 把分散的 bootstrap、events、lock、offsets、interrupts 能力收口为 `internal/eventbus` 服务入口，并在 `openoctopus run` 创建 session 后立即完成 bus bootstrap。

**Architecture:** 保持 `cmd/openoctopus/run.go` 只做参数解析、配置校验和调用服务；真正的总线文件路径、bootstrap 细节、错误类型和协议细节都收敛在 `internal/eventbus`。这样后续 orchestrator、role-runtime、human-gate 只需要依赖统一服务入口。

**Tech Stack:** Go、Cobra

---

### Task 1: 组装 `internal/eventbus` 服务入口

**Files:**
- Create: `internal/eventbus/store.go`
- Test: `internal/eventbus/store_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定服务聚合入口**

- 通过一个 `Store` 或等价服务实例调用 bootstrap、append、acquire、commit offset。
- 断言服务层不需要调用方手工拼 `bus/*.md` 路径。

Run: `go test ./internal/eventbus -run TestStoreFacade -v`
Expected: 先失败，说明服务层还未收口。

**Step 2: 实现服务封装**

- 提供统一构造函数。
- 在服务层集中校验 session 路径和 bus 初始化状态。
- 不把 CLI 输出格式带进内部包。

**Step 3: 回跑服务封装测试**

Run: `go test ./internal/eventbus -run TestStoreFacade -v`
Expected: 通过。

### Task 2: 接入 `run` 成功路径

**Files:**
- Modify: `cmd/openoctopus/run.go`
- Modify: `cmd/openoctopus/command_test.go`
- Reference: `docs/session/001/prd.md`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定 `run` 后 bus 已 bootstrap**

- 执行最小合法 `run`。
- 断言 session 目录下 `bus/events.md` 含 `SESSION_CREATED`。
- 断言 `bus/lock.md` 不再是 `Initialized by session 001.`。

Run: `go test ./cmd/openoctopus -run TestRunCommandBootstrapsEventBus -v`
Expected: 先失败，说明 `run` 仍只创建 session 骨架。

**Step 2: 在 `run` 成功路径调用 event-bus bootstrap**

- 保持顺序：先配置校验，再创建 session，再 bootstrap bus。
- bus bootstrap 失败时，明确返回错误，不输出伪成功信息。
- 不在命令层拼 bus 文件路径。

**Step 3: 回跑命令级测试**

Run: `go test ./cmd/openoctopus -run TestRunCommandBootstrapsEventBus -v`
Expected: 通过。

### Task 3: 保持旧约束不回退

**Files:**
- Modify: `cmd/openoctopus/command_test.go`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 回放已有关键行为**

- 非法配置时，仍必须在 session 与 bus 创建前失败。
- 成功输出中仍保持 `session created:`，但 bus 文件内容已升级。

Run: `go test ./cmd/openoctopus -run TestRunCommand -v`
Expected: 先红或暴露回归风险。

**Step 2: 修正回归并补充断言**

- 保持 config gate 不变。
- 保持 session 路径输出不变。
- 增加 bus 文件断言，不放宽已有 session 断言。

**Step 3: 回跑命令级全量测试**

Run: `go test ./cmd/openoctopus -v`
Expected: 命令级测试全部通过。
