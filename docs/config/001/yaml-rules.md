# OpenOctopus Config 001 YAML 规则文档

## 1. 文档定位

本文档是 OpenOctopus `config 001` 阶段的**配置生成协议**。它不是泛泛介绍 YAML 的说明文档，而是给人工与 AI 共同使用的正式规则集。

推荐使用方式：

1. 先阅读本文档，确认要表达的工作流结构。
2. 按本文档规则生成 `octopus.yaml`。
3. 执行 `openoctopus validate --config ./octopus.yaml`。
4. 若报错，优先根据错误中的 `rule_id` 回到本文档修正。
5. 校验通过后，再执行 `openoctopus run`。

约束原则：

- 文档允许的写法，运行时必须允许。
- 文档禁止的写法，运行时必须阻断。
- 新增 YAML 能力时，必须先更新本文档，再开放运行时支持。

## 2. 规则编号约定

每条规则都有稳定编号，建议运行时错误直接回传该编号：

- `ROOT-*`：顶层结构规则
- `META-*`：`meta` 规则
- `RUNTIME-*`：`runtime` 规则
- `LLM-*`：`llm_profiles` 规则
- `TOOL-*`：`tool_registry` 规则
- `SEC-*`：`security` 规则
- `POL-*`：`policies` 规则
- `ROLE-*`：`roles` 规则
- `STAGE-*`：`stages` 规则
- `TRANS-*`：`transitions` 规则
- `IMM-*`：只读产物与写权限规则
- `EX-*`：示例与最佳实践规则

## 3. 顶层结构规则

### 3.1 顶层字段

- `ROOT-001`：根节点必须是一个 YAML object，禁止是数组或标量。
- `ROOT-002`：顶层允许字段为：`version`、`meta`、`runtime`、`llm_profiles`、`tool_registry`、`security`、`policies`、`roles`、`stages`、`transitions`。
- `ROOT-003`：禁止出现未声明顶层字段，避免运行时与文档分叉。
- `ROOT-004`：`version`、`meta`、`llm_profiles`、`tool_registry`、`roles`、`stages`、`transitions` 为首版必填。
- `ROOT-005`：`runtime`、`security`、`policies` 允许部分缺省，但缺省部分只能由首版默认值补齐。

### 3.2 最小合法骨架

满足以下骨架，才可能成为 `001` 阶段最小可运行配置：

```yaml
version: "2.1"

meta:
  workflow_id: "example-workflow"
  name: "Example Workflow"

llm_profiles:
  default_cli:
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
    name: "Example Agent"
    type: "react"
    llm_profile: "default_cli"
    system_prompt: "你负责执行示例任务。"
    tools: ["file_read"]

stages:
  - id: "stage_a"
    name: "示例阶段"
    role: "agent_a"
    output:
      - type: "artifact"
        name: "example_output"

transitions:
  - from: "stage_a"
    to: "__END__"
```

## 4. `version` 与 `meta` 规则

### 4.1 `version`

- `META-001`：`version` 必须是字符串。
- `META-002`：`001` 阶段固定支持 `"2.1"`，其他版本号必须在运行时显式拒绝或提示未支持。

### 4.2 `meta`

- `META-003`：`meta.workflow_id` 必填，建议使用小写字母、数字、中划线、下划线组合。
- `META-004`：`meta.name` 必填，用于终端展示与报告标题。
- `META-005`：`meta.owner` 可选，但建议在团队协作场景填写。
- `META-006`：`workflow_id` 应全局稳定，不要把临时时间戳塞进 `workflow_id`。

## 5. `runtime` 规则

### 5.1 `runtime.workspace`

- `RUNTIME-001`：`runtime.workspace.root` 默认值为 `.octopus`。
- `RUNTIME-002`：`runtime.workspace.sessions_dir` 默认值为 `.octopus/sessions`。
- `RUNTIME-003`：`runtime.workspace.artifacts_dir` 默认值为 `.octopus/artifacts`。
- `RUNTIME-004`：`runtime.workspace.logs_dir` 默认值为 `.octopus/logs`。
- `RUNTIME-005`：工作目录类路径必须是项目内可写路径，不能落到 `/etc`、`/usr` 这类系统只读目录。

### 5.2 `runtime.tmux`

- `RUNTIME-006`：`runtime.tmux.enabled` 可选，未声明时按实现默认值处理。
- `RUNTIME-007`：`runtime.tmux.socket_name` 如声明，应允许 `{session_id}` 占位符。
- `RUNTIME-008`：`runtime.tmux.main_pane_ratio` 必须是正数，建议 `(0,1)` 区间。
- `RUNTIME-009`：`runtime.tmux.role_layout` 仅允许受支持值，例如 `adaptive_grid`、`tiled`。

