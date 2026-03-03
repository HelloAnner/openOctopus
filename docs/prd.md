
## 1 产品定位

OpenOctopus 是一个**命令行研发流程编排工具**，通过 YAML 配置将文档设计、代码实现、代码审查、测试验证等研发环节串联成可自动执行的工作流，解决研发人员在多个 AI 会话间手动切换的痛点。

### 1.1 核心价值

| 价值        | 说明                 |
| --------- | ------------------ |
| **流程自动化** | 将"人肉编排"转变为"自动驱动"   |
| **状态可追溯** | 执行结果、中间产物、决策依据全记录  |
| **人机协作**  | 支持自动执行与人工介入的灵活切换   |
| **可复用性**  | YAML 配置驱动，团队共享最佳实践 |

### 1.2 目标用户

- 使用 Claude Code / Codex 进行日常开发的研发人员
- 需要规范研发流程的技术团队

## 2 核心理念

OpenOctopus 的核心不是“多 Agent 本身”，而是“**基于文件系统的可追溯协作总线**”。

### 2.1 单一事实源：Markdown 文件

- 会话状态、调度计划、角色结论、执行时间线全部落在 `.octopus/sessions/{session_id}/` 下的 `.md` 文件。
- 不依赖数据库作为主状态源；内存态只是运行缓存，重启后可由文件完全恢复。
- 每个关键步骤都必须有对应文件记录，保证复盘、审计、回放可执行。

### 2.2 主 Agent 决策原则

- 主 Agent 只基于文件做决策，不基于临时上下文做隐式判断。
- 主决策输入固定为：
    - 各子 Agent 的 `roles/{role_id}/conclusion.md`
    - 主控排程 `planner/master_schedule.md`
- 主 Agent 负责任务拆分、路由、回流和终态判定；子 Agent 只负责按角色执行并产出结论文件。

### 2.3 协作机制：文件即协议

- `inbox.md` / `outbox.md`：任务分发与回执。
- `lock.md`：并发调度锁，避免多写冲突。
- `events.md`：先写事件，再推进状态（WAL 思想）。
- `heartbeat.md`：角色存活探测与超时处理依据。

### 2.4 设计目标

1. **稳定协作**：主 Agent 与多个子 Agent 并行时不乱序、不覆盖。
2. **可中断恢复**：任意阶段可打断、可重排、可续跑。
3. **全过程可追溯**：每次拆分、每次回合、每次决策都有文件证据链。
4. **实现简洁**：优先文件系统原语，避免过度工程化。

---

## 3 核心概念

### 3.1 术语定义

| 术语 | 定义 |
|------|------|
| **Workflow** | 完整的研发流程定义，由多个 Stage 组成 |
| **Stage** | 流程阶段，对应具体研发环节（如代码实现、审查） |
| **Role** | 执行 Stage 的角色，包含系统提示词、工具集、LLM 配置 |
| **Context** | 跨 Stage 传递的上下文，包含中间产物、决策记录 |
| **Transition** | Stage 间流转规则，支持条件分支和循环 |
| **Artifact** | 流程产物（文档、代码、报告等） |
| **Session** | 一次具体的流程执行实例 |

### 3.2 系统架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           OpenOctopus 架构                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  表现层: CLI (Typer) │ TMUX 可视化 │ Web 监控 (Streamlit)        │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  应用服务层: 流程调度 │ 状态管理 │ 事件总线 │ 人机协作中心         │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  LangGraph 核心: StateGraph │ 条件边 │ 中断恢复 │ Checkpointer   │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  领域层: Claude/Codex/本地 LLM 执行器 │ 工具链 │ 产物管理          │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  基础设施层: 文件存储(主) │ 内存缓存(辅) │ SQLite 索引(可选) │ TMUX 管理器 │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 4 CLI 命令集

```bash
# 工作流生命周期
openoctopus init --template <name>          # 初始化工作流配置
openoctopus run [--config ./octopus.yaml]   # 执行工作流
openoctopus status [--session <id>]         # 查看状态
openoctopus resume --session <id>           # 恢复执行
openoctopus stop --session <id>             # 停止工作流
openoctopus validate --config ./octopus.yaml # 配置静态校验

# 辅助命令
openoctopus report --session <id>           # 生成执行报告
openoctopus switch --role <role_id>         # 切换 TMUX 角色会话
openoctopus debug --session <id>            # 调试模式
```

---

## 5 YAML 配置规范

