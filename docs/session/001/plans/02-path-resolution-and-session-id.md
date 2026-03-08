# Session 001 Plan 02: 路径解析与 Session ID

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 落地 `runtime.workspace.sessions_dir` 的绝对/相对路径解析、`session_id` 生成和关键结果路径组装，保证 session 创建位置稳定可预期。

**Architecture:** 把“路径如何算”从“文件如何写”里拆出来。路径解析与 ID 生成应该是纯函数或近似纯函数，方便单测锁死，避免后续 CLI、E2E、session 写入逻辑各自重复推导目录。

**Tech Stack:** Go、标准库

---

### Task 1: 建立 `sessions_dir` 解析规则

**Files:**
- Create: `internal/session/path.go`
- Test: `internal/session/path_test.go`
- Reference: `docs/session/001/prd.md`

**Step 1: 写失败测试，锁定路径解析规则**

- `sessions_dir` 为绝对路径时直接使用。
- `sessions_dir` 为相对路径时相对 `dirname(configPath)` 解析。
- 不从 `workspace.root` 再做二次隐式推导。

Run: `go test ./internal/session -run TestResolveSessionsDir -v`
Expected: 先失败，说明路径规则还没被实现。

**Step 2: 实现解析函数**

- 新增 `ResolveSessionsDir(configPath, sessionsDir string) string` 或等价 helper。
- 只在 `path.go` 维护该逻辑，CLI 不重复实现。

**Step 3: 回跑路径解析测试**

Run: `go test ./internal/session -run TestResolveSessionsDir -v`
Expected: 通过，路径规则稳定。

### Task 2: 建立 `session_id` 生成与冲突重试

**Files:**
- Modify: `internal/session/path.go`
- Test: `internal/session/path_test.go`

**Step 1: 写失败测试，锁定 `session_id` 格式与唯一性**

- `session_id` 必须匹配 `sess_{unix_nano}`。
- 连续生成两次结果不同。
- 已存在同名目录时允许有限次重试，而不是直接覆盖。

Run: `go test ./internal/session -run TestGenerateSessionID -v`
Expected: 先失败，说明 ID 规则与重试策略未建立。

**Step 2: 实现 ID 生成 helper**

- 新增 `GenerateSessionID()` 和有限次冲突重试逻辑。
- 不引入复杂 UUID 依赖，保持首版简单。

**Step 3: 回跑 ID 测试**

Run: `go test ./internal/session -run TestGenerateSessionID -v`
Expected: 通过，ID 生成稳定。

### Task 3: 组装 `CreateResult` 路径字段

**Files:**
- Modify: `internal/session/path.go`
- Modify: `internal/session/types.go`
- Test: `internal/session/path_test.go`

**Step 1: 写失败测试，锁定结果路径**

- `CreateResult` 中的 `MetadataPath`、`StatePath`、`TimelinePath`、`EffectiveConfigPath`、`InitialCheckpoint` 必须都位于同一个 `SessionDir` 下。

Run: `go test ./internal/session -run TestBuildCreateResultPaths -v`
Expected: 先失败，说明结果路径仍是临时拼装。

**Step 2: 实现路径组装 helper**

- 根据 `SessionDir` 统一生成所有关键路径。
- 下游调用只消费 `CreateResult`，不再自行 `filepath.Join`。

**Step 3: 回跑 session 路径测试**

Run: `go test ./internal/session -run 'TestBuildCreateResultPaths|TestResolveSessionsDir|TestGenerateSessionID' -v`
Expected: 通过。