### 5.3 `runtime.checkpoint`

- `RUNTIME-010`：`runtime.checkpoint.enabled` 默认 `true`。
- `RUNTIME-011`：`runtime.checkpoint.on_stage_boundary` 默认 `true`。
- `RUNTIME-012`：`runtime.checkpoint.every_n_events` 如声明，必须为正整数。

### 5.4 `runtime.scheduler`

- `RUNTIME-013`：`runtime.scheduler.max_parallel_roles` 如声明，必须为正整数。
- `RUNTIME-014`：`runtime.scheduler.dispatch_strategy` 仅允许受支持枚举值。

### 5.5 `runtime.role_runtime`

- `RUNTIME-015`：`runtime.role_runtime.trigger_mode` 仅允许受支持枚举值。
- `RUNTIME-016`：`runtime.role_runtime.idle_poll_seconds` 默认 `2`，必须为正整数。
- `RUNTIME-017`：`runtime.role_runtime.bootstrap_read_order` 如声明，必须是非空字符串数组。
- `RUNTIME-018`：`runtime.role_runtime.safe_interrupt_boundary` 必须是运行时可识别边界值。

### 5.6 `runtime.human_io` 与 `runtime.master_watch`

- `RUNTIME-019`：`runtime.human_io.main_agent_only` 建议固定为 `true`，首版不建议开放子 Agent 直接读取用户输入。
- `RUNTIME-020`：`runtime.human_io.conversation_file` 如声明，必须指向 Markdown 路径。
- `RUNTIME-021`：`runtime.master_watch.enabled` 默认建议为 `true`。
- `RUNTIME-022`：`runtime.master_watch.progress_file` 默认 `planner/global_progress.md`。
- `RUNTIME-023`：`runtime.master_watch.blocker_file` 默认 `planner/blockers.md`。
- `RUNTIME-024`：`runtime.master_watch.max_no_progress_rounds` 必须为正整数。

## 6. `llm_profiles` 规则

- `LLM-001`：`llm_profiles` 必须是 object，key 为 profile id。
- `LLM-002`：至少定义一个 profile。
- `LLM-003`：profile id 在当前 YAML 内必须唯一。
- `LLM-004`：每个 profile 必须声明 `provider` 与 `mode`。
- `LLM-005`：`mode=cli` 时，建议声明 `cli_path`。
- `LLM-006`：`max_tokens` 如声明，必须为正整数。
- `LLM-007`：`temperature` 如声明，必须是合法浮点数，建议在 `0` 到 `1` 之间。
- `LLM-008`：`roles[*].llm_profile` 必须引用这里已定义的 profile id。

## 7. `tool_registry` 规则

- `TOOL-001`：`tool_registry` 必须至少包含一个已注册工具。
- `TOOL-002`：工具 id 在所有 registry 域内必须唯一。
- `TOOL-003`：内置工具至少需要 `module` 与 `class`。
- `TOOL-004`：MCP 工具至少需要启动命令与参数信息。
- `TOOL-005`：角色中声明的 `tools` 必须都能在 `tool_registry` 中解析到。
- `TOOL-006`：如果某工具涉及高风险写入或命令执行，必须与安全策略联动校验。

## 8. `security` 规则

### 8.1 `security.shell`

- `SEC-001`：只要任一角色启用了 `shell_exec`，就必须声明 `security.shell`。
- `SEC-002`：`security.shell.allowlist_prefixes` 必须是字符串数组。
- `SEC-003`：`security.shell.denylist_keywords` 必须是字符串数组。
- `SEC-004`：首版建议 allowlist 优先、denylist 兜底；不能只给 denylist 不给 allowlist。

### 8.2 `security.path`

- `SEC-005`：`security.path.writable_roots` 必须是路径数组。
- `SEC-006`：`security.path.read_only_roots` 必须是路径数组。
- `SEC-007`：同一路径不能同时出现在 writable 和 read-only 中。
- `SEC-008`：角色约束与全局路径约束冲突时，以更严格者为准。

## 9. `policies` 规则

### 9.1 重试、超时、回路保护

