# Event Bus 001 Plan 02: 事件 WAL 与哈希链

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `events.md` 的真实追加写、读取、尾事件查询和链校验能力，让 event-bus 成为 session 目录中的唯一事件 WAL。

**Architecture:** 先实现事件条目的稳定序列化与解析，再实现“读全量 -> 内存追加 -> 原子替换写”的 append 流程，最后补齐 `event_id`、`sequence`、`prev_event_hash`、`event_hash` 的一致性校验。首版优先追求原子性和可读性，不做复杂索引或流式追加优化。

**Tech Stack:** Go、标准库、`sha256`

---

### Task 1: 实现事件条目序列化与解析

**Files:**
- Create: `internal/eventbus/events.go`
- Test: `internal/eventbus/events_parse_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定单条事件的解析和渲染**

- 构造一条最小事件 Markdown，断言可被解析为结构体。
- 构造一个事件结构体，断言渲染结果含 `event_id`、`sequence`、`event_hash` 等关键字段。

Run: `go test ./internal/eventbus -run TestEventParseAndRender -v`
Expected: 先失败，说明事件解析器和渲染器尚未建立。

**Step 2: 实现稳定渲染与解析**

- 固定字段顺序，保证 diff 稳定。
- 对空字段也输出占位键，避免解析分支过多。
- 仅支持首版约定格式，不做宽松兼容解析。

**Step 3: 回跑解析与渲染测试**

Run: `go test ./internal/eventbus -run TestEventParseAndRender -v`
Expected: 通过。

### Task 2: 实现 append 与尾事件查询

**Files:**
- Modify: `internal/eventbus/events.go`
- Test: `internal/eventbus/events_append_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定追加行为**

- bootstrap 后追加两条事件。
- 断言新事件 `event_id` 是 `event-000002`、`event-000003`。
- 断言 `Tail()` 返回最后一条事件。

Run: `go test ./internal/eventbus -run TestAppendEvent -v`
Expected: 先失败，说明追加和尾读行为尚未建立。

**Step 2: 实现 append 与 `Tail()`**

- 读取现有 `events.md`。
- 生成下一个 `event_id` 与 `sequence`。
- 用上一条事件的 `event_hash` 填充 `prev_event_hash`。
- 使用 `*.tmp + rename` 原子替换事件文件。

**Step 3: 回跑追加测试**

Run: `go test ./internal/eventbus -run TestAppendEvent -v`
Expected: 通过。

### Task 3: 实现链校验与增量读取

**Files:**
- Modify: `internal/eventbus/events.go`
- Test: `internal/eventbus/events_read_test.go`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 写失败测试，锁定链完整性和按 `after_event_id` 读取**

- 断言 `ListAfter("event-000001")` 只返回后续事件。
- 手工破坏 `prev_event_hash` 后，断言读取报 `event chain broken`。

Run: `go test ./internal/eventbus -run TestListAndValidateChain -v`
Expected: 先失败，说明增量读取和链校验尚未完成。

**Step 2: 实现 `List()` / `ListAfter()` / `ValidateChain()`**

- 每次读取都校验序号连续性。
- 对哈希链错误快速失败，不做静默修复。
- 让 `after_event_id` 不存在时返回明确错误。

**Step 3: 回跑 WAL 相关测试**

Run: `go test ./internal/eventbus -run 'Test(AppendEvent|ListAndValidateChain)' -v`
Expected: 通过。
