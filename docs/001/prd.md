# OpenOctopus 阶段一：MVP 可运行版本

## 1. 概述

### 1.1 目标
搭建**可实际运行**的项目基础。阶段一完成后，用户可以通过 CLI 初始化项目、配置并**实际执行**最简单的单 Stage 工作流。

**核心原则**：第一个版本就要"用起来"，而非仅静态配置。

### 1.2 范围
| 模块 | 功能 | 优先级 |
|------|------|--------|
| 项目初始化 | UV 项目、目录结构、配置生成 | P0 |
| 配置系统 | Pydantic 模型、加载、校验 | P0 |
| **简单执行器** | **单 Stage 顺序执行** | **P0** |
| **Session 管理** | **session 创建、状态追踪** | **P0** |
| **产物管理** | **artifact 生成、存储** | **P0** |
| CLI | init, validate, **run**, version | P0 |

### 1.3 非目标（阶段二实现）
- TMUX 多窗格集成
- LangGraph 复杂编排
- 多角色并行执行
- 条件流转（transitions 仅支持简单顺序）
- 循环/重试逻辑
- 人机交互中断

### 1.4 执行模式（阶段一）
阶段一只支持**最简单的单线执行**：
```
Stage A -> Stage B -> Stage C -> END
```
- 按 stages 数组顺序执行
- 每个 stage 执行对应的 role
- role 类型只实现 `simple`（单次执行）
- 产物写入 `.octopus/sessions/{id}/artifacts/`

---

## 2. 功能设计

### 2.1 项目初始化 (openoctopus init)

**命令**：`openoctopus init [--template <name>] [--force] [--path <dir>]`

**目录结构（阶段一）**：
```
{target_path}/
├── octopus.yaml              # 工作流配置
└── .octopus/                 # 工作目录
    ├── sessions/             # 会话目录
    │   └── {session_id}/     # 每次 run 创建一个
    │       ├── session.state.md      # 状态文件
    │       ├── timeline.md           # 执行时间线
    │       └── artifacts/            # 产物目录
    │           └── {artifact_name}.md
    ├── cache/                # 缓存（预留）
    └── logs/                 # 日志（预留）
```

**默认模板**：
生成一个可直接运行的简单配置：读取需求文件 -> 生成分析报告。

```yaml
version: "2.1"

meta:
  workflow_id: "hello-world"
  name: "Hello World Workflow"

runtime:
  workspace:
    root: ".octopus"

llm_profiles:
  default:
    provider: "claude_code"
    mode: "cli"

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"
    file_write:
      module: "openoctopus.tools.file"
      class: "FileWriteTool"

roles:
  - id: "analyzer"
    name: "需求分析器"
    type: "simple"
    llm_profile: "default"
    system_prompt: "你是一位需求分析师。请分析给定的需求文档，输出结构化的分析结果。"
    tools: ["file_read", "file_write"]

stages:
  - id: "analyze"
    name: "需求分析"
    role: "analyzer"
    input:
      - type: "requirement_file"
        path: "./requirement.md"
    output:
      - type: "artifact"
        name: "analysis_report"

transitions:
  - from: "analyze"
    to: "__END__"
```

---

### 2.2 配置校验 (openoctopus validate)

**命令**：`openoctopus validate --config <path> [--format table|json]`

**阶段一校验规则**：
1. **结构校验**：Pydantic 模型验证
2. **引用校验**：stage.role, role.llm_profile, role.tools 必须存在
3. **简化校验**：阶段一不校验环路（只支持简单顺序）
4. **安全警告**：shell_exec 需要 security 配置

---

### 2.3 工作流执行 (openoctopus run) ⭐ 核心功能

**命令**：`openoctopus run --config <path> --requirement <path> [--session <id>]`

