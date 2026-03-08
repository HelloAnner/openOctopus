# OpenOctopus `octopus.yaml` 构建帮助文档

> 本文档用于后续**人工/AI 自动构建 `octopus.yaml`**。
> 如与历史 README、旧版设计稿、旧 PRD 示例冲突，**一律以当前仓库真实实现为准**。

## 1. 文档定位

这份文档不是泛泛的 YAML 介绍，而是给配置生成器使用的“当前可落地协议”。

建议按下面的优先级理解本文件：

1. `internal/config/model/`：决定当前真实支持哪些字段。
2. `internal/config/defaults/defaults.go`：决定哪些字段会被自动补默认值。
3. `internal/config/validator/validator.go`：决定哪些字段会被强校验。
4. `cmd/openoctopus/validate.go`、`cmd/openoctopus/run.go`：决定当前支持哪些 flags 覆盖。
5. `docs/prd/prd-001.md`、`docs/config/001/yaml-rules.md`：提供设计背景，但其中有部分字段已经与当前实现漂移。

**生成器必须遵守的核心原则：**

- 只生成本文档中明确列出的字段。
- 不要依赖“未知字段会报错”这一点，因为**当前实现对未知字段并不严格阻断，部分未知键可能被静默忽略**。
- 优先生成“当前稳定子集”，不要主动输出历史文档里提过、但当前代码未建模的字段。
- 生成完成后，必须执行：

```bash
./openoctopus validate --config ./octopus.yaml
```

## 2. 当前真实加载与覆盖规则

当前配置加载优先级如下：

```text
defaults < octopus.yaml < env < flags
```

### 2.1 YAML 文件

主配置源就是 `octopus.yaml`，应承载完整流程结构：

- `meta`
- `runtime`
- `llm_profiles`
- `tool_registry`
- `security`
- `policies`
- `roles`
- `stages`
- `transitions`

### 2.2 环境变量覆盖

当前环境变量规则：

- 前缀固定：`OPENOCTOPUS_`
- 层级分隔：双下划线 `__`
- 路径会被转成小写并映射到配置路径

例如：

```bash
OPENOCTOPUS_RUNTIME__TMUX__ENABLED=true
OPENOCTOPUS_RUNTIME__SCHEDULER__MAX_PARALLEL_ROLES=2
OPENOCTOPUS_RUNTIME__WORKSPACE__ROOT=.octopus-ci
```

当前标量解析规则：

- `true` / `false` -> 布尔
- 整数文本 -> `int`
- 含小数点的数字 -> `float64`
- 含逗号 -> 字符串数组
- 其他情况 -> 字符串

**建议：**

- env 只覆盖运行环境相关的简单字段。
- 不要用 env 构建整个 `roles`、`stages`、`transitions` 结构。
- 不要把 env 当成主配置协议。

### 2.3 Flags 覆盖

当前 CLI 真正支持的 flags 只有两个：

```bash
./openoctopus validate --config ./octopus.yaml --workspace-root .octopus-ci --max-parallel-roles 2
./openoctopus run --config ./octopus.yaml --workspace-root .octopus-ci --max-parallel-roles 2
```

对应路径：

- `--workspace-root` -> `runtime.workspace.root`
- `--max-parallel-roles` -> `runtime.scheduler.max_parallel_roles`

注意：`docs/prd/prd-001.md` 里提到过更多 flags 规划，但**当前代码只实现了上面两个**。

## 3. 生成器必须遵守的硬规则

### 3.1 固定版本

`version` 当前必须固定为字符串：

```yaml
version: "2.1"
```

不要输出：

- `version: 2.1`
- `version: "v2.1"`
- 其他任意版本号

### 3.2 顶层结构固定

当前推荐只生成以下顶层 key：

```yaml
version:
meta:
runtime:
llm_profiles:
tool_registry:
security:
policies:
roles:
stages:
transitions:
```

其中：

