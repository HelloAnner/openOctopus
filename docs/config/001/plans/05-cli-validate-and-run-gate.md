# Config 001 Plan 05: CLI 接入与启动阻断

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 将 `config` 模块正式接入 `openoctopus validate` 与 `openoctopus run`，确保 `validate` 能稳定展示结果，`run` 能在 session 创建前阻断非法配置。

**Architecture:** CLI 层只做参数解析和结果展示，所有加载、默认值和校验都委托给 `config` 包。`validate` 和 `run` 共享同一套配置入口，避免两个命令各自维护不同配置逻辑。

**Tech Stack:** Go、Cobra

---

### Task 1: 建立 `LoadForValidate` / `LoadForRun` 服务接口

**Files:**
- Create: `internal/config/service/load_for_validate.go`
- Create: `internal/config/service/load_for_run.go`
- Modify: `internal/config/loader/loader.go`
- Modify: `internal/config/defaults/defaults.go`
- Modify: `internal/config/validator/validator.go`
- Test: `internal/config/service/service_test.go`

**Step 1: 写失败测试，锁定服务层返回值**

- `LoadForValidate` 返回 `RuntimeConfig`、`AppliedDefaults`、`[]ConfigError`。
- `LoadForRun` 在错误场景直接失败，不返回半成品配置。

Run: `go test ./internal/config/service -run TestLoadForValidateAndRun -v`
Expected: 先失败，说明服务层尚未建立。

**Step 2: 实现服务层组装**

- `LoadForValidate`：加载 -> 默认值 -> 全量校验。
- `LoadForRun`：加载 -> 默认值 -> 全量校验 -> 有错直接返回阻断错误。

**Step 3: 回跑服务层测试**

Run: `go test ./internal/config/service -v`
Expected: 服务层返回值稳定。

### Task 2: 接入 `validate` 命令

**Files:**
- Create: `cmd/openoctopus/validate.go`
- Create: `internal/config/output/validate_output.go`
- Test: `cmd/openoctopus/validate_test.go`
- Reference: `docs/config/001/e2e.md:143`

**Step 1: 写失败测试，锁定 validate 输出行为**

- 合法配置退出码为 `0`。
- 非法配置退出码非 `0`。
- 错误输出包含 category、path、suggestion。

Run: `go test ./cmd/openoctopus -run TestValidateCommand -v`
Expected: 先失败，命令未接入或输出格式不稳定。

**Step 2: 实现 validate 命令**

- 读取 `--config`。
- 调用 `LoadForValidate`。
- 成功时打印校验通过与默认值摘要。
- 失败时打印结构化错误摘要并返回非 `0`。

**Step 3: 回跑 validate 命令测试**

Run: `go test ./cmd/openoctopus -run TestValidateCommand -v`
Expected: 通过。

### Task 3: 接入 `run` 前置阻断

**Files:**
- Create: `cmd/openoctopus/run.go`
- Modify: `internal/session/...`
- Test: `cmd/openoctopus/run_test.go`
- Reference: `docs/config/001/e2e.md:157`

**Step 1: 写失败测试，锁定“非法配置不创建 session”**

- 用非法 YAML 执行 `run`，断言 session 目录未创建。

Run: `go test ./cmd/openoctopus -run TestRunCommand_ConfigGate -v`
Expected: 先失败，说明 run 尚未在 session 创建前阻断。

**Step 2: 实现 run 的前置配置门禁**

- 在任何 session 初始化之前调用 `LoadForRun`。
- 只有配置合法才继续进入后续流程。

**Step 3: 回跑 CLI 全量测试**

Run: `go test ./cmd/openoctopus -v`
Expected: validate 与 run 的配置入口行为一致。
