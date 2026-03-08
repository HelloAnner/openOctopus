# Session 001 Plan 03: 初始化文件与配置快照

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 落地 `metadata.md`、`session.state.md`、`timeline.md`、`state/effective_config.yaml`、`state/checkpoints/0000-init.md` 的首版内容模板与配置快照协议。

**Architecture:** 先把关键文件的渲染逻辑做成纯模板函数，再接入写入流程。配置快照使用“可读 YAML + 稳定 hash”的双轨策略：文件可读，摘要稳定，便于后续恢复与复盘。

**Tech Stack:** Go、标准库、现有 YAML 序列化能力

---

### Task 1: 渲染 `metadata.md`

**Files:**
- Create: `internal/session/render.go`
- Test: `internal/session/render_test.go`
- Reference: `docs/session/001/prd.md`

**Step 1: 写失败测试，锁定元数据字段**

- 断言 `metadata.md` 至少包含：`session_id`、`workflow_id`、`workflow_name`、`status`、`created_at`、`config_path`、`sessions_dir`、`workspace_root`、`effective_config_path`、`effective_config_hash`、`applied_defaults_count`。

Run: `go test ./internal/session -run TestRenderMetadata -v`
Expected: 先失败，说明元数据模板尚未落地。

**Step 2: 实现元数据模板**

- 以 Markdown 列表或表格输出均可，但字段必须稳定。
- 初始 `status` 固定为 `INITIAL`。

**Step 3: 回跑元数据模板测试**

Run: `go test ./internal/session -run TestRenderMetadata -v`
Expected: 通过。

### Task 2: 渲染当前态、时间线和初始 checkpoint

**Files:**
- Modify: `internal/session/render.go`
- Test: `internal/session/render_test.go`

**Step 1: 写失败测试，锁定三类初始化文件内容**

- `session.state.md` 包含 `status: INITIAL`、`checkpoint_seq: 0`、`last_event: SESSION_CREATED`。
- `timeline.md` 首条记录包含 `SESSION_CREATED`。
- `0000-init.md` 至少引用 `effective_config.yaml`。

Run: `go test ./internal/session -run TestRenderInitialStateFiles -v`
Expected: 先失败，说明初始化内容未建立。

**Step 2: 实现三类模板函数**

- 新增 `RenderSessionState`、`RenderTimeline`、`RenderInitialCheckpoint` 或等价 helper。
- 所有模板只依赖输入对象，不直接读写文件系统。

**Step 3: 回跑状态文件模板测试**

Run: `go test ./internal/session -run TestRenderInitialStateFiles -v`
Expected: 通过。

### Task 3: 落地有效配置快照与稳定 hash

**Files:**
- Modify: `internal/session/render.go`
- Test: `internal/session/render_test.go`
- Test: `internal/session/session_test.go`

**Step 1: 写失败测试，锁定快照与 hash 协议**

- `state/effective_config.yaml` 必须包含 `RuntimeConfig` 的完整有效配置。
- `effective_config_hash` 对同一配置稳定，对不同配置变化。

Run: `go test ./internal/session -run TestEffectiveConfigSnapshotAndHash -v`
Expected: 先失败，说明快照和 hash 规则未落实。

**Step 2: 实现配置快照生成**

- 使用 YAML 输出 `effective_config.yaml`。
- 使用稳定序列化结果计算 `sha256` 并注入 `metadata.md`。

**Step 3: 回跑渲染相关测试**

Run: `go test ./internal/session -run 'TestRenderMetadata|TestRenderInitialStateFiles|TestEffectiveConfigSnapshotAndHash' -v`
Expected: 通过。