- `runtime` / `security` / `policies` 可以省略部分子字段。
- `llm_profiles` 必须是 **map/object**。
- `tool_registry.builtin` / `tool_registry.mcp` 必须是 **map/object**。
- `roles` 必须是 **数组**。
- `stages` 必须是 **数组**。
- `transitions` 必须是 **数组**。

### 3.3 生成器应优先使用的稳定子集

虽然模型里有一些“自由度较高”的字段，但为了稳定，建议生成器优先使用以下安全子集：

- `roles[*].type` 统一生成 `react`
- `llm_profiles.*.mode` 统一生成 `cli`
- `stages[*].output[*].type` 优先使用 `artifact`
- `stages[*].input[*].type` 优先使用 `artifact`
- `runtime.tmux.role_layout` 仅使用 `adaptive_grid` 或 `tiled`
- `stages[*].mode` 只在需要时使用 `session_reset`，否则省略

## 4. 当前支持的真实 Schema

下面不是 JSON Schema，而是更适合生成器消费的结构说明。

```yaml
version: string
meta:
  workflow_id: string
  name: string
  owner?: string
runtime?:
  workspace?:
    root?: string
    sessions_dir?: string
    artifacts_dir?: string
    logs_dir?: string
  scheduler?:
    max_parallel_roles?: int
    dispatch_strategy?: string
  tmux?:
    enabled?: bool
    socket_name?: string
    main_pane_ratio?: float
    role_layout?: string
  role_runtime?:
    trigger_mode?: string
    idle_poll_seconds?: int
    bootstrap_read_order?: string[]
    safe_interrupt_boundary?: string
  master_watch?:
    enabled?: bool
    progress_file?: string
    blocker_file?: string
    auto_interrupt_all_on_deadlock?: bool
    max_no_progress_rounds?: int
llm_profiles:
  <profile_id>:
    provider: string
    mode: string
    cli_path?: string
    tmux_command?: string
    max_tokens?: int
    temperature?: float
tool_registry:
  builtin?:
    <tool_id>:
      module: string
      class: string
  mcp?:
    <tool_id>:
      command: string
      args?: string[]
      env?:
        <env_name>: string
security?:
  shell?:
    allowlist_prefixes?: string[]
    denylist_keywords?: string[]
  path?:
    writable_roots?: string[]
    read_only_roots?: string[]
policies?:
  retry?:
    max_retry_per_stage?: int
    backoff_seconds?: int[]
  timeout?:
    stage_timeout_seconds?: int
    role_heartbeat_timeout_seconds?: int
  loop_guard?:
    max_rounds_per_task?: int
    min_quality_gain?: float
  human_gate?:
    on_high_risk?: bool
    high_risk_threshold?: float
    write_manual_input_to_markdown?: bool
    main_agent_ack_required?: bool
  session_reset?:
    enabled?: bool
    preserve_files?: bool
    keep_turn_history?: bool
  review?:
    require_design_approval?: bool
    require_code_doc_diff?: bool
  artifact?:
    hash_algo?: string
    keep_latest_versions?: int
  immutable_artifacts?:
    paths?: string[]
    allow_writers?: string[]
  deadlock_guard?:
    enabled?: bool
    max_no_progress_rounds?: int
    blocked_statuses?: string[]
    on_trigger?: string
roles:
  - id: string
    name: string
    type: string
    llm_profile: string
    system_prompt: string
    reset_prompt?: string
    react_config?: object
    constraints?:
      must_read_files?: string[]
      forbidden_writes?: string[]
    tools?: string[]
stages:
  - id: string
    name: string
    role: string
    mode?: string
    clear_cli_context?: bool
    preserve?:
      artifacts?: string[]
    input?:
      - type?: string
        ref?: string
        name?: string
        path?: string
        access?: string
    output?:
      - type?: string
        ref?: string
        name?: string
        path?: string
        access?: string
transitions:
  - from: string
    to?: string
    repeat?:
      max_rounds?: int
      on_complete?: string
    condition?:
      type?: string
      expr?: string
      mode?: string
      rules?: string[]
    on_true?: string
    on_false?: string
```

## 5. 必填字段与最小可运行集合