- `POL-001`：`policies.retry.max_retry_per_stage` 默认 `2`，必须为非负整数。
- `POL-002`：`policies.retry.backoff_seconds` 默认 `[5, 20]`，必须是正整数数组。
- `POL-003`：`policies.timeout.stage_timeout_seconds` 默认 `1800`，必须为正整数。
- `POL-004`：`policies.timeout.role_heartbeat_timeout_seconds` 默认 `120`，必须为正整数。
- `POL-005`：`policies.loop_guard.max_rounds_per_task` 默认 `6`，必须为正整数。
- `POL-006`：`policies.loop_guard.min_quality_gain` 默认 `0.05`，必须大于 `0`。

### 9.2 人工接管、会话重置、产物与死锁

- `POL-007`：`policies.human_gate.on_high_risk` 如启用，需同时给出可解释阈值。
- `POL-008`：`policies.human_gate.high_risk_threshold` 必须在合理区间内，建议 `0` 到 `1`。
- `POL-009`：`policies.session_reset.enabled` 如启用，必须与阶段中的 `mode: session_reset` 规则一致。
- `POL-010`：`policies.artifact.keep_latest_versions` 必须为正整数。
- `POL-011`：`policies.immutable_artifacts.paths` 必须是路径模式数组。
- `POL-012`：`policies.immutable_artifacts.allow_writers` 必须是角色 id 数组。
- `POL-013`：`policies.deadlock_guard.max_no_progress_rounds` 必须为正整数。
- `POL-014`：`policies.deadlock_guard.blocked_statuses` 必须是非空数组。
- `POL-015`：`policies.deadlock_guard.on_trigger` 必须为受支持动作。

## 10. `roles` 规则

### 10.1 基础字段

- `ROLE-001`：`roles` 必须是非空数组。
- `ROLE-002`：每个角色必须声明 `id`、`name`、`type`、`llm_profile`。
- `ROLE-003`：`roles[*].id` 在全局必须唯一。
- `ROLE-004`：`roles[*].llm_profile` 必须引用已定义 profile。
- `ROLE-005`：`roles[*].type` 必须为运行时支持类型。

### 10.2 提示词与约束

- `ROLE-006`：`system_prompt` 建议必填，且必须能序列化为 Markdown 文本。
- `ROLE-007`：`reset_prompt` 在需要 `session_reset` 的角色上强烈建议填写。
- `ROLE-008`：`constraints.must_read_files` 如声明，必须是字符串路径数组。
- `ROLE-009`：`constraints.forbidden_writes` 如声明，必须是字符串路径模式数组。
- `ROLE-010`：首版禁止把角色约束写成无法解析的自由结构，约束字段必须可结构化读取。

### 10.3 工具与权限

- `ROLE-011`：`tools` 必须是已注册工具 id 数组。
- `ROLE-012`：审核型角色建议不要拥有会改变代码和测试基线的高风险写权限。
- `ROLE-013`：若角色被定位为只读审核角色，应通过 `constraints.forbidden_writes` 与产物只读规则双重限制。
- `ROLE-014`：负责 E2E 执行的角色不得写入 `immutable_artifacts` 中声明为只读的测试套件路径，除非该角色在 `allow_writers` 中。

## 11. `stages` 规则

### 11.1 基础字段

- `STAGE-001`：`stages` 必须是非空数组。
- `STAGE-002`：每个阶段必须声明 `id`、`name`、`role`。
- `STAGE-003`：`stages[*].id` 在全局必须唯一。
- `STAGE-004`：`stages[*].role` 必须引用已定义角色。

### 11.2 输入输出

- `STAGE-005`：`input` 与 `output` 如声明，必须为数组。
- `STAGE-006`：`input.type=artifact` 且使用 `ref` 时，必须引用前序阶段已产出的 artifact。
- `STAGE-007`：`output.type=artifact` 时，必须声明稳定 `name`。
- `STAGE-008`：同一阶段内输出 artifact 名称不得重复。

### 11.3 会话重置阶段

- `STAGE-009`：`mode: session_reset` 只能用于显式的重置阶段。
- `STAGE-010`：`mode: session_reset` 的阶段必须绑定已存在角色。
- `STAGE-011`：`mode: session_reset` 的阶段必须声明 `clear_cli_context`。
- `STAGE-012`：`mode: session_reset` 的阶段必须声明保留策略，例如保留已批准文档、审查意见或测试结果。

## 12. `transitions` 规则

