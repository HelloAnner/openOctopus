# Session 001 Plan 04: 原子写入与失败回滚

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 落地 session 创建的真实写入流程，保证关键文件使用原子替换，创建过程任一失败时会清理半成品目录。

**Architecture:** 把“写文件”和“业务模板”分开。模板只负责产出内容，`writer.go` 负责目录创建、临时文件写入、重命名替换和失败清理，让 session 创建过程形成可测的顺序化 pipeline。

**Tech Stack:** Go、标准库

---

### Task 1: 建立原子写文件 helper

**Files:**
- Create: `internal/session/writer.go`
- Test: `internal/session/writer_test.go`

**Step 1: 写失败测试，锁定原子替换语义**

- 新文件写入使用 `*.tmp` 再 `rename`。
- 已存在文件被覆盖时，不允许留下 `.tmp` 残留。

Run: `go test ./internal/session -run TestWriteFileAtomically -v`
Expected: 先失败，说明原子写 helper 尚未建立。

**Step 2: 实现最小原子写 helper**

- 新增 `WriteFileAtomically(path string, content []byte) error` 或等价函数。
- 所有覆盖写文件都统一走这条路径。

**Step 3: 回跑原子写测试**

Run: `go test ./internal/session -run TestWriteFileAtomically -v`
Expected: 通过。

### Task 2: 建立 session 创建顺序与目录初始化

**Files:**
- Modify: `internal/session/session.go`
- Modify: `internal/session/layout.go`
- Modify: `internal/session/writer.go`
- Test: `internal/session/session_test.go`

**Step 1: 写失败测试，锁定创建顺序后的结果**

- 合法配置创建后必须得到完整骨架。
- 关键文件都存在并有内容。

Run: `go test ./internal/session -run TestCreateSessionSkeleton -v`
Expected: 先失败，说明仍是旧的最小 skeleton 行为。

**Step 2: 实现真实创建 pipeline**

- 按 PRD 约定顺序：目录 -> 配置快照 -> 元数据 -> 当前态 -> 时间线 -> 初始 checkpoint -> 其余占位文件。
- 关键文件统一使用原子写 helper。

**Step 3: 回跑骨架创建测试**

Run: `go test ./internal/session -run TestCreateSessionSkeleton -v`
Expected: 通过。

### Task 3: 建立失败回滚能力

**Files:**
- Modify: `internal/session/session.go`
- Modify: `internal/session/writer.go`
- Test: `internal/session/session_test.go`

**Step 1: 写失败测试，锁定“失败即清理”**

- 使用可注入故障点或测试专用 hook 模拟中途写文件失败。
- 断言失败返回后 `session_dir` 已被删除。

Run: `go test ./internal/session -run TestCreateSessionRollbackOnFailure -v`
Expected: 先失败，说明回滚能力未实现。

**Step 2: 实现整目录回滚**

- 任一步骤失败时执行 `os.RemoveAll(sessionDir)`。
- 只在关键路径创建完成后返回成功，不返回半成品 `CreateResult`。

**Step 3: 回跑 session 包测试**

Run: `go test ./internal/session -v`
Expected: 原子写、骨架创建、失败回滚全部通过。