一个最小可通过 `validate` 的配置，至少需要这些信息：

- `version: "2.1"`
- `meta.workflow_id`
- `meta.name`
- 至少 1 个 `llm_profiles.<profile_id>`
  - `provider`
  - `mode`
  - 若 `mode=cli`，则还要有 `cli_path`
- 至少 1 个注册工具
  - `tool_registry.builtin` 或 `tool_registry.mcp` 任一非空即可
- 至少 1 个角色
  - `id`
  - `name`
  - `type`
  - `llm_profile`
  - `system_prompt`
- 至少 1 个阶段
  - `id`
  - `name`
  - `role`
- 至少 1 条流转
  - `from`
  - 推荐再提供 `to` 或 `on_true/on_false`

## 6. 默认值注入规则

当前代码会自动补以下默认值。生成器可以省略它们，但为了配置更显式，也可以直接写出。

### 6.1 `runtime.workspace`

- `runtime.workspace.root` -> `.octopus`
- `runtime.workspace.sessions_dir` -> `.octopus/sessions`
- `runtime.workspace.artifacts_dir` -> `.octopus/artifacts`
- `runtime.workspace.logs_dir` -> `.octopus/logs`

### 6.2 `runtime.tmux`

- `runtime.tmux.socket_name` -> `octopus-{session_id}`
- `runtime.tmux.main_pane_ratio` -> `0.5`
- `runtime.tmux.role_layout` -> `adaptive_grid`

### 6.3 `runtime.role_runtime`

- `runtime.role_runtime.idle_poll_seconds` -> `2`

### 6.4 `runtime.master_watch`

- `runtime.master_watch.enabled` -> `true`
- `runtime.master_watch.progress_file` -> `planner/global_progress.md`
- `runtime.master_watch.blocker_file` -> `planner/blockers.md`
- `runtime.master_watch.max_no_progress_rounds` -> `3`

### 6.5 `policies`

- `policies.retry.max_retry_per_stage` -> `2`
- `policies.retry.backoff_seconds` -> `[5, 20]`
- `policies.timeout.stage_timeout_seconds` -> `1800`
- `policies.timeout.role_heartbeat_timeout_seconds` -> `120`
- `policies.loop_guard.max_rounds_per_task` -> `6`
- `policies.loop_guard.min_quality_gain` -> `0.05`

## 7. 当前真实校验规则

这一节只写“当前代码确实会检查”的规则，适合作为生成器的硬约束。

### 7.1 根级校验

- `version` 必须等于 `"2.1"`
- `meta.workflow_id` 不能为空
- `meta.name` 不能为空
- `llm_profiles` 不能为空
- `tool_registry` 至少要注册一个工具
- `roles` 不能为空
- `stages` 不能为空
- `transitions` 不能为空
- `runtime.scheduler.max_parallel_roles` 如果声明：
  - 不能小于 `0`
  - 一旦显式配置，就必须大于 `0`

### 7.2 `runtime.tmux`

- 若 `runtime.tmux.enabled=true`，则 `socket_name` 不能为空
- `main_pane_ratio` 必须位于 `(0, 1)`
- `role_layout` 只能是：
  - `adaptive_grid`
  - `tiled`

### 7.3 `llm_profiles`

每个 profile：

- `provider` 不能为空
- `mode` 不能为空
- 当 `mode=cli` 时，`cli_path` 不能为空
- `tmux_command` 若声明：
  - 仅适用于 `mode=cli`
  - 若使用完整模板，支持占位符：`{session_dir}`、`{role_id}`、`{prompt}`
- `max_tokens` 若声明，不能为负数

### 7.4 `roles`

每个角色：

- `id` 不能为空，且全局唯一
- `name` 不能为空
- `type` 不能为空
- `llm_profile` 不能为空，且必须引用已存在的 profile
- `system_prompt` 不能为空
- `tools[*]` 若声明，必须全部在 `tool_registry` 中已注册

### 7.5 `stages`

每个阶段：