- `TRANS-001`：`transitions` 必须是非空数组。
- `TRANS-002`：每条 transition 必须声明 `from`。
- `TRANS-003`：`from` 必须引用已定义阶段。
- `TRANS-004`：`to`、`on_true`、`on_false` 的目标必须引用已定义阶段或 `__END__`。
- `TRANS-005`：条件表达式必须使用运行时支持的 condition 类型。
- `TRANS-006`：允许业务回路，但必须受 `policies.loop_guard` 约束。
- `TRANS-007`：禁止出现永远无法到达或无法退出的无保护环路。

## 13. 只读产物与写权限规则

- `IMM-001`：`policies.immutable_artifacts.paths` 中的路径默认视为只读。
- `IMM-002`：只有 `allow_writers` 中声明的角色允许写入这些路径。
- `IMM-003`：如果某输入 artifact 被显式声明 `access: read_only`，后续消费阶段不得把同一路径作为输出目标。
- `IMM-004`：E2E 套件类产物一旦进入只读区，非授权修复角色只能读取，不能覆盖。
- `IMM-005`：角色约束中的 `forbidden_writes` 与全局 `immutable_artifacts` 同时命中时，无条件阻断。

## 14. 环境变量与 flags 覆盖规则

本文档主要描述 YAML 本体，但 `config 001` 同时允许 env 与 flags 覆盖部分字段，因此必须遵守以下补充规则：

- `EX-001`：优先级固定为 `defaults < yaml < env < flags`。
- `EX-002`：环境变量前缀统一为 `OPENOCTOPUS_`。
- `EX-003`：环境变量层级分隔符统一使用双下划线 `__`。
- `EX-004`：环境变量和 flags 只建议覆盖标量型字段与运行环境相关字段，不要重写整个 `roles`、`stages` 结构。
- `EX-005`：如果 env 或 flags 覆盖后产生非法值，仍然必须被静态校验阻断。

## 15. 常见坑点清单

- `EX-006`：写了 `roles[*].tools`，但忘了在 `tool_registry` 注册同名工具。
- `EX-007`：阶段引用了不存在的角色或 artifact。
- `EX-008`：引入业务回路，但没有配置 `loop_guard`，导致潜在死循环。
- `EX-009`：给审核角色过宽的写权限，导致“审核角色直接改产物”破坏职责边界。
- `EX-010`：配置了 `shell_exec`，却没有 `security.shell` allowlist。
- `EX-011`：`immutable_artifacts` 和角色写路径冲突，导致 E2E 套件被误覆盖。
- `EX-012`：`session_reset` 只声明了模式，没有声明保留策略，导致重置后上下文丢失。
- `EX-013`：`master_watch`、`deadlock_guard` 阈值过小，造成系统频繁误判阻塞。

## 16. 可复制示例

### 16.1 A/B/C + Master 自动化模板

