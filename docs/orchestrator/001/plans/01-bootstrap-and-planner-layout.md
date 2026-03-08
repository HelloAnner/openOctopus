# Orchestrator 001 Plan 01: Bootstrap 与 Planner 文档基线

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `orchestrator 001` 的基础模型、稳定错误类型和 planner bootstrap 入口，让 session 001 创建出来的 planner 占位文件可以被升级为真实主控协议文件。

**Architecture:** 先定义 `internal/orchestrator` 的核心类型和错误，再实现 `Bootstrap()`，把 `planner/*.md` 占位文件升级为真实协议文件，并补齐 session 001 尚未创建的 `task_board.md`、`task_graph.mmd`、`dispatch_log.md`、`decision_log.md`。bootstrap 只负责“把主控目录点亮”，不承担阶段推进和角色结论收口。

**Tech Stack:** Go、标准库、`gopkg.in/yaml.v3`

---

### Task 1: 定义基础类型与错误模型

**Files:**
- Create: `internal/orchestrator/types.go`
- Create: `internal/orchestrator/errors.go`
- Test: `internal/orchestrator/types_test.go`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定基础模型形状**

- 为 `HumanMessage`、`RequirementSnapshot`、`StageSchedule`、`DispatchTask`、`TickResult` 写最小测试。
- 为错误类型写 `errors.Is(...)` 断言，确认可稳定识别 `planner not initialized`、`unsupported transition shape`、`invalid conclusion`、`dispatch conflict`。

Run: `go test ./internal/orchestrator -run TestTypes -v`
Expected: 先失败，提示类型或错误尚未定义。

**Step 2: 实现基础模型与错误类型**

- 定义对外输入输出模型。
- 定义稳定错误变量，避免后续靠字符串比较错误原因。
- 保持字段职责清晰，不把 role-runtime 或 recovery 的业务概念塞进 orchestrator 模型。

**Step 3: 回跑类型测试**

Run: `go test ./internal/orchestrator -run TestTypes -v`
Expected: 通过。

### Task 2: 建立 planner 文件路径与初始内容模板

**Files:**
- Create: `internal/orchestrator/layout.go`
- Create: `internal/orchestrator/render.go`
- Test: `internal/orchestrator/render_test.go`
- Reference: `docs/session/001/prd.md`
- Reference: `docs/orchestrator/001/prd.md`

**Step 1: 写失败测试，锁定 bootstrap 初始内容**

- 断言 `requirement.snapshot.md` 初始内容包含 `snapshot_version: 1`。
- 断言 `master_schedule.md` 是合法文档而不是 session placeholder。
- 断言 `task_board.md`、`task_graph.mmd`、`dispatch_log.md`、`decision_log.md` 会被创建。

Run: `go test ./internal/orchestrator -run TestRenderBootstrapFiles -v`
Expected: 先失败，说明模板内容未落地。

**Step 2: 实现 planner 文件模板构造**

- 统一管理 `planner/*.md` 路径。
- 生成合法标题、初始字段和空状态占位内容。
- 保证所有时间字段使用 UTC RFC3339。

**Step 3: 回跑渲染测试**

Run: `go test ./internal/orchestrator -run TestRenderBootstrapFiles -v`
Expected: 通过。

### Task 3: 实现 placeholder 升级与幂等 bootstrap

**Files:**
- Create: `internal/orchestrator/bootstrap.go`
- Test: `internal/orchestrator/bootstrap_test.go`
- Reference: `docs/orchestrator/001/e2e.md`

**Step 1: 写失败测试，锁定两类关键行为**

- session 001 的 planner 占位文件会被升级为真实 orchestrator 文件。
- 对已 bootstrap 的 session 再执行一次 bootstrap，不会重置已有 `schedule_version` 或覆盖已有业务内容。

Run: `go test ./internal/orchestrator -run TestBootstrap -v`
Expected: 先失败，说明升级与幂等行为尚未建立。

**Step 2: 实现最小 bootstrap 入口**

- 读取 `state/effective_config.yaml`。
- 识别“session 001 占位文本”与“已 bootstrap 协议文件”。
- 用 `*.tmp + rename` 原子替换当前态文档。

**Step 3: 回跑 bootstrap 测试**

Run: `go test ./internal/orchestrator -run TestBootstrap -v`
Expected: 通过。