- `id` 不能为空，且全局唯一
- `name` 不能为空
- `role` 必须引用已定义角色
- 若 `mode=session_reset`：
  - `clear_cli_context` 必须为 `true`
  - `preserve.artifacts` 不能为空
- 若 `input[*].type=artifact` 且填写了 `ref`：
  - `ref` 必须引用**前序阶段已经产出**的 artifact 名称

### 7.6 `transitions`

- `transitions[*].from` 必须引用已存在阶段
- `transitions[*].to`、`on_true`、`on_false` 若填写：
  - 必须引用已存在阶段，或者使用 `__END__`
- 当前稳定支持的循环语义只有：

```yaml
repeat:
  max_rounds: 3
  on_complete: "__END__"
```

- `repeat.max_rounds` 若启用，必须大于 `0`
- `repeat.on_complete` 若启用，必须引用合法出口阶段或 `__END__`
- 当前 `repeat` 不是运行时动态判断，而是 orchestrator 在 graph build 阶段静态展开

### 7.7 `security`

- 只要任意角色使用了 `shell_exec` 工具，`security.shell.allowlist_prefixes` 就必须非空

### 7.8 `policies`

- `runtime.master_watch.max_no_progress_rounds` 若显式配置：
  - 不能小于 `0`
  - 一旦显式配置，就必须大于 `0`
- `policies.deadlock_guard.max_no_progress_rounds` 若显式配置：
  - 不能小于 `0`
  - 一旦显式配置，就必须大于 `0`
- `policies.loop_guard.max_rounds_per_task` 若显式配置：
  - 不能小于 `0`
  - 一旦显式配置，就必须大于 `0`

### 7.9 `immutable_artifacts`

如果配置了：

```yaml
policies:
  immutable_artifacts:
    paths: [...]
    allow_writers: [...]
```

那么：

- 不在 `allow_writers` 中的角色，必须通过 `constraints.forbidden_writes` 阻断这些只读路径
- 当前校验是**按字符串完全相等**判断的，所以生成器应让：
  - `policies.immutable_artifacts.paths[*]`
  - 与各角色 `constraints.forbidden_writes[*]`
  使用**完全一致的路径模式字符串**

## 8. 生成器推荐构建步骤

推荐按下面顺序组装 `octopus.yaml`：

1. 先写固定头：`version`、`meta`
2. 决定 LLM 供应方式，生成 `llm_profiles`
3. 决定角色能力，生成 `tool_registry`
4. 按职责生成 `roles`
5. 按执行顺序生成 `stages`
6. 根据阶段关系生成 `transitions`
7. 只有在确实需要时，再补 `runtime` / `security` / `policies`
8. 输出后立刻运行 `validate`

### 8.1 角色生成建议

如果没有更细的业务约束，建议从以下角色模式中选择：

- 方案角色：偏 `file_read` / `file_write` / `git_operation`
- 实现角色：通常再加 `shell_exec`
- 审核角色：通常不要给过宽写权限
- E2E 角色：如果会写测试产物，建议配合 `immutable_artifacts` 使用

### 8.2 阶段生成建议

推荐把阶段设计为显式线性或显式分支，而不是把分支塞到 stage 内部：

- stage 只描述“谁执行、读什么、产出什么”
- transition 单独描述“下一步怎么走”

## 9. 与旧文档冲突时的处理规则

这部分非常重要，目的是避免生成器被仓库里历史示例误导。

### 9.1 README 里的旧示例不要照搬

README 当前示例已经和真实模型漂移，主要问题包括：

- 把 `roles` 写成了对象，而当前真实结构要求 `roles` 是数组
- 用了 `roles.profile`，而当前真实字段是 `roles[*].llm_profile`
- 把阶段流转写在 `stages[*].transition.next` 里，而当前真实结构要求单独使用顶层 `transitions`

### 9.2 旧设计文档里提到但当前未建模的字段，不要生成

以下字段在历史文档里出现过，但当前 `internal/config/model/` **并没有对应结构**，生成器不要输出：

- `runtime.checkpoint.*`
- `runtime.human_io.*`
- `stages[*].reset_session.*`
- `roles[*].constraints.emit_read_only_artifacts`
- `stages[*].input[*].optional`

