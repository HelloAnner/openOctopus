# Orchestrator 001 Plan 03: Requirement Snapshot 与 Human Messages 游标

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `human_messages.md` 的解析、`requirement.snapshot.md` 的稳定渲染以及 `human_message_cursor` 的推进能力，让主控能吸收人工输入而不重复消费旧消息。

**Architecture:** 把 `human_messages.md` 视为 append-only 输入日志，orchestrator 只负责读取和推进消费游标。首版快照不依赖额外 LLM 总结，而是做确定性文本合并，以保证可预测、易测试、可恢复。

**Tech Stack:** Go、标准库

---

### Task 1: 实现 human messages 解析器

**Files:**
- Create: `internal/orchestrator/human_messages.go`
- Test: `internal/orchestrator/human_messages_test.go`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定消息块解析**

- 空文档或 placeholder 文档返回 0 条消息。
- 合法 Markdown 可被解析为有序 `HumanMessage` 列表。
- 缺失 `message_id` 或 `created_at` 的块返回稳定错误。

Run: `go test ./internal/orchestrator -run TestParseHumanMessages -v`
Expected: 先失败，说明解析器尚未建立。

**Step 2: 实现消息块解析**

- 固定支持 `## message: <id>` 块格式。
- 保留消息顺序，不在解析阶段做业务裁剪。
- 对 placeholder 和空白内容做兼容处理。

**Step 3: 回跑消息解析测试**

Run: `go test ./internal/orchestrator -run TestParseHumanMessages -v`
Expected: 通过。

### Task 2: 实现 requirement snapshot 渲染与版本递增

**Files:**
- Create: `internal/orchestrator/snapshot.go`
- Test: `internal/orchestrator/snapshot_test.go`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定快照增长语义**

- 首次 bootstrap 后 `snapshot_version=1`。
- 新增一条消息后再次吸收，`snapshot_version` 递增。
- 没有新消息时再次吸收，`snapshot_version` 不重复增长。

Run: `go test ./internal/orchestrator -run TestRequirementSnapshot -v`
Expected: 先失败，说明快照合并尚未完成。

**Step 2: 实现快照渲染与递增规则**

- 从 `effective_config.yaml` 提取 workflow 摘要。
- 只拼接 cursor 之后的新消息到 `latest_messages` 段。
- 生成稳定的 `snapshot_version`、`source_message_count`、`human_message_cursor`。

**Step 3: 回跑快照测试**

Run: `go test ./internal/orchestrator -run TestRequirementSnapshot -v`
Expected: 通过。

### Task 3: 更新 `session.state.md` 的主控消费位点

**Files:**
- Create: `internal/orchestrator/session_state.go`
- Test: `internal/orchestrator/session_state_test.go`
- Reference: `docs/session/001/prd.md`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定状态推进**

- 吸收新消息后，`session.state.md` 的 `human_message_cursor` 会更新。
- 无新消息时，不会重复改写无关字段。
- 阻塞场景会把 `status` 推进为 `WAITING_HUMAN`。

Run: `go test ./internal/orchestrator -run TestSessionStateProjection -v`
Expected: 先失败，说明 session 状态投影尚未落地。

**Step 2: 实现 session 状态投影**

- 复用已有 `session.state.md` 文件，而不是另建第二份全局当前态。
- 只更新 orchestrator 负责的字段。
- 保持原子替换写，不破坏 session 001 已有字段。

**Step 3: 回跑 session 状态测试**

Run: `go test ./internal/orchestrator -run TestSessionStateProjection -v`
Expected: 通过。
