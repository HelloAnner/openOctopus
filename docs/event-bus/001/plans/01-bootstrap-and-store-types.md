# Event Bus 001 Plan 01: Bootstrap 入口与类型基线

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `event-bus 001` 的基础模型、稳定错误类型和 bootstrap 入口，让 session 创建出来的 bus 占位文件可以被升级为真实协议文件。

**Architecture:** 先定义 `internal/eventbus` 的核心类型和错误，再实现 `Bootstrap(...)`，把 session 001 生成的 `bus/*.md` 占位文件替换成可解析的初始 bus 文件。bootstrap 只负责“把总线点亮”，不承担事件追加、锁竞争和业务推进。

**Tech Stack:** Go、标准库

---

### Task 1: 定义基础类型与错误模型

**Files:**
- Create: `internal/eventbus/types.go`
- Create: `internal/eventbus/errors.go`
- Test: `internal/eventbus/types_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定基础模型形状**

- 为 `BootstrapOptions`、`Lease`、`AppendEvent`、`OffsetCommit`、`InterruptRecord` 写最小测试。
- 为错误类型写 `errors.Is(...)` 断言，确认可稳定识别 `lease conflict`、`offset regression`、`event chain broken`。

Run: `go test ./internal/eventbus -run TestTypes -v`
Expected: 先失败，提示类型或错误尚未定义。

**Step 2: 实现基础模型与错误类型**

- 定义对外输入输出模型。
- 定义稳定错误变量，避免后续靠字符串比较错误原因。
- 保持字段职责清晰，不把业务层概念塞进总线模型。

**Step 3: 回跑类型测试**

Run: `go test ./internal/eventbus -run TestTypes -v`
Expected: 通过。

### Task 2: 建立 bus 文件路径与初始内容模板

**Files:**
- Create: `internal/eventbus/layout.go`
- Create: `internal/eventbus/render.go`
- Test: `internal/eventbus/render_test.go`
- Reference: `docs/session/001/prd.md`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定 bootstrap 初始内容**

- 断言 `events.md` 初始内容包含首条 `SESSION_CREATED` 事件。
- 断言 `lock.md` 初始状态为 `FREE`。
- 断言 `offsets.md`、`interrupts.md` 是合法文档而不是空白文件。

Run: `go test ./internal/eventbus -run TestRenderBootstrapFiles -v`
Expected: 先失败，说明模板内容未落地。

**Step 2: 实现 bus 文件模板构造**

- 统一管理 `bus/*.md` 路径。
- 生成合法标题、初始字段和首条事件内容。
- 保证所有时间字段使用 UTC RFC3339。

**Step 3: 回跑渲染测试**

Run: `go test ./internal/eventbus -run TestRenderBootstrapFiles -v`
Expected: 通过。

### Task 3: 实现 placeholder 升级与幂等 bootstrap

**Files:**
- Create: `internal/eventbus/bootstrap.go`
- Test: `internal/eventbus/bootstrap_test.go`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 写失败测试，锁定两类关键行为**

- session 001 的占位文件会被升级为真实 bus 文件。
- 对已 bootstrap 的 session 再执行一次 bootstrap，不会追加第二条 `SESSION_CREATED`。

Run: `go test ./internal/eventbus -run TestBootstrap -v`
Expected: 先失败，说明升级与幂等行为尚未建立。

**Step 2: 实现最小 bootstrap 入口**

- 检查 `bus/` 目录是否存在。
- 识别“session 001 占位文本”与“已 bootstrap 协议文件”。
- 用 `*.tmp + rename` 原子替换初始文件。

**Step 3: 回跑 bootstrap 测试**

Run: `go test ./internal/eventbus -run TestBootstrap -v`
Expected: 通过。