### 9.3 当前“已建模但校验较弱”的字段

以下字段当前模型支持，但校验并不强；只有在明确需求下再输出：

- `runtime.scheduler.dispatch_strategy`
- `runtime.role_runtime.trigger_mode`
- `runtime.role_runtime.bootstrap_read_order`
- `runtime.role_runtime.safe_interrupt_boundary`
- `security.path.*`
- `policies.human_gate.*`
- `policies.session_reset.*`
- `policies.review.*`
- `policies.artifact.*`
- `policies.deadlock_guard.blocked_statuses`
- `policies.deadlock_guard.on_trigger`
- `roles[*].react_config`
- `stages[*].input[*].path`
- `stages[*].input[*].access`
- `stages[*].output[*].path`

### 9.4 tmux / repeat 常见坑点（非常重要）

#### 坑点 1：`tmux_command` 如果写“半模板半默认”，容易触发重复参数

当前 tmux role pane 启动逻辑分两种：

1. **简写模式**：`tmux_command` 不包含占位符时，`codex + cli` 会自动追加：
   - `--skip-git-repo-check`
   - `--no-alt-screen`
   - `-C {session_dir}`
   - `{prompt}`
2. **完整模板模式**：只要 `tmux_command` 包含任意占位符：
   - `{session_dir}`
   - `{role_id}`
   - `{prompt}`
   就视为“完全自定义”，运行时不再补默认参数。

因此如果你已经明确知道自己要什么参数，**必须优先使用完整模板模式**。

另外，`tmux_command` 是通过 pane 内 shell 执行的，**会受到 alias / shell wrapper 影响**；
这和 `llm_profiles.*.cli_path` 不同，`cli_path` 在 role-runtime 中是 `exec.Command(...)` 直调二进制，不会吃到交互 shell alias。

推荐写法：

```yaml
llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"
    tmux_command: "/usr/bin/env codex --dangerously-bypass-approvals-and-sandbox --no-alt-screen -C {session_dir} {prompt}"
```

不要写成：

```yaml
tmux_command: "codex exec --skip-git-repo-check ..."
```

否则非常容易和运行时自动补参叠加，或者命中本机 shell alias，出现类似：

- `the argument '--skip-git-repo-check' cannot be used multiple times`
- `unexpected argument '--no-alt-screen' found`

在当前宿主机上，交互式 `zsh` 的真实情况就是：

```bash
alias codex='codex exec --skip-git-repo-check'
```

所以：

- `tmux_command: "codex ..."` 可能被 alias 篡改成 `codex exec ...`
- `tmux_command: "/usr/bin/env codex ..."` 可以稳定绕过 alias
- `tmux_command` 里如果使用 `{session_dir}`，必须确保它最终展开为**绝对路径**；相对路径在 pane 内先 `cd` 后再传给 `-C` 时，极易变成错误的嵌套路径并触发 `No such file or directory (os error 2)`

#### 坑点 2：main pane 不是 agent 交互页

当前实现里：

- `main` pane 只负责显示 session banner / 总览入口
- 真正的 agent 交互页在各个 `role pane`

所以看到 main pane 出现类似：

```text
printf '[openoctopus] main session=...'
[openoctopus] main session=...
```

这通常是 bootstrap banner 的正常现象，不代表主 pane 里会自动进入某个 agent。

#### 坑点 3：`repeat` 当前只支持“固定轮次自动收口”

当前稳定支持的是：

- 两个或少量阶段形成**唯一有界回路**
- 通过 `repeat.max_rounds + repeat.on_complete` 静态展开

当前**不要**生成以下形态：

- `condition/on_true/on_false` 和 `repeat` 混用
- 多个回路同时存在
- 多 entry + 回路混合图
- 依赖运行时表达式判断是否继续循环

#### 坑点 4：entry stage 不能直接引用 loop-back artifact

当前 `STAGE-006` 的真实校验规则是：