**执行流程**：
```
1. 加载并校验配置
2. 创建 session（生成唯一ID，创建目录结构）
3. 按顺序遍历 stages 数组
   3.1 对每个 stage：
       - 写入状态: RUNNING
       - 执行对应 role（调用 LLM CLI）
       - 收集输出产物
       - 写入状态: COMPLETED / FAILED
4. 生成最终报告
5. 输出 session_id 和结果摘要
```

**状态文件格式** (`.octopus/sessions/{id}/session.state.md`)：
```markdown
# Session State

## Metadata
- session_id: sess_abc123
- workflow_id: hello-world
- status: RUNNING | COMPLETED | FAILED
- created_at: 2026-03-03T10:00:00Z
- updated_at: 2026-03-03T10:05:00Z

## Stages
| Stage | Status | Started | Completed |
|-------|--------|---------|-----------|
| analyze | COMPLETED | 10:00:00 | 10:05:00 |

## Artifacts
- analysis_report: `.octopus/sessions/sess_abc123/artifacts/analysis_report.md`
```

**时间线格式** (`.octopus/sessions/{id}/timeline.md`)：
```markdown
# Execution Timeline

### 10:00:00 - SESSION_CREATED
session_id: sess_abc123

### 10:00:01 - STAGE_STARTED
stage_id: analyze
role_id: analyzer

### 10:05:00 - STAGE_COMPLETED
stage_id: analyze
artifacts: ["analysis_report"]

### 10:05:01 - SESSION_COMPLETED
total_stages: 1
```

**Role 执行（simple 类型）**：
```python
# 阶段一简化执行流程
def execute_simple_role(role: Role, stage: Stage, context: dict) -> ExecutionResult:
    1. 构建 prompt = system_prompt + 输入内容
    2. 调用 llm_profile 对应的 CLI（如 `claude` 或 `codex`）
    3. 捕获输出
    4. 将输出写入 artifact 文件
    5. 返回 ExecutionResult(status="completed", artifacts=[...])
```

---

### 2.4 状态查询 (openoctopus status)

**命令**：`openoctopus status --session <id>`

**输出示例**：
```
Session: sess_abc123
Workflow: hello-world
Status: COMPLETED
Created: 2026-03-03 10:00:00
Duration: 5m 30s

Stages:
  ✓ analyze (COMPLETED)

Artifacts:
  - analysis_report: .octopus/sessions/sess_abc123/artifacts/analysis_report.md
```

---

## 3. 数据模型（阶段一简化版）

### 3.1 配置模型

保留完整模型定义（见原 PRD），但阶段一只使用子集：
- `role.type`: 只支持 `"simple"`
- `transition`: 只支持简单 `from -> to`，不支持 condition
- `stage.input`: 只支持 `requirement_file` 类型
- `stage.output`: 只支持单个 artifact

### 3.2 运行时模型

```python
class Session(BaseModel):
    """会话状态"""
    session_id: str
    workflow_id: str
    config_path: Path
    status: Literal["CREATED", "RUNNING", "COMPLETED", "FAILED"]
    created_at: datetime
    updated_at: datetime
    stages_status: dict[str, StageExecution]  # stage_id -> 执行状态
    artifacts: list[Artifact]  # 生成的产物列表

class StageExecution(BaseModel):
    """Stage 执行状态"""
    stage_id: str
    status: Literal["PENDING", "RUNNING", "COMPLETED", "FAILED"]
    started_at: datetime | None
    completed_at: datetime | None
    error_message: str | None

class Artifact(BaseModel):
    """产物"""
    name: str
    path: Path
    stage_id: str
    created_at: datetime
    content_hash: str  # sha256

class ExecutionResult(BaseModel):
    """Role 执行结果"""
    status: Literal["completed", "failed", "skipped"]
    output: str  # 原始输出内容
    artifacts: list[str]  # artifact 名称列表
    error_message: str | None
```

---

## 4. 模块设计

### 4.1 目录结构

