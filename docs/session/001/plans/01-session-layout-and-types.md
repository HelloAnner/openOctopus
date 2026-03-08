# Session 001 Plan 01: 目录骨架与类型模型

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 定义 `session 001` 的创建输入输出模型、标准目录骨架和关键文件常量，为后续写入逻辑提供统一协议入口。

**Architecture:** 先把“创建什么”固定下来，再实现“怎么创建”。`internal/session` 入口文件保持很薄，目录结构、结果对象和文件列表通过独立小文件维护，避免后续把路径和文件名散落在 CLI、session、E2E 三处。

**Tech Stack:** Go、标准库

---

### Task 1: 定义 session 创建输入输出模型

**Files:**
- Create: `internal/session/types.go`
- Modify: `internal/session/session.go`
- Test: `internal/session/session_test.go`

**Step 1: 写失败测试，锁定创建结果对象**

- 约束 `CreateResult` 至少返回 `SessionID`、`SessionDir`、`MetadataPath`、`StatePath`、`TimelinePath`、`EffectiveConfigPath`、`InitialCheckpoint`。
- 约束 `CreateOptions` 至少接收 `RuntimeConfig`、`ConfigPath`、`AppliedDefaults`。

Run: `go test ./internal/session -run TestCreateResultContract -v`
Expected: 先失败，说明类型和结果协议尚未建立。

**Step 2: 实现最小类型定义**

- 新建 `CreateOptions`、`CreateResult`。
- 让 `session.go` 的对外入口只编排参数，不再直接拼具体路径字符串。

**Step 3: 回跑 session 类型测试**

Run: `go test ./internal/session -run TestCreateResultContract -v`
Expected: 通过，结果对象稳定。

### Task 2: 定义标准目录骨架常量

**Files:**
- Create: `internal/session/layout.go`
- Test: `internal/session/layout_test.go`
- Reference: `docs/session/001/prd.md`

**Step 1: 写失败测试，锁定必须存在的目录与文件**

- 断言骨架中至少包含：`planner/`、`bus/`、`roles/`、`artifacts/`、`state/checkpoints/`、`audit/`。
- 断言关键文件至少包含：`metadata.md`、`session.state.md`、`timeline.md`、`artifacts/index.md`、`state/effective_config.yaml`、`state/checkpoints/0000-init.md`。

Run: `go test ./internal/session -run TestSessionLayoutDefinition -v`
Expected: 先失败，说明布局定义尚未建立。

**Step 2: 实现布局描述**

- 在 `layout.go` 中集中定义目录与文件清单。
- 关键文件与占位文件分开表示，避免后续写入逻辑混在一起。

**Step 3: 回跑布局测试**

Run: `go test ./internal/session -run TestSessionLayoutDefinition -v`
Expected: 通过，布局清单稳定。

### Task 3: 收敛对外入口，避免后续继续扩散路径常量

**Files:**
- Modify: `internal/session/session.go`
- Test: `internal/session/session_test.go`

**Step 1: 写失败测试，锁定入口只依赖模型与布局**

- 测试目标不是完整创建成功，而是保证入口函数不再直接硬编码 `metadata.md`、`timeline.md` 等散落字符串。

Run: `go test ./internal/session -run TestCreateSessionUsesLayoutDefinition -v`
Expected: 先失败，说明入口仍然是旧的临时 skeleton 实现。

**Step 2: 让 `session.go` 变成薄入口**

- 只保留对 `CreateOptions`、布局定义和后续写入流程的调用。
- 为后续 `path.go`、`render.go`、`writer.go` 留出清晰扩展点。

**Step 3: 回跑 session 包测试**

Run: `go test ./internal/session -v`
Expected: 基础模型与布局测试通过。