- `input.type=artifact` 的 `ref` 必须引用**在 YAML 阶段顺序里更早声明的 stage output**

这意味着下面这种首轮 loop-back 写法当前不稳定：

```yaml
stages:
  - id: "split_prd"
    input:
      - type: "artifact"
        ref: "review_feedback"
```

如果 `review_feedback` 是后面 `review_prd` 才产出的 artifact，那么 `validate` 会直接失败。

当前稳定做法：

1. entry stage 只读取原始 `requirement_file` / 其他首轮一定存在的输入。
2. review 结果先作为 artifact 落盘，供人工查看、tmux 观察和后续扩展使用。
3. 如果后续要实现“第 N 轮 review_feedback 自动喂回第 N+1 轮 splitter”，需要单独扩展 artifact loop-back 语义，不要假设当前实现已经支持。

#### 坑点 5：修改 YAML 后，旧 tmux session 不会热更新

tmux pane 启动命令是在 `run` 创建 session 时一次性注入的。

这意味着：

- 你修改了 `octopus.yaml` / `config/dev.yaml`
- 或者你更新了二进制代码
- **已经创建好的旧 session 不会自动应用这些变化**

如果你仍在旧 session 里观察 pane 输出，看到的仍然会是旧命令行为。

正确做法：

1. 结束旧 session（或至少不要继续拿旧 session 验证新配置）。
2. 用新配置重新执行一次 `run`，确认拿到一个新的 `session_id`。
3. 只对新的 session 验证 tmux pane 行为。

#### 坑点 6：`oo` 可能不是当前仓库最新代码构建出来的二进制

如果你在终端里执行的是：

```bash
oo run --config config/dev.yaml
```

要特别注意：当前 shell 里的 `oo` 可能是之前安装的旧二进制，而不是当前仓库工作区的最新代码。

因此会出现一种假象：

- 仓库里的 `config/dev.yaml` 和代码已经修好了
- 但你实际执行的 `oo` 仍然在跑旧逻辑

建议排查方式：

```bash
which oo
type oo
```

如果只是想验证当前仓库代码，优先直接用仓库二进制或 `go run`，不要先假设全局 `oo` 已经同步：

```bash
go build -o openoctopus ./cmd/openoctopus
./openoctopus run --config config/dev.yaml
```
- `stages[*].output[*].access`
- `transitions[*].condition.*`

这些字段不是“不能用”，而是“当前运行时对它们的语义约束还不够严格”，因此生成器应保守使用。

## 10. 可直接复制的 YAML 示例

### 10.1 最小可通过示例

这个例子适合：

- 冒烟测试
- 本地最小验证
- 自动生成器的基础回归用例

```yaml
version: "2.1"

meta:
  workflow_id: "minimal-demo"
  name: "Minimal Demo"

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"

roles:
  - id: "agent_a"
    name: "Agent A"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责执行任务。"
    tools: ["file_read"]

stages:
  - id: "stage_a"
    name: "Stage A"
    role: "agent_a"
    output:
      - type: "artifact"
        name: "artifact_a"

transitions:
  - from: "stage_a"
    to: "__END__"
```

### 10.2 线性多阶段示例

这个例子适合：

- 方案 -> 实现 -> 审核 的标准链路
- 不需要 session reset
- 不需要 tmux