### 5.1 完整配置示例

```yaml
version: "2.1"

meta:
  workflow_id: "feature-dev-loop"
  name: "功能开发闭环流程"
  owner: "team-openoctopus"

runtime:
  workspace:
    root: ".octopus"
    sessions_dir: ".octopus/sessions"
    artifacts_dir: ".octopus/artifacts"
    logs_dir: ".octopus/logs"
  tmux:
    enabled: true
    socket_name: "octopus-{session_id}"
    main_pane_ratio: 0.5
    role_layout: "adaptive_grid"   # adaptive_grid | tiled
  checkpoint:
    enabled: true
    on_stage_boundary: true
    every_n_events: 20
  recovery:
    replay_order: ["bus/events.md", "state/checkpoints", "planner/master_schedule.md"]
    strict_mode: true              # 丢关键文件直接失败，避免脏恢复
  scheduler:
    max_parallel_roles: 4
    dispatch_strategy: "priority_then_dependency"

llm_profiles:
  claude_code_cli:
    provider: "claude_code"
    mode: "cli"
    cli_path: "claude"
    max_tokens: 4096
    temperature: 0.2
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"
    max_tokens: 4096
    temperature: 0.1

tool_registry:
  builtin:
    file_read: { module: "openoctopus.tools.file", class: "FileReadTool" }
    file_write: { module: "openoctopus.tools.file", class: "FileWriteTool" }
    shell_exec: { module: "openoctopus.tools.shell", class: "ShellTool" }
    git_operation: { module: "openoctopus.tools.git", class: "GitTool" }
    web_search: { module: "openoctopus.tools.web", class: "WebSearchTool" }
  mcp:
    slack:
      command: "npx"
      args: ["-y", "@modelcontextprotocol/server-slack"]
      env:
        SLACK_BOT_TOKEN: "${SLACK_BOT_TOKEN}"

security:
  shell:
    allowlist_prefixes: ["git status", "git diff", "pytest", "npm test", "pnpm test"]
    denylist_keywords: ["rm -rf /", "shutdown", "reboot"]
  path:
    writable_roots: ["./", ".octopus/"]
    read_only_roots: ["/etc", "/var", "/usr"]

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
  artifact:
    hash_algo: "sha256"
    keep_latest_versions: 5

roles:
  - id: "req_split_agent"
    name: "需求拆分"
    type: "plan_exec"
    llm_profile: "claude_code_cli"
    system_prompt: |
      你负责需求拆分、范围澄清和任务依赖建模。
    tools: ["file_read", "file_write", "web_search"]

  - id: "impl_agent"
    name: "实现工程师"
    type: "simple"
    llm_profile: "claude_code_cli"
    system_prompt: |
      你负责编码、修复和变更说明输出。
    tools: ["file_read", "file_write", "shell_exec", "git_operation"]

  - id: "test_e2e_agent"
    name: "测试工程师"
    type: "react"
    llm_profile: "claude_code_cli"
    react_config:
      max_iterations: 8
    system_prompt: |
      你负责执行测试、提炼失败证据并输出可复现步骤。
    tools: ["file_read", "file_write", "shell_exec"]

  - id: "review_agent"
    name: "代码审查员"
    type: "react"
    llm_profile: "codex_cli"
    react_config:
      max_iterations: 10
    system_prompt: |
      你负责质量评审、风险识别和准入建议。
    tools: ["file_read", "file_write", "git_operation"]

stages:
  - id: "requirements_analysis"
    name: "需求拆分"
    role: "req_split_agent"
    input:
      - type: "requirement_file"
        path: "./docs/feature.md"
      - type: "user_prompt"
        optional: true
    output:
      - type: "artifact"
        name: "task_tree"

  - id: "code_implementation"
    name: "代码实现"
    role: "impl_agent"
    input:
      - type: "artifact"
        ref: "task_tree"
    output:
      - type: "artifact"
        name: "implemented_code"

  - id: "test_e2e"
    name: "测试验证"
    role: "test_e2e_agent"
    input:
      - type: "artifact"
        ref: "implemented_code"
    output:
      - type: "artifact"
        name: "test_report"

  - id: "code_review"
    name: "代码审查"
    role: "review_agent"
    input:
      - type: "artifact"
        ref: "implemented_code"
      - type: "artifact"
        ref: "test_report"
    output:
      - type: "artifact"
        name: "review_report"

transitions:
  - from: "requirements_analysis"
    to: "code_implementation"

  - from: "code_implementation"
    to: "test_e2e"

  - from: "test_e2e"
    condition:
      type: "expression"
      expr: "artifacts.test_report.passed == true"
    on_true: "code_review"
    on_false: "code_implementation"

  - from: "code_review"
    condition:
      type: "aggregate"
      mode: "all"
      rules:
        - "artifacts.review_report.ready_for_merge == true"
        - "artifacts.review_report.blocking_issues == 0"
    on_true: "__END__"
    on_false: "code_implementation"
```

