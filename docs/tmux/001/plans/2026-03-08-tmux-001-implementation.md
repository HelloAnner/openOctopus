# TMUX 001 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 为 OpenOctopus 落地首版 tmux 模块，让 `run` 能在启用配置后创建真实 tmux 布局，并提供稳定的 `switch` 能力与黑盒 E2E 验证。

**Architecture:** 新增 `internal/tmux` 独立承载 tmux 会话创建、布局落盘、pane 解析与切换；`cmd/openoctopus/run.go` 只在 session 创建后按配置调用 bootstrap；新增 `cmd/openoctopus/switch.go` 复用 layout 协议做目标解析与可选切换，不让 CLI 直接拼装杂乱的 tmux 命令。

**Tech Stack:** Go、Cobra、宿主机真实 `tmux`、pytest、现有 session / config / CLI 模块。

---

### Task 1: `runtime.tmux` 配置模型与校验

**Files:**
- Modify: `internal/config/model/runtime.go`
- Modify: `internal/config/defaults/defaults.go`
- Modify: `internal/config/validator/validator.go`
- Modify: `internal/config/service/service_test.go`

**Step 1: Write the failing tests**
- 覆盖 `runtime.tmux` 默认值注入。
- 覆盖非法 `main_pane_ratio` 与非法 `role_layout` 的校验错误。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/config/service ./internal/config/validator -run 'TestLoadForValidate|TestValidate.*Tmux' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 新增 `TmuxConfig`。
- 注入默认值：`enabled=false`、`socket_name=octopus-{session_id}`、`main_pane_ratio=0.5`、`role_layout=adaptive_grid`。
- 增加静态校验。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/config/service ./internal/config/validator -run 'TestLoadForValidate|TestValidate.*Tmux' -v`
- Expected: PASS。

### Task 2: tmux layout 协议与底层服务

**Files:**
- Create: `internal/tmux/types.go`
- Create: `internal/tmux/service.go`
- Create: `internal/tmux/layout.go`
- Create: `internal/tmux/command.go`
- Create: `internal/tmux/service_test.go`

**Step 1: Write the failing tests**
- 覆盖 socket 名称模板展开。
- 覆盖 `layout.md` 渲染与解析。
- 覆盖 `switch` 目标查找失败分支。

**Step 2: Run tests to verify they fail**
- Run: `go test ./internal/tmux -run 'TestExpand|TestRender|TestRead|TestResolve' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 提供 `Bootstrap`、`ReadLayout`、`ResolveTarget`、`CapturePane`、`KillSession`。
- 用小接口封装 tmux 命令执行，测试里用 fake runner。

**Step 4: Run tests to verify they pass**
- Run: `go test ./internal/tmux -run 'TestExpand|TestRender|TestRead|TestResolve' -v`
- Expected: PASS。

### Task 3: `run` 与 `switch` 命令接线

**Files:**
- Modify: `cmd/openoctopus/root.go`
- Modify: `cmd/openoctopus/run.go`
- Create: `cmd/openoctopus/switch.go`
- Modify: `cmd/openoctopus/errors.go`
- Modify: `cmd/openoctopus/command_test.go`

**Step 1: Write the failing tests**
- 覆盖 `run` 在 `runtime.tmux.enabled=true` 时生成 `state/tmux/layout.md`。
- 覆盖 `switch --format json` 返回目标 pane 信息。

**Step 2: Run tests to verify they fail**
- Run: `go test ./cmd/openoctopus -run 'TestRunCommand.*Tmux|TestSwitch' -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- `run` 中按配置执行 tmux bootstrap，并在后续 bootstrap 失败时清理 tmux session。
- root command 接入 `switch`。
- `switch` 在非 tmux 客户端环境只返回目标，不主动 attach。

**Step 4: Run tests to verify they pass**
- Run: `go test ./cmd/openoctopus -run 'TestRunCommand.*Tmux|TestSwitch' -v`
- Expected: PASS。

### Task 4: tmux 真实 E2E

**Files:**
- Create: `e2e/tmux/fixtures/valid-basic-layout/octopus.yaml`
- Create: `e2e/tmux/fixtures/invalid-layout-config/octopus.yaml`
- Create: `e2e/tmux/test_tmux_layout.py`
- Modify: `e2e/README.md`
- Modify: `Makefile`

**Step 1: Write the failing tests**
- 覆盖 tmux session 创建、layout 文件、pane 标题、pane capture、`switch --format json`、非法配置阻断。

**Step 2: Run tests to verify they fail**
- Run: `python3 -m pytest e2e/tmux -v`
- Expected: FAIL。

**Step 3: Write minimal implementation**
- 补 fixtures。
- 补 pytest 黑盒测试。
- 接入 `e2e` 目标与 README。

**Step 4: Run tests to verify they pass**
- Run: `python3 -m pytest e2e/tmux -v`
- Expected: PASS。

### Task 5: 全量验证

**Files:**
- Modify: `README.md`
- Modify: `docs/timeline.md`

**Step 1: Run focused Go tests**
- Run: `go test ./internal/tmux ./cmd/openoctopus ./internal/config/service ./internal/config/validator -v`

**Step 2: Run tmux E2E**
- Run: `python3 -m pytest e2e/tmux -v`

**Step 3: Run full repo checks**
- Run: `make check`
- Expected: PASS。

**Step 4: Sync docs tree references if changed**
- 更新 `README.md`、`e2e/README.md`、`docs/timeline.md` 中涉及 tmux 的目录与命令说明。

**Note:** 仓库规则禁止 `git add` / `git commit`，因此本计划故意省略提交步骤。