```yaml
version: "2.1"

meta:
  workflow_id: "linear-dev-flow"
  name: "Linear Dev Flow"
  owner: "team-openoctopus"

llm_profiles:
  claude_cli:
    provider: "claude_code"
    mode: "cli"
    cli_path: "claude"
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"
    file_write:
      module: "openoctopus.tools.file"
      class: "FileWriteTool"
    git_operation:
      module: "openoctopus.tools.git"
      class: "GitTool"

roles:
  - id: "planner"
    name: "Planner"
    type: "react"
    llm_profile: "claude_cli"
    system_prompt: "你负责整理需求并输出方案。"
    tools: ["file_read", "file_write", "git_operation"]

  - id: "engineer"
    name: "Engineer"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责根据方案实现代码。"
    tools: ["file_read", "file_write", "git_operation"]

  - id: "reviewer"
    name: "Reviewer"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责审核实现结果。"
    tools: ["file_read", "file_write", "git_operation"]

stages:
  - id: "plan"
    name: "Plan"
    role: "planner"
    output:
      - type: "artifact"
        name: "solution_doc"

  - id: "implement"
    name: "Implement"
    role: "engineer"
    input:
      - type: "artifact"
        ref: "solution_doc"
    output:
      - type: "artifact"
        name: "code_patch"

  - id: "review"
    name: "Review"
    role: "reviewer"
    input:
      - type: "artifact"
        ref: "solution_doc"
      - type: "artifact"
        ref: "code_patch"
    output:
      - type: "artifact"
        name: "review_report"

transitions:
  - from: "plan"
    to: "implement"
  - from: "implement"
    to: "review"
  - from: "review"
    to: "__END__"
```

### 10.3 高级示例：tmux + shell + session_reset + immutable_artifacts

这个例子适合：

- 多角色协作
- 审核回路
- 需要重置实现角色上下文
- 需要保护 E2E 产物只读
- 需要 tmux 布局

```yaml
version: "2.1"

meta:
  workflow_id: "advanced-review-loop"
  name: "Advanced Review Loop"
  owner: "team-openoctopus"

runtime:
  workspace:
    root: ".octopus"
  scheduler:
    max_parallel_roles: 1
  tmux:
    enabled: true
    socket_name: "octopus-{session_id}"
    main_pane_ratio: 0.5
    role_layout: "adaptive_grid"
  role_runtime:
    trigger_mode: "fs_event_then_poll"
    idle_poll_seconds: 2
    bootstrap_read_order: ["context.md", "inbox.md"]
    safe_interrupt_boundary: "turn_boundary"
  master_watch:
    enabled: true
    progress_file: "planner/global_progress.md"
    blocker_file: "planner/blockers.md"
    auto_interrupt_all_on_deadlock: true
    max_no_progress_rounds: 3

llm_profiles:
  claude_cli:
    provider: "claude_code"
    mode: "cli"
    cli_path: "claude"
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"
    file_write:
      module: "openoctopus.tools.file"
      class: "FileWriteTool"
    shell_exec:
      module: "openoctopus.tools.shell"
      class: "ShellTool"
    git_operation:
      module: "openoctopus.tools.git"
      class: "GitTool"

security:
  shell:
    allowlist_prefixes: ["git status", "git diff", "go test", "pytest"]
    denylist_keywords: ["rm -rf /", "shutdown", "reboot"]
  path:
    writable_roots: ["./", ".octopus/"]
    read_only_roots: ["/etc", "/usr"]

policies:
  retry:
    max_retry_per_stage: 2
    backoff_seconds: [5, 20]
  timeout:
    stage_timeout_seconds: 1800
    role_heartbeat_timeout_seconds: 120
  loop_guard:
    max_rounds_per_task: 6
    min_quality_gain: 0.05
  human_gate:
    on_high_risk: true
    high_risk_threshold: 0.8
    write_manual_input_to_markdown: true
    main_agent_ack_required: true
  session_reset:
    enabled: true
    preserve_files: true
    keep_turn_history: true
  review:
    require_design_approval: true
    require_code_doc_diff: true
  artifact:
    hash_algo: "sha256"
    keep_latest_versions: 5
  immutable_artifacts:
    paths:
      - "artifacts/{session_id}/tests/e2e/**"
    allow_writers: ["qa_agent"]
  deadlock_guard:
    enabled: true
    max_no_progress_rounds: 3
    blocked_statuses: ["BLOCKED", "INTERRUPTED_PENDING_ACK"]
    on_trigger: "interrupt_all_and_wait_human"

roles:
  - id: "planner"
    name: "Planner"
    type: "react"
    llm_profile: "claude_cli"
    system_prompt: "你负责拆分任务并输出方案。"
    constraints:
      forbidden_writes: ["artifacts/{session_id}/tests/e2e/**"]
    tools: ["file_read", "file_write", "git_operation"]

  - id: "engineer"
    name: "Engineer"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责根据方案实现代码。"
    reset_prompt: "只保留批准后的方案和审核结论。"
    constraints:
      forbidden_writes: ["artifacts/{session_id}/tests/e2e/**"]
    tools: ["file_read", "file_write", "shell_exec", "git_operation"]

  - id: "reviewer"
    name: "Reviewer"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责审核实现结果。"
    constraints:
      forbidden_writes: ["artifacts/{session_id}/tests/e2e/**"]
    tools: ["file_read", "file_write", "git_operation"]

  - id: "qa_agent"
    name: "QA Agent"
    type: "react"
    llm_profile: "claude_cli"
    system_prompt: "你负责产出和维护 E2E 测试。"
    tools: ["file_read", "file_write", "shell_exec"]

stages:
  - id: "plan"
    name: "Plan"
    role: "planner"
    output:
      - type: "artifact"
        name: "solution_doc"

  - id: "implement"
    name: "Implement"
    role: "engineer"
    input:
      - type: "artifact"
        ref: "solution_doc"
    output:
      - type: "artifact"
        name: "code_patch"

  - id: "review"
    name: "Review"
    role: "reviewer"
    input:
      - type: "artifact"
        ref: "solution_doc"
      - type: "artifact"
        ref: "code_patch"
    output:
      - type: "artifact"
        name: "review_report"

  - id: "reset_engineer"
    name: "Reset Engineer"
    role: "engineer"
    mode: "session_reset"
    clear_cli_context: true
    preserve:
      artifacts: ["solution_doc", "review_report"]

  - id: "e2e"
    name: "E2E"
    role: "qa_agent"
    input:
      - type: "artifact"
        ref: "solution_doc"
      - type: "artifact"
        ref: "code_patch"
    output:
      - type: "artifact"
        name: "e2e_suite"

transitions:
  - from: "plan"
    to: "implement"
  - from: "implement"
    to: "review"
  - from: "review"
    condition:
      type: "expression"
      expr: "artifacts.review_report.approved == true"
    on_true: "reset_engineer"
    on_false: "implement"
  - from: "reset_engineer"
    to: "e2e"
  - from: "e2e"
    to: "__END__"
```