### 5.2 数据目录结构（Markdown First）

```
project-root/
├── octopus.yaml                       # 核心配置文件
├── src/                               # 源代码
└── .octopus/                          # OpenOctopus 工作目录
    ├── sessions/{session_id}/         # 会话数据
    │   ├── session.state.md           # 全局状态快照（当前）
    │   ├── timeline.md                # 执行时间线（追加写）
    │   ├── metadata.md                # 会话元数据
    │   ├── planner/
    │   │   ├── master_schedule.md
    │   │   ├── task_board.md
    │   │   └── decision_log.md
    │   ├── bus/
    │   │   ├── events.md              # 事件总线日志（追加写）
    │   │   ├── offsets.md
    │   │   ├── lock.md
    │   │   └── interrupts.md
    │   ├── state/
    │   │   └── checkpoints/
    │   │       ├── step-0001.md
    │   │       └── step-0002.md
    │   └── roles/
    │       └── {role_id}/
    │           ├── state.md
    │           ├── inbox.md
    │           ├── outbox.md
    │           ├── conclusion.md
    │           └── turns/
    ├── artifacts/{session_id}/        # 产物存储
    │   ├── tech_spec.v1.md
    │   ├── tech_spec.v2.md            # 自动版本控制
    │   ├── src/
    │   └── manifest.md
    ├── logs/{session_id}/             # 日志文件
    ├── cache/                         # 缓存数据
    └── .venv/                         # Python 虚拟环境
```

### 5.3 YAML 校验与默认策略（必须）

1. **结构校验**：`roles.id`、`stages.id` 全局唯一；`stage.role` 必须引用已定义角色。
2. **引用校验**：`input.ref` 必须指向已产出的 artifact，禁止悬空引用。
3. **流转校验**：`transitions.from` 必须存在，`on_true/on_false/to` 目标必须存在或为 `__END__`。
4. **环路校验**：允许业务回路，但必须受 `policies.loop_guard` 限制，避免死循环。
5. **工具校验**：角色声明的 `tools` 必须能在 `tool_registry` 中解析。
6. **安全校验**：启用 `shell_exec` 的角色必须存在 `security.shell` 配置。
7. **默认值策略**：未配置 `retry/timeout/checkpoint` 时使用系统默认保守值，并记录到启动日志。
8. **阻断规则**：校验失败时禁止 `run`；`validate` 仅输出错误并返回非 0 退出码。

---

## 6 TMUX 编排与运行视图

### 6.1 指定 YAML 后的启动时序

最终运行入口统一为：

```bash
openoctopus run --config ./octopus.yaml --requirement ./docs/feature.md
```

系统固定按以下步骤执行：

```text
[读取 YAML 配置]
        |
        v
[校验 roles/stages/transitions]
        |
        v
[创建 session_id 与工作目录]
        |
        v
[启动 tmux server: octopus-{session_id}]
        |
        v
[构建主控+角色窗格布局]
        |
        v
[加载 LangGraph 总线图]
        |
        v
[主 Agent 读取需求并拆分任务]
        |
        v
[按角色分发子任务并执行]
        |
        v
[汇总结果/打断处理/归档]
```

### 6.2 固定布局：左主右子（自动横竖拆分）

布局规则：

1. 整体窗口先做一次水平分割，左 50% 固定给主 Agent（主会话）。
2. 右 50% 作为角色区，根据 `roles` 数量自动拆分多个 pane。
3. 右侧拆分策略采用“近似网格”：
    - 角色数 1：右侧单 pane。
    - 角色数 2：上下二分（垂直拆分）。
    - 角色数 3-4：先上下，再左右，形成 2x2。
    - 角色数 >4：继续按 `tiled` 扩展，保持可读性。

示意（4 个角色）：