```yaml
version: "2.1"

meta:
  workflow_id: "feature-dev-loop"
  name: "功能开发闭环流程"
  owner: "team-openoctopus"

runtime:
  workspace:
    root: ".octopus"
  role_runtime:
    trigger_mode: "fs_event_then_poll"
    idle_poll_seconds: 2
  human_io:
    main_agent_only: true
    conversation_file: "planner/human_messages.md"
  master_watch:
    enabled: true
    progress_file: "planner/global_progress.md"
    blocker_file: "planner/blockers.md"
    max_no_progress_rounds: 3

llm_profiles:
  claude_code_cli:
    provider: "claude_code"
    mode: "cli"
    cli_path: "claude"
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read: { module: "openoctopus.tools.file", class: "FileReadTool" }
    file_write: { module: "openoctopus.tools.file", class: "FileWriteTool" }
    shell_exec: { module: "openoctopus.tools.shell", class: "ShellTool" }
    git_operation: { module: "openoctopus.tools.git", class: "GitTool" }

security:
  shell:
    allowlist_prefixes: ["git status", "git diff", "pytest"]
    denylist_keywords: ["rm -rf /", "shutdown", "reboot"]
  path:
    writable_roots: ["./", ".octopus/"]
    read_only_roots: ["/etc", "/usr", "/var"]

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
  immutable_artifacts:
    paths:
      - "artifacts/{session_id}/tests/e2e/**"
    allow_writers: ["agent_c"]
  deadlock_guard:
    enabled: true
    max_no_progress_rounds: 3
    blocked_statuses: ["BLOCKED", "INTERRUPTED_PENDING_ACK"]
    on_trigger: "interrupt_all_and_wait_human"

roles:
  - id: "agent_a"
    name: "方案与实现 Agent"
    type: "react"
    llm_profile: "claude_code_cli"
    system_prompt: "你先负责方案设计，设计通过后再实现代码。"
    reset_prompt: "新会话只保留批准文档、审查意见和测试结果。"
    constraints:
      must_read_files: ["roles/{role_id}/context.md", "roles/{role_id}/inbox.md"]
      forbidden_writes: ["artifacts/{session_id}/tests/e2e/**"]
    tools: ["file_read", "file_write", "shell_exec", "git_operation"]

  - id: "agent_b"
    name: "审核 Agent"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责审核方案与代码文档一致性。"
    constraints:
      forbidden_writes: ["src/**", "artifacts/{session_id}/tests/e2e/**"]
    tools: ["file_read", "file_write", "git_operation"]

  - id: "agent_c"
    name: "E2E Agent"
    type: "react"
    llm_profile: "claude_code_cli"
    system_prompt: "你负责产出和维护 E2E 用例。"
    tools: ["file_read", "file_write", "shell_exec"]

stages:
  - id: "design_solution"
    name: "方案设计"
    role: "agent_a"
    output:
      - type: "artifact"
        name: "solution_doc"

  - id: "design_review"
    name: "方案审核"
    role: "agent_b"
    input:
      - type: "artifact"
        ref: "solution_doc"
    output:
      - type: "artifact"
        name: "design_review_report"

  - id: "reset_agent_a_session"
    name: "重置实现会话"
    role: "agent_a"
    mode: "session_reset"
    clear_cli_context: true
    preserve:
      artifacts: ["solution_doc", "design_review_report"]

  - id: "e2e_authoring"
    name: "E2E 编写"
    role: "agent_c"
    input:
      - type: "artifact"
        ref: "solution_doc"
    output:
      - type: "artifact"
        name: "e2e_suite"

transitions:
  - from: "design_solution"
    to: "design_review"
  - from: "design_review"
    condition:
      type: "expression"
      expr: "artifacts.design_review_report.approved == true"
    on_true: "reset_agent_a_session"
    on_false: "design_solution"
  - from: "reset_agent_a_session"
    to: "e2e_authoring"
  - from: "e2e_authoring"
    to: "__END__"
```

### 16.2 单 Agent 极简模板

```yaml
version: "2.1"

meta:
  workflow_id: "single-agent-demo"
  name: "单 Agent 极简流程"

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

tool_registry:
  builtin:
    file_read: { module: "openoctopus.tools.file", class: "FileReadTool" }
    file_write: { module: "openoctopus.tools.file", class: "FileWriteTool" }

roles:
  - id: "agent_a"
    name: "单 Agent"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你负责完成需求分析并输出文档。"
    tools: ["file_read", "file_write"]

stages:
  - id: "analyze"
    name: "需求分析"
    role: "agent_a"
    output:
      - type: "artifact"
        name: "analysis_report"

transitions:
  - from: "analyze"
    to: "__END__"
```

### 16.3 带人工打断模板

```yaml
version: "2.1"

meta:
  workflow_id: "human-gate-demo"
  name: "带人工介入流程"

runtime:
  human_io:
    main_agent_only: true
    conversation_file: "planner/human_messages.md"

llm_profiles:
  claude_code_cli:
    provider: "claude_code"
    mode: "cli"
    cli_path: "claude"

tool_registry:
  builtin:
    file_read: { module: "openoctopus.tools.file", class: "FileReadTool" }
    file_write: { module: "openoctopus.tools.file", class: "FileWriteTool" }

policies:
  human_gate:
    on_high_risk: true
    high_risk_threshold: 0.8
    write_manual_input_to_markdown: true
    main_agent_ack_required: true

roles:
  - id: "agent_a"
    name: "主执行 Agent"
    type: "react"
    llm_profile: "claude_code_cli"
    system_prompt: "你负责输出方案，但高风险时需要等待人工确认。"
    tools: ["file_read", "file_write"]

stages:
  - id: "draft_solution"
    name: "方案草拟"
    role: "agent_a"
    output:
      - type: "artifact"
        name: "solution_doc"

transitions:
  - from: "draft_solution"
    to: "__END__"
```

## 17. 编写建议

- 优先保持 YAML 显式，不要过度依赖隐式默认值。
- 引用关系优先写稳定 id，不要用临时命名。
- 一份配置里，角色职责边界要清楚：谁设计、谁审核、谁测、谁修复，不要混成一个万能角色。
- 当需要只读约束时，优先同时使用全局只读产物规则与角色局部 `forbidden_writes`。
- 当需要 AI 自动生成配置时，先套最接近的示例，再按规则增量修改。