```
openoctopus/
├── __init__.py
├── __main__.py
├── cli/
│   ├── __init__.py
│   ├── main.py              # Typer 主应用
│   ├── commands/
│   │   ├── __init__.py
│   │   ├── init.py
│   │   ├── validate.py
│   │   ├── run.py           # ⭐ 实际执行逻辑
│   │   └── status.py        # 查询状态
│   └── utils/
│       ├── console.py       # Rich 输出
│       └── formatters.py
├── config/
│   ├── __init__.py
│   ├── models.py            # Pydantic 配置模型
│   ├── loader.py            # YAML 加载
│   ├── validator.py         # 配置校验
│   └── templates.py         # 内置模板
├── core/
│   ├── __init__.py
│   ├── constants.py         # 版本、常量
│   ├── exceptions.py        # 异常定义
│   └── utils.py             # 通用工具
└── executor/                # ⭐ 执行器模块（新增）
    ├── __init__.py
    ├── session.py           # Session 管理
    ├── runner.py            # 工作流执行器
    ├── role_executor.py     # Role 执行逻辑
    └── artifact_manager.py  # 产物管理
```

### 4.2 核心类

#### SessionManager
```python
class SessionManager:
    """会话管理器"""

    def create_session(self, config: OctopusConfig, config_path: Path) -> Session:
        """创建新 session，生成目录结构"""

    def load_session(self, session_id: str) -> Session:
        """从 state 文件加载 session"""

    def update_session(self, session: Session) -> None:
        """更新 session 状态文件"""

    def list_sessions(self) -> list[Session]:
        """列出所有 sessions"""
```

#### WorkflowRunner
```python
class WorkflowRunner:
    """工作流执行器"""

    def __init__(self, config: OctopusConfig, session: Session):
        self.config = config
        self.session = session

    def run(self, requirement_path: Path) -> ExecutionSummary:
        """
        执行工作流

        阶段一简化逻辑：
        1. 按 stages 数组顺序执行
        2. 每个 stage 调用 RoleExecutor
        3. 收集产物
        4. 更新 session 状态
        """

    def _run_stage(self, stage: Stage, requirement_path: Path) -> ExecutionResult:
        """执行单个 stage"""
```

#### RoleExecutor
```python
class RoleExecutor:
    """Role 执行器"""

    def execute(self, role: Role, stage: Stage, inputs: dict) -> ExecutionResult:
        """
        执行 role

        阶段一只实现 simple 类型：
        - 构建完整 prompt
        - 调用 LLM CLI
        - 返回输出
        """

    def _call_llm(self, profile: LLMProfile, prompt: str) -> str:
        """调用 LLM CLI（claude/codex）"""
```

#### ArtifactManager
```python
class ArtifactManager:
    """产物管理器"""

    def __init__(self, session_dir: Path):
        self.artifacts_dir = session_dir / "artifacts"

    def save_artifact(self, name: str, content: str, stage_id: str) -> Artifact:
        """保存产物"""

    def load_artifact(self, name: str) -> str:
        """读取产物内容"""

    def list_artifacts(self) -> list[Artifact]:
        """列出所有产物"""
```

---

## 5. CLI 设计

### 5.1 命令列表

| 命令 | 功能 | 示例 |
|------|------|------|
| `init` | 初始化项目 | `openoctopus init` |
| `validate` | 校验配置 | `openoctopus validate -c octopus.yaml` |
| `run` | 执行工作流 | `openoctopus run -c octopus.yaml -r req.md` |
| `status` | 查询状态 | `openoctopus status -s sess_abc123` |
| `list` | 列出 sessions | `openoctopus list` |

### 5.2 Run 命令详细

```bash
openoctopus run \
  --config ./octopus.yaml \
  --requirement ./feature.md \
  [--session sess_xxx]  # 可选，用于重跑
```