```
┌──────────────────────────────┬──────────────────────────────┐
│            主 Agent          │          角色A 对话          │
│ 需求拆分 / 调度 / 决策 /总线 │──────────────────────────────│
│                              │          角色B 对话          │
│                              ├──────────────────────────────┤
│                              │          角色C 对话          │
│                              │──────────────────────────────│
│                              │          角色D 对话          │
└──────────────────────────────┴──────────────────────────────┘
```

### 6.3 主 Agent 与子 Agent 的 Pane 责任

| 区域 | 责任 | 输入 | 输出 |
|------|------|------|------|
| 左侧主 Pane | 全局规划、任务拆分、调度、结果验收、回路控制 | 用户需求、YAML、各角色 `conclusion.md` | `master_schedule.md`、任务分发、阶段判定 |
| 右侧角色 Pane | 按角色定位执行子任务（实现/测试/e2e/审查等） | 主 Agent 指令、上下文快照 | `conclusion.md`、回合产物、状态更新、异常/打断事件 |

### 6.4 人工打断与动态更新

系统支持在执行中对任意子 Agent 发起打断或更新：

- `openoctopus interrupt --session <id> --role <role_id> --reason "..."`  
  立即中断该角色当前回合，进入 `INTERRUPTED`。
- `openoctopus inject --session <id> --role <role_id> --input ./patch.md`  
  向指定角色注入补充约束或修正需求。
- `openoctopus reroute --session <id> --from <stage> --to <stage>`  
  主 Agent 重新编排后续流转。

所有打断都会写入事件总线并落盘，保证可回放和可审计。

### 6.5 常用快捷键

| 操作 | 快捷键 | 说明 |
|------|--------|------|
| 切回主 Pane | `C-b o` | 快速回到左侧主控窗格 |
| 在右侧角色间切换 | `C-b ;` | 循环切换角色窗格 |
| 角色 pane 放大 | `C-b z` | 聚焦单个角色细看 |
| 重新平铺右侧 | `C-b M-5` | 强制重排角色区布局 |

---

## 7 LangGraph 总线编排引擎

### 7.1 总线化 StateGraph 实现图

```text
[START]
   |
   v
[LoadConfig] -> [InitSessionState] -> [InitTmuxLayout] -> [MainAgent.Plan]
                                                              |
                                                              v
                                                     [TaskBus.Dispatch]
                                                              |
                                                              v
                                                 [RoleAgent.Execute 并行子图]
                                                      |                |
                                                      |                +--> [InterruptHandler]
                                                      |                         |
                                                      +--> [TaskBus.Collect] <--+
                                                               |
                                                               v
                                                       [MainAgent.Evaluate]
                                                          |    |        |
                                                          |    |        +--> [HumanGate.Wait] -> [审批结果] --继续--> [MainAgent.Plan]
                                                          |    |                                         \
                                                          |    |                                          --终止--> [END]
                                                          |    |
                                                          |    +--> [继续拆分] -> [MainAgent.Plan]
                                                          |
                                                          +--> [完成] -> [END]
```

该图对应“主 Agent + 事件总线 + 多角色子图”的统一模型：

- 主 Agent 只做编排，不做具体实现细节。
- 子 Agent 只执行角色职责，通过总线回传结构化结果。
- 是否继续、回滚、重分配，统一由主 Agent 判定。

### 7.2 主图与子图关系（LangGraph 分层）

```text
Main Coordinator Graph:
[需求解析] -> [任务拆分] -> [任务分配] -> [结果验收] -> [通过?]
                                                      |        |
                                                      |否      |是
                                                      v        v
                                                   [任务拆分] [归档结束]

Role Worker Subgraph:
[接收任务] -> [执行回合] -> [写 turns 与 conclusion.md] -> [回执总线]

Graph Relation:
[任务分配] ---> [接收任务]
[回执总线] ---> [结果验收]
```

### 7.3 状态模型（文件系统 + Markdown）

系统不依赖数据库做主状态，统一使用 `.md` 文件表达状态与协作协议。

**SessionContext（全局）字段**

- `session_id`: 会话标识
- `requirement_snapshot`: 当前需求快照版本
- `task_board`: 主 Agent 任务看板
- `role_registry`: 角色能力映射（来自 YAML）
- `routing_table`: 任务与角色路由
- `interrupt_queue`: 打断队列
- `artifact_index`: 全局产物索引
- `decision_log`: 主 Agent 决策链

**RoleState（每角色）字段**

- `role_id` / `pane_id`: 角色与 tmux 窗格绑定
- `current_task_id`: 当前任务
- `turn`: 当前执行轮次
- `last_output_ref`: 最近回执产物
- `status`: IDLE | RUNNING | BLOCKED | INTERRUPTED | DONE