## 11. 自动生成器输出前自检清单

生成器在落盘前，至少做下面这些自检：

- `version` 是否为 `"2.1"`
- `roles` / `stages` / `transitions` 是否都是数组
- `roles[*].llm_profile` 是否都能在 `llm_profiles` 找到
- `roles[*].tools[*]` 是否都能在 `tool_registry` 找到
- `stages[*].role` 是否都能在 `roles` 找到
- `artifact` 输入引用是否只依赖前序已产出 artifact
- `transitions` 的目标阶段是否存在，或是否为 `__END__`
- 若用了 `shell_exec`，是否同时给了 `security.shell.allowlist_prefixes`
- 若用了 `mode: session_reset`，是否同时给了：
  - `clear_cli_context: true`
  - `preserve.artifacts`
- 若用了 `immutable_artifacts`，非授权角色是否都加了相同路径到 `forbidden_writes`

## 12. 推荐验证命令

### 12.1 只校验配置

```bash
./openoctopus validate --config ./octopus.yaml
```

### 12.2 带覆盖项校验

```bash
./openoctopus validate --config ./octopus.yaml --workspace-root .octopus-ci --max-parallel-roles 2
```

### 12.3 环境变量覆盖校验

```bash
OPENOCTOPUS_RUNTIME__TMUX__ENABLED=true \
OPENOCTOPUS_RUNTIME__TMUX__SOCKET_NAME='octopus-{session_id}' \
./openoctopus validate --config ./octopus.yaml
```

---

如果后续要基于这份文档自动生成 YAML，**建议把“第 3 节硬规则 + 第 4 节 Schema + 第 7 节真实校验规则 + 第 11 节自检清单”作为生成器主协议**，把示例作为 few-shot 模板，而不是把历史 README 示例当模板。