**执行输出示例**：
```
$ openoctopus run -c octopus.yaml -r feature.md

OpenOctopus Runner
═══════════════════════════════════════════════════

Loading config: octopus.yaml ✓
Validating config ✓
Creating session: sess_7f8a9b ✓

Executing Workflow: hello-world
───────────────────────────────────────────────────

Stage 1/1: analyze [analyzer]
  Input: feature.md
  Running...
  ✓ Completed (45s)
  Output: analysis_report.md

═══════════════════════════════════════════════════
Workflow completed successfully!

Session ID: sess_7f8a9b
Artifacts:
  - .octopus/sessions/sess_7f8a9b/artifacts/analysis_report.md

View details: openoctopus status --session sess_7f8a9b
```

---

## 6. 产物格式

### 6.1 Artifact 文件结构

```markdown
# analysis_report

## Metadata
- artifact_name: analysis_report
- stage_id: analyze
- role_id: analyzer
- created_at: 2026-03-03T10:05:00Z
- session_id: sess_7f8a9b

## Content

[Role 生成的内容]
```

### 6.2 Session State 文件

状态文件使用 Markdown 格式，便于人工阅读和调试：
- 每次状态变更覆盖写入
- 包含完整的执行上下文
- 与 timeline.md 互补（state 是当前态，timeline 是历史）

---

## 7. 错误处理

### 7.1 退出码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1 | 通用错误 |
| 2 | 配置错误 |
| 3 | 校验失败 |
| 4 | 初始化失败 |
| 5 | 执行失败（stage 执行出错） |
| 6 | Session 不存在 |

### 7.2 执行错误处理

```python
try:
    result = role_executor.execute(...)
except LLMNotFoundError:
    # CLI 工具未安装
    console.print("[red]Error: 'claude' command not found. Please install Claude CLI.[/red]")
    raise typer.Exit(5)
except ExecutionTimeout:
    # 执行超时
    session.status = "FAILED"
    session_manager.update_session(session)
    console.print("[red]Error: Stage execution timed out.[/red]")
    raise typer.Exit(5)
```

---

## 8. 快速开始示例

用户从 0 到运行第一个工作流：

```bash
# 1. 安装
pip install openoctopus

# 2. 初始化项目
mkdir my-project && cd my-project
openoctopus init

# 3. 创建需求文档
echo "# Feature Request
实现一个用户登录功能，包含：
- 用户名密码验证
- JWT Token 生成
- 登录状态保持" > requirement.md

# 4. 运行工作流
openoctopus run -c octopus.yaml -r requirement.md

# 5. 查看结果
openoctopus status -s sess_xxx
cat .octopus/sessions/sess_xxx/artifacts/analysis_report.md
```

---

## 9. 阶段一验收标准

### 功能验收
- [ ] `openoctopus init` 生成可运行的配置
- [ ] `openoctopus validate` 正确校验配置
- [ ] `openoctopus run` 能执行单 stage 工作流
- [ ] `openoctopus status` 显示执行状态
- [ ] 执行完成后产物文件正确生成
- [ ] session 状态文件正确维护

### 质量验收
- [ ] 支持 `simple` role 类型完整执行
- [ ] 支持 `claude_code` 和 `codex` LLM provider
- [ ] 执行失败时错误信息清晰
- [ ] 单元测试覆盖率 ≥ 80%
- [ ] E2E 测试覆盖 init -> validate -> run -> status 完整流程

### 用户体验
- [ ] Rich 终端美化输出
- [ ] 执行进度实时显示
- [ ] 清晰的产物路径提示
- [ ] 完整的 help 信息

---

## 10. 后续阶段扩展

| 阶段二功能 | 当前预留点 |
|-----------|-----------|
| TMUX 多窗格 | executor 支持多 pane 调用 |
| LangGraph 编排 | runner 支持 graph 执行 |
| 条件流转 | transition.condition 解析 |
| 循环/重试 | runner 支持循环检测 |
| 人机交互 | session 状态支持 WAITING |