**字段落盘文件约定**

| 状态对象 | 文件路径 | 写入策略 |
|------|------|------|
| SessionContext 当前态 | `.octopus/sessions/{id}/session.state.md` | 覆盖写（原子替换） |
| 主控排程 | `.octopus/sessions/{id}/planner/master_schedule.md` | 局部更新（原子替换） |
| 主任务看板 | `.octopus/sessions/{id}/planner/task_board.md` | 局部更新（原子替换） |
| 决策链 | `.octopus/sessions/{id}/planner/decision_log.md` | 追加写 |
| 角色状态 | `.octopus/sessions/{id}/roles/{role}/state.md` | 覆盖写（原子替换） |
| 角色结论 | `.octopus/sessions/{id}/roles/{role}/conclusion.md` | 覆盖写（保留版本快照） |
| 角色回合输入 | `.octopus/sessions/{id}/roles/{role}/turns/000N-input.md` | 新文件 |
| 角色回合输出 | `.octopus/sessions/{id}/roles/{role}/turns/000N-output.md` | 新文件 |

### 7.4 循环与回路控制（主 Agent 统一收口）

典型回路：需求拆分 → 实现 → 测试/e2e → 审查 → 决策。  
主 Agent 对每一轮做三类判定：

1. **通过**：推进到下一阶段或结束。
2. **修复**：回流到实现角色，附带差异化修复指令。
3. **升级**：进入人工审批或重新拆分策略。

控制项：

- `max_rounds_per_task`: 单任务最大回合数
- `max_retry_per_role`: 角色最大重试次数
- `improvement_threshold`: 增量改善阈值（防止无效循环）
- `timeout_policy`: 超时后的自动策略（重试/暂停/终止）

### 7.5 条件流转与聚合策略

**流转条件类型：**

- `expression`: 简单布尔表达式（受限 DSL，不执行任意代码）
- `aggregate`: 多条件组合（all/any/majority）
- `role_aggregate`: 多角色结果聚合（all/any/majority）

**聚合结果字段建议：**

- `quality_score`
- `risk_level`
- `blocking_issues`
- `ready_for_merge`

---

## 8 事件总线与信息流转

### 8.1 核心设计原则

1. **所有状态变更必须通过事件总线** - 禁止直接修改状态
2. **事件持久化先行（WAL 模式）** - 先写 Markdown 日志，再更新内存
3. **完整的信息血缘追踪** - 产物来源、决策依据全记录
4. **断点可恢复性** - 任意时刻可重建完整状态

### 8.2 事件类型体系

```
Event Types
│
├── System Events
│   ├── WORKFLOW_INITIALIZED / STARTED / COMPLETED / FAILED
│   ├── WORKFLOW_PAUSED / RESUMED / CANCELLED
│
├── Stage Events
│   ├── STAGE_SCHEDULED / STARTED / COMPLETED / FAILED
│   ├── STAGE_RETRYING / SKIPPED / WAITING_HUMAN
│
├── Role Events
│   ├── ROLE_ASSIGNED / PROMPT_SENT / RESPONSE_RECEIVED
│   ├── ROLE_STREAM_STARTED / STREAM_CHUNK / STREAM_COMPLETED
│   ├── ROLE_TOOL_CALLED / TOOL_COMPLETED
│
├── Artifact Events
│   ├── ARTIFACT_CREATED / UPDATED / VERSIONED / PERSISTED
│
└── User Events
    ├── USER_INPUT_RECEIVED / APPROVAL_GIVEN / REJECTION_GIVEN
```

### 8.3 Markdown 多层持久化架构

```
Layer 1: Event WAL (Markdown)
├── 位置: .octopus/sessions/{session_id}/bus/events.md
├── 格式: 追加写 Markdown 条目（时间戳 + 事件ID + 类型 + 载荷 + hash）
├── 策略: 同步写入，写入成功后才允许状态推进
└── 用途: 崩溃恢复、事件回放

Layer 2: State Snapshots (Markdown)
├── 位置: .octopus/sessions/{session_id}/state/checkpoints/*.md
├── 格式: 每步一个快照文件（step-0001.md）
├── 策略: 阶段边界和回合结束触发
└── 用途: 快速恢复、定位问题

Layer 3: Role Runtime Logs (Markdown)
├── 位置: .octopus/sessions/{session_id}/roles/{role_id}/
├── 格式: state.md / turns/*.md / events.md
└── 用途: 单角色执行历史、回合审计

Layer 4: Audit & Archive (Markdown)
├── 位置: .octopus/sessions/{session_id}/audit/*.md
├── 格式: timeline.md / lineage.md / summary.md
└── 用途: 长期复盘、团队共享
```

