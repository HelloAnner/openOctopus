# Config 001 Plan 02: 加载器与覆盖链路

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现 `octopus.yaml`、环境变量和 flags 的统一加载与覆盖顺序，稳定产出“尚未注入默认值、尚未执行复杂校验”的原始配置对象。

**Architecture:** 通过 `loader` 层封装 `defaults < yaml < env < flags` 的固定优先级，并把 env/flags 转换逻辑集中管理，避免 CLI 自己解析一套、运行时再解析一套。加载器只负责“把多来源输入合成一份原始配置”，不负责业务校验。

**Tech Stack:** Go、`Koanf`、Cobra flags

---

### Task 1: 建立加载器入口与选项对象

**Files:**
- Create: `internal/config/loader/options.go`
- Create: `internal/config/loader/loader.go`
- Test: `internal/config/loader/loader_test.go`
- Reference: `docs/config/001/prd.md:86`
- Reference: `docs/config/001/yaml-rules.md:238`

**Step 1: 写失败测试，固定加载器 API**

- 设计 `LoadRawConfig(path string, env map[string]string, flags map[string]any)` 或等价 API。
- 为不存在文件、空路径、合法 YAML 三种场景写测试。

Run: `go test ./internal/config/loader -run TestLoadRawConfig -v`
Expected: 先失败，提示 API 或返回值未定义。

**Step 2: 实现 YAML 文件加载**

- 使用 `Koanf` 加载单文件 YAML。
- 不支持 include/import。
- 对不存在文件与语法错误返回稳定错误类型。

**Step 3: 回跑基础加载测试**

Run: `go test ./internal/config/loader -run TestLoadRawConfig -v`
Expected: 文件存在时可成功加载，文件问题可稳定失败。

### Task 2: 接入环境变量覆盖

**Files:**
- Create: `internal/config/loader/env.go`
- Modify: `internal/config/loader/loader.go`
- Test: `internal/config/loader/env_test.go`

**Step 1: 写失败测试，固定 env 规则**

- 测试 `OPENOCTOPUS_RUNTIME__TMUX__ENABLED=true` 等映射。
- 测试非法 env 值在后续解码前后会产生可识别错误。

Run: `go test ./internal/config/loader -run TestEnvOverride -v`
Expected: 先失败，提示 env 未参与合并。

**Step 2: 实现 env 覆盖**

- 统一前缀 `OPENOCTOPUS_`。
- 使用双下划线 `__` 做层级映射。
- 仅支持首版定义的标量与简单数组场景。

**Step 3: 回跑 env 覆盖测试**

Run: `go test ./internal/config/loader -run TestEnvOverride -v`
Expected: env 可以覆盖 YAML 中的目标字段。

### Task 3: 接入 flags 覆盖并锁定最终优先级

**Files:**
- Create: `internal/config/loader/flags.go`
- Modify: `internal/config/loader/options.go`
- Modify: `internal/config/loader/loader.go`
- Test: `internal/config/loader/priority_test.go`

**Step 1: 写失败测试，锁定 `defaults < yaml < env < flags`**

- 针对 `workspace.root`、`runtime.tmux.enabled`、`scheduler.max_parallel_roles` 等字段写优先级测试。

Run: `go test ./internal/config/loader -run TestOverridePriority -v`
Expected: 先失败，说明 flags 尚未生效或顺序不正确。

**Step 2: 实现 flags 覆盖**

- 只接入 `config 001` PRD 已声明的有限 flags。
- 不允许通过 flags 覆盖整段复杂对象。

**Step 3: 回跑 loader 全量测试**

Run: `go test ./internal/config/loader -v`
Expected: 加载、env 覆盖、flags 优先级测试全部通过。