事件条目建议格式（示例）：

```markdown
### event-000123
- ts: 2026-03-03T10:15:21Z
- event_type: ROLE_RESPONSE_RECEIVED
- session_id: sess_abc123
- role_id: impl_agent
- correlation_id: task-42-turn-03
- payload_ref: roles/impl_agent/turns/0003-output.md
- hash: sha256:8ef2...
```

### 8.4 步骤级存储机制（全过程 Markdown 落盘）

为保证主 Agent 与多个子 Agent 协作稳定，执行信息和状态全部写入 Markdown：

```
.octopus/sessions/{session_id}/
├── session.state.md                    # 全局当前状态（主 Agent 读写）
├── timeline.md                         # 全局时间线（追加写）
├── planner/
│   ├── requirement.snapshot.md         # 当前需求快照
│   ├── master_schedule.md              # 主 Agent 统筹排程（唯一调度真相）
│   ├── task_board.md                   # 任务看板：Todo/Doing/Done/Blocked
│   ├── task_graph.mmd                  # 任务依赖图
│   ├── dispatch_log.md                 # 分发日志
│   └── decision_log.md                 # 主 Agent 决策日志
├── bus/
│   ├── events.md                       # 总线事件日志（唯一事实源）
│   ├── interrupts.md                   # 打断与注入记录
│   ├── offsets.md                      # 子 Agent 消费位点
│   └── lock.md                         # 调度锁与持有者
├── roles/
│   ├── req_split_agent/
│   │   ├── state.md                    # 当前角色状态
│   │   ├── inbox.md                    # 待处理任务（主 Agent 投递）
│   │   ├── outbox.md                   # 结果回执（子 Agent 回传）
│   │   ├── conclusion.md               # 角色最新结论（主 Agent 直接读取）
│   │   ├── heartbeat.md                # 心跳时间与存活标记
│   │   ├── events.md                   # 角色事件日志
│   │   └── turns/
│   │       ├── 0001-input.md
│   │       ├── 0001-output.md
│   │       └── ...
│   ├── impl_agent/
│   ├── test_e2e_agent/
│   └── review_agent/
├── artifacts/
│   ├── index.md                        # 产物索引
│   └── {task_id}/
│       ├── result.v1.md
│       ├── result.v2.md
│       └── diff.v1-v2.md
└── audit/
    ├── lineage.md                      # 任务-角色-产物血缘
    ├── replay.md                       # 恢复与回放说明
    └── final-report.md                 # 最终报告
```

写入原则：

1. 先写 `bus/events.md`，再写对应 `state.md`（WAL 先行）。
2. 主 Agent 与子 Agent 通过 `inbox.md/outbox.md` 解耦，不共享内存。
3. 每一轮必须生成 `turns/NNNN-input.md` 与 `turns/NNNN-output.md`。
4. 主 Agent 决策只读取各角色 `conclusion.md` 与 `planner/master_schedule.md`。
5. 事件类文件（`events.md`/`decision_log.md`）采用追加写；当前态文件（`state.md`/`session.state.md`）采用原子替换写。
6. 所有覆盖写文件在落盘前先写 `*.tmp`，再 `rename` 替换，避免半写状态。

### 8.5 主从协作文件协议（避免并发冲突）

为保证多 Agent 并行时不乱序、不覆盖，约定如下：

1. **锁协议**：主 Agent 写 `bus/lock.md`（`holder` + `lease_token` + `lease_version` + `expire_at`），分发结束后释放。
2. **领取协议**：子 Agent 从 `inbox.md` 领取任务后写 `state.md` 为 `RUNNING`。
3. **结论协议**：子 Agent 每轮必须更新 `conclusion.md`，主 Agent 仅基于该文件做阶段判定。
4. **回执协议**：子 Agent 完成后写 `outbox.md`，由主 Agent 统一消费并更新 `master_schedule.md` 与 `task_board.md`。
5. **心跳协议**：子 Agent 定期刷新 `heartbeat.md`；超时由主 Agent 触发打断或重分配。
6. **冲突协议**：写入前校验 `lease_version`，版本不一致则拒绝写入并重试。
7. **恢复协议**：崩溃后按 `events.md -> checkpoints/*.md -> master_schedule.md` 顺序重建状态。

---

## 9 核心功能详细设计

### 9.1 会话生命周期

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│INITIAL  │───→│ RUNNING │───→│ WAITING │───→│ RUNNING │───→│COMPLETED│
│ 初始化  │    │ 运行中  │    │ 等待人工 │    │ 运行中  │    │ 已完成  │
└─────────┘    └────┬────┘    └────┬────┘    └─────────┘    └─────────┘
                    │              │
                    ▼              │ 用户输入
               ┌─────────┐         │
               │ FAILED  │◀────────┘
               │ 失败    │
               └─────────┘
```

**持久化时机：**
- 阶段开始/完成：保存 checkpoint
- 人工介入点：保存等待状态
- 定时触发：防止意外丢失
- 优雅停止：标记暂停状态

### 9.2 人机协作系统

| 场景 | 触发条件 | 用户操作 | 系统响应 |
|------|----------|----------|----------|
| 方案确认 | 技术方案完成 | 查看/批准/驳回 | 通过则继续，驳回则重设计 |
| 代码审查 | 审查发现问题 | 查看/修复确认 | 确认后标记修复完成 |
| 测试失败 | 测试不通过 | 查看日志/决策 | 重试/跳过/中止 |
| 超时等待 | 阶段执行超时 | 确认继续/停止 | 根据选择执行 |

### 9.3 产物版本管理

**版本控制机制：**
- 自动检测同名产物，递增版本号
- 记录版本间差异（diff）
- 维护版本血缘关系（previous_version 引用）
- 产物 ID 格式：`{session_id}:{name}:v{version}`

**产物元数据：**
- 创建时间、创建者（stage/role）
- 内容指纹（hash）
- 版本差异对比
- 引用关系图谱

### 9.4 错误处理与恢复

**错误分类：**
- **ConfigError**: YAML 语法错误、必填项缺失、类型不匹配
- **ExecutionError**: LLM 调用失败、工具执行失败、超时
- **TransitionError**: 条件表达式错误、目标阶段不存在
- **SystemError**: 存储错误、网络错误、权限错误

**处理策略：**
- `retry`: 延迟后重试
- `skip`: 跳过当前阶段
- `pause`: 暂停等待人工
- `abort`: 中止工作流

### 9.5 主 Agent 实现图（任务拆分总线）

```text
[读取主需求与 YAML 角色定位]
              |
              v
       [构建任务树 TaskTree]
              |
              v
      [按角色能力拆分子任务]
              |
              v
      [发布任务到总线 TaskBus]
              |
              v
         [子 Agent 并行执行]
              |
              v
      [各角色写 conclusion.md]
              |
              v
[主 Agent 读取 conclusion.md + master_schedule.md]
              |
              v
       [主 Agent 质量门禁评估]
              |
              v
         [是否达成目标?]
            |        |
           是        否
            |        v
            |   [问题类型?]
            |    |    |    |
            |   修复  变更  阻塞
            |    |    |    |
            |    v    v    v
            | [生成修复任务] [更新需求快照并重拆分] [触发人工介入]
            |      |               |                  |
            +------|---------------+------------------+
                   v
         [发布任务到总线 TaskBus]
```

主 Agent 的核心能力分为四层：

1. **拆分层**：根据角色定位（需求拆分/实现/e2e/审查）生成任务包。
2. **调度层**：将任务包按优先级和依赖关系投递给子 Agent。
3. **验收层**：统一收敛结果，执行质量门禁与风险判定。
4. **演进层**：在打断、需求更新、失败重试时重建任务树。

### 9.6 子 Agent 打断与更新机制

支持两类打断：

- **主控打断**：主 Agent 根据全局风险主动中断某角色执行。
- **角色自打断**：角色发现阻塞（权限、依赖、环境）主动上报中断。

处理流程：

1. 角色进入 `INTERRUPTED` 并写入 `interrupts.md`。
2. 主 Agent 生成 `replan_action`（继续、换角色、降级执行、人工确认）。
3. 更新后的任务指令重新下发，角色从最近 `state/checkpoints/step-XXXX.md` 续跑。

### 9.7 复杂功能开发的标准角色编排模板

以“复杂功能开发需求”为例，推荐最小角色集：

- `req_split_agent`：将主需求拆成可执行子需求，输出任务树。
- `impl_agent`：实现与重构，输出代码与变更说明。
- `test_e2e_agent`：执行集成测试/e2e，输出测试报告。
- `review_agent`：执行质量审查与风险识别，输出准入结论。

主 Agent 统一把控：

- 入口需求是否可执行。
- 子任务是否覆盖完整。
- 测试与审查是否满足发布条件。
- 所有中间步骤是否已落盘可追溯。

---

## 10 故障恢复与数据完整性

### 10.1 故障场景与恢复

| 场景 | 检测方式 | 恢复策略 |
|------|----------|----------|
| 进程崩溃 | 心跳超时 | 读快照 → 重放 WAL → 重建状态 → 继续执行 |
| TMUX 断开 | 连接丢失 | 尝试重连 → 失败则日志模式 → 后台继续 |
| 磁盘满 | 写入失败 | 暂停生成 → 清理临时文件 → 归档旧数据 → 恢复 |
| 断电/崩溃 | 启动检查 | 扫描未完成会话 → 恢复每个会话 → 提示继续或取消 |

### 10.2 数据完整性校验

- **事件校验**: 签名验证
- **文件校验**: SHA256（可选 BLAKE3）
- **序列校验**: 序列号连续性
- **因果校验**: 父事件存在性

---

## 11 扩展性设计

### 11.1 插件系统

```
Hooks
├── on_init                    # 初始化时
├── on_stage_start             # 阶段开始时
├── on_stage_complete          # 阶段完成时
├── on_human_intervention      # 人工介入时
└── on_workflow_complete       # 工作流完成时

Plugin Types
├── NotificationPlugin         # 通知插件
├── MetricPlugin               # 指标插件
├── StoragePlugin              # 存储插件
└── CustomExecutor             # 自定义执行器
```

### 11.2 模板系统

```
Templates
├── 内置模板
│   ├── feature_dev.yaml       # 功能开发
│   ├── bug_fix.yaml           # Bug 修复
│   ├── code_review.yaml       # 代码审查
│   └── refactor.yaml          # 代码重构
├── 用户模板: ~/.octopus/templates/
└── 项目模板: .octopus/templates/
```

---

## 12 技术选型

| 技术 | 选择理由 |
|------|----------|
| **LangGraph** | 状态机编排、可视化支持、中断恢复、与 LangChain 生态兼容 |
| **UV** | 现代 Python 包管理、快速依赖解析、单文件配置 |
| **Pydantic** | 配置验证、类型安全 |
| **Typer** | CLI 开发、类型提示 |
| **Rich** | 终端美化、进度展示 |
| **TMUX** | 会话持久、多窗格、远程友好、可编程 |

---

## 13 附录

### 13.1 使用示例

```bash
# 1. 初始化项目
openoctopus init --template feature_dev

# 2. 编辑配置
vim octopus.yaml

# 3. 先校验配置
openoctopus validate --config ./octopus.yaml

# 4. 启动工作流
openoctopus run --config ./octopus.yaml --requirement ./docs/feature.md

# 5. 查看状态
openoctopus status --watch

# 6. 人工介入后恢复
openoctopus resume --session sess_abc123 --approve
```

### 13.2 极简配置示例

```yaml
version: "2.1"

runtime:
  workspace:
    root: ".octopus"
  tmux:
    enabled: false

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
  - id: "review_agent"
    name: "代码审查员"
    type: "react"
    llm_profile: "codex_cli"
    system_prompt: "你是一位严格的代码审查员..."
    tools: ["file_read", "file_write"]

stages:
  - id: "review"
    name: "代码审查"
    role: "review_agent"
    input:
      - type: "requirement_file"
        path: "./review.diff"
    output:
      - type: "artifact"
        name: "review_result"

transitions:
  - from: "review"
    to: "__END__"
```

### 13.3 设计健全性检查清单（建议）

1. 是否存在“事件先落盘，状态后推进”的可验证证据。
2. 是否每个阶段都配置了超时、重试和回路上限。
3. 是否每个 `shell_exec` 角色都受 allowlist/denylist 约束。
4. 是否能在删除内存态后，仅靠 `.octopus/sessions/{id}` 完成恢复。
5. 是否能用 `conclusion.md + master_schedule.md` 解释最终决策链。
6. 是否具备失败降级路径（暂停/人工接管/中止），而不是无限重试。
7. 是否能把最终报告追溯到具体事件、角色、回合和产物版本。

---
