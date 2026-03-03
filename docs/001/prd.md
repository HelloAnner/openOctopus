# OpenOctopus 阶段一：基础架构与项目骨架

## 1. 概述

### 1.1 目标
搭建可运行的项目基础，实现核心配置解析与目录结构初始化。本阶段完成后，用户可以通过 CLI 初始化项目、编写 YAML 配置并进行校验。

### 1.2 范围
- 项目初始化（UV + 依赖管理）
- 配置模型定义（Pydantic v2）
- 配置校验逻辑
- 目录结构初始化
- CLI 基础命令（init, validate）

### 1.3 非目标
- 不实现完整的工作流执行（run 命令仅占位）
- 不实现 TMUX 集成
- 不实现 LangGraph 编排
- 不实现角色执行逻辑

---

## 2. 功能设计

### 2.1 项目初始化 (openoctopus init)

#### 命令规格
```bash
openoctopus init [--template <name>] [--force] [--path <dir>]
```

#### 参数说明
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| --template | str | "default" | 初始化模板名称 |
| --force | bool | false | 强制覆盖已存在的配置文件 |
| --path | Path | "." | 目标目录路径 |

#### 执行流程
```
1. 检查目标路径是否存在，不存在则创建
2. 检查 .octopus/ 目录是否存在
   - 存在且非 force: 报错退出
   - 存在且 force: 保留并提示
3. 创建 .octopus/ 目录结构
4. 根据模板生成 octopus.yaml
5. 输出成功信息
```

#### 目录结构
```
{target_path}/
├── octopus.yaml              # 生成的配置文件
└── .octopus/                 # 工作目录（阶段一只创建空结构）
    ├── sessions/             # 会话数据目录
    ├── artifacts/            # 产物存储目录
    ├── logs/                 # 日志目录
    └── cache/                # 缓存目录
```

#### 默认模板 (default)
生成最小可用的配置，包含：
- 基础 meta 信息
- 简单的 runtime 配置（tmux 禁用）
- 示例 llm_profile（codex_cli）
- 示例 tool_registry（file_read, file_write）
- 单个 role（review_agent 示例）
- 单个 stage（review）
- 简单的 transition

### 2.2 配置校验 (openoctopus validate)

#### 命令规格
```bash
openoctopus validate --config <path> [--verbose] [--format <json|yaml|table>]
```

#### 参数说明
| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| --config | Path | 必填 | YAML 配置文件路径 |
| --verbose | bool | false | 输出详细校验信息 |
| --format | str | "table" | 错误输出格式 |

#### 执行流程
```
1. 读取 YAML 文件
2. 解析为 Pydantic 模型
   - 失败: 输出解析错误，退出码 1
3. 执行引用校验
   - roles 引用校验
   - stages 引用校验
   - transitions 引用校验
   - artifacts 引用校验
4. 执行环路检测
5. 执行安全校验
6. 输出校验结果
   - 通过: 输出成功信息，退出码 0
   - 失败: 输出错误列表，退出码 1
```

#### 校验规则

**结构校验 (Pydantic)**
- 字段类型匹配
- 必填字段存在
- 枚举值合法
- 嵌套结构正确

**引用校验**
1. `stages[].role` 必须引用已定义的 `roles[].id`
2. `transitions[].from` 必须引用已定义的 `stages[].id`
3. `transitions[].to/on_true/on_false` 必须为有效 stage_id 或 `__END__`
4. `roles[].llm_profile` 必须引用已定义的 `llm_profiles` 键
5. `roles[].tools[]` 必须能在 `tool_registry.builtin` 中找到

**环路校验**
- 检测 transition 图中的环路
- 记录环路路径用于错误报告

**安全校验**
- 启用 `shell_exec` 工具的角色必须有 `security.shell` 配置
- `security.shell.denylist_keywords` 非空时进行检查

### 2.3 CLI 基础框架

#### 全局选项
```bash
openoctopus [--version] [--help] [--verbose] <command>
```

#### 命令列表（阶段一）
| 命令 | 功能 | 状态 |
|------|------|------|
| init | 初始化项目 | 完整实现 |
| validate | 校验配置 | 完整实现 |
| run | 执行工作流 | 占位（输出"暂未实现"） |
| version | 显示版本 | 完整实现 |

---

## 3. 数据模型设计

### 3.1 配置模型 (Pydantic v2)

#### 根模型: OctopusConfig
```python
class OctopusConfig(BaseModel):
    version: Literal["2.1"]
    meta: MetaConfig
    runtime: RuntimeConfig
    llm_profiles: dict[str, LLMProfile]
    tool_registry: ToolRegistry
    security: SecurityConfig | None = None
    policies: PoliciesConfig | None = None
    roles: list[Role]
    stages: list[Stage]
    transitions: list[Transition]
```

#### MetaConfig
```python
class MetaConfig(BaseModel):
    workflow_id: str = Field(..., pattern=r"^[a-z0-9_-]+$")
    name: str
    owner: str | None = None
    description: str | None = None
```

#### RuntimeConfig
```python
class RuntimeConfig(BaseModel):
    workspace: WorkspaceConfig
    tmux: TmuxConfig | None = None
    checkpoint: CheckpointConfig | None = None
    recovery: RecoveryConfig | None = None
    scheduler: SchedulerConfig | None = None
```

#### WorkspaceConfig
```python
class WorkspaceConfig(BaseModel):
    root: str = ".octopus"
    sessions_dir: str = ".octopus/sessions"
    artifacts_dir: str = ".octopus/artifacts"
    logs_dir: str = ".octopus/logs"
```

#### LLMProfile
```python
class LLMProfile(BaseModel):
    provider: Literal["claude_code", "codex", "openai", "anthropic"]
    mode: Literal["cli", "api", "sdk"]
    cli_path: str | None = None
    api_key: str | None = None
    base_url: str | None = None
    model: str | None = None
    max_tokens: int = 4096
    temperature: float = 0.2
```

#### ToolRegistry
```python
class ToolRegistry(BaseModel):
    builtin: dict[str, ToolSpec] | None = None
    mcp: dict[str, MCPServerSpec] | None = None

class ToolSpec(BaseModel):
    module: str
    class_name: str = Field(..., alias="class")

class MCPServerSpec(BaseModel):
    command: str
    args: list[str] | None = None
    env: dict[str, str] | None = None
```

#### SecurityConfig
```python
class SecurityConfig(BaseModel):
    shell: ShellSecurityConfig | None = None
    path: PathSecurityConfig | None = None

class ShellSecurityConfig(BaseModel):
    allowlist_prefixes: list[str] | None = None
    denylist_keywords: list[str] | None = None

class PathSecurityConfig(BaseModel):
    writable_roots: list[str] | None = None
    read_only_roots: list[str] | None = None
```

#### PoliciesConfig
```python
class PoliciesConfig(BaseModel):
    retry: RetryPolicy | None = None
    timeout: TimeoutPolicy | None = None
    loop_guard: LoopGuardPolicy | None = None
    human_gate: HumanGatePolicy | None = None
    artifact: ArtifactPolicy | None = None

class RetryPolicy(BaseModel):
    max_retry_per_stage: int = 2
    backoff_seconds: list[int] = [5, 20]

class TimeoutPolicy(BaseModel):
    stage_timeout_seconds: int = 1800
    role_heartbeat_timeout_seconds: int = 120

class LoopGuardPolicy(BaseModel):
    max_rounds_per_task: int = 6
    min_quality_gain: float = 0.05

class HumanGatePolicy(BaseModel):
    on_high_risk: bool = True
    high_risk_threshold: float = 0.8

class ArtifactPolicy(BaseModel):
    hash_algo: Literal["sha256", "md5", "blake3"] = "sha256"
    keep_latest_versions: int = 5
```

#### Role
```python
class Role(BaseModel):
    id: str = Field(..., pattern=r"^[a-z0-9_]+$")
    name: str
    type: Literal["simple", "react", "plan_exec"]
    llm_profile: str
    system_prompt: str
    tools: list[str] | None = None
    react_config: ReactConfig | None = None

class ReactConfig(BaseModel):
    max_iterations: int = 8
```

#### Stage
```python
class StageInput(BaseModel):
    type: Literal["requirement_file", "user_prompt", "artifact"]
    path: str | None = None
    ref: str | None = None
    optional: bool = False

class StageOutput(BaseModel):
    type: Literal["artifact"]
    name: str

class Stage(BaseModel):
    id: str = Field(..., pattern=r"^[a-z0-9_]+$")
    name: str
    role: str
    input: list[StageInput]
    output: list[StageOutput]
```

#### Transition
```python
class ConditionRule(BaseModel):
    type: Literal["expression", "aggregate", "role_aggregate"]
    expr: str | None = None
    mode: Literal["all", "any", "majority"] | None = None
    rules: list[str] | None = None

class Transition(BaseModel):
    from_: str = Field(..., alias="from")
    to: str | None = None
    condition: ConditionRule | None = None
    on_true: str | None = None
    on_false: str | None = None
```

### 3.2 校验结果模型

```python
class ValidationError(BaseModel):
    type: Literal["structure", "reference", "loop", "security"]
    field: str
    message: str
    location: list[str]  # JSON Path 风格

class ValidationResult(BaseModel):
    valid: bool
    errors: list[ValidationError]
    warnings: list[ValidationError]
```

---

## 4. 技术设计

### 4.1 模块结构

```
openoctopus/
├── __init__.py              # 版本号
├── __main__.py              # 入口点
├── cli/
│   ├── __init__.py
│   ├── main.py              # Typer 应用定义
│   ├── commands/
│   │   ├── __init__.py
│   │   ├── init.py          # init 命令实现
│   │   ├── validate.py      # validate 命令实现
│   │   ├── run.py           # run 占位
│   │   └── version.py       # version 命令
│   └── utils/
│       ├── __init__.py
│       ├── console.py       # Rich console 封装
│       └── formatters.py    # 输出格式化
├── config/
│   ├── __init__.py
│   ├── models.py            # Pydantic 模型定义
│   ├── loader.py            # YAML 加载器
│   ├── validator.py         # 校验逻辑
│   └── templates.py         # 内置模板
└── core/
    ├── __init__.py
    ├── constants.py         # 常量定义
    └── exceptions.py        # 自定义异常
```

### 4.2 核心类设计

#### ConfigLoader
```python
class ConfigLoader:
    """配置加载器"""

    @staticmethod
    def load(path: Path) -> OctopusConfig:
        """加载并解析 YAML 配置"""

    @staticmethod
    def load_raw(path: Path) -> dict:
        """加载原始 YAML 数据"""
```

#### ConfigValidator
```python
class ConfigValidator:
    """配置校验器"""

    def __init__(self, config: OctopusConfig, raw_data: dict):
        self.config = config
        self.raw_data = raw_data
        self.errors: list[ValidationError] = []
        self.warnings: list[ValidationError] = []

    def validate(self) -> ValidationResult:
        """执行完整校验"""
        self._validate_references()
        self._validate_loops()
        self._validate_security()
        return ValidationResult(
            valid=len(self.errors) == 0,
            errors=self.errors,
            warnings=self.warnings
        )

    def _validate_references(self) -> None:
        """校验引用关系"""

    def _validate_loops(self) -> None:
        """检测环路"""

    def _validate_security(self) -> None:
        """安全校验"""
```

#### ProjectInitializer
```python
class ProjectInitializer:
    """项目初始化器"""

    def __init__(self, target_path: Path, template: str, force: bool = False):
        self.target_path = target_path
        self.template = template
        self.force = force

    def initialize(self) -> InitResult:
        """执行初始化"""

    def _create_directory_structure(self) -> None:
        """创建目录结构"""

    def _generate_config(self) -> Path:
        """生成配置文件"""
```

### 4.3 异常设计

```python
class OpenOctopusError(Exception):
    """基础异常"""
    pass

class ConfigError(OpenOctopusError):
    """配置错误"""
    def __init__(self, message: str, location: list[str] = None):
        super().__init__(message)
        self.location = location or []

class ValidationError(OpenOctopusError):
    """校验错误"""
    def __init__(self, errors: list[ValidationError]):
        self.errors = errors

class InitError(OpenOctopusError):
    """初始化错误"""
    pass
```

---

## 5. 错误处理

### 5.1 错误码

| 退出码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1 | 通用错误 |
| 2 | 配置错误（YAML 语法错误） |
| 3 | 校验失败 |
| 4 | 初始化失败（目录已存在等） |

### 5.2 错误输出格式

**Table 格式（默认）**
```
┌──────────┬─────────────────────┬─────────────────────────────────┐
│ Type     │ Field               │ Message                         │
├──────────┼─────────────────────┼─────────────────────────────────┤
│ reference│ stages[1].role      │ Role 'impl_agent' not found     │
│ loop     │ transitions         │ Cycle detected: a -> b -> a     │
└──────────┴─────────────────────┴─────────────────────────────────┘
```

**JSON 格式**
```json
{
  "valid": false,
  "errors": [
    {
      "type": "reference",
      "field": "stages[1].role",
      "message": "Role 'impl_agent' not found",
      "location": ["stages", 1, "role"]
    }
  ],
  "warnings": []
}
```

---

## 6. 内置模板

### 6.1 default 模板

见 prd.md 13.2 极简配置示例。

### 6.2 feature_dev 模板（预留结构）

阶段一只预留模板名，实际生成与 default 相同，后续阶段完善。

---

## 7. 接口契约

### 7.1 配置文件加载

- 支持标准 YAML 1.2 语法
- 支持环境变量替换（`${VAR}` 语法，预留接口）
- 文件编码：UTF-8

### 7.2 路径处理

- 配置文件中的相对路径基于配置文件所在目录
- CLI 中的相对路径基于当前工作目录
- 统一使用 Path 对象，支持跨平台

---

## 8. 开发约定

### 8.1 代码规范
- Python 3.10+ 类型注解
- Pydantic v2 模型定义
- Ruff 代码格式化

### 8.2 测试要求
- 单元测试覆盖率 > 80%
- E2E 测试覆盖主要场景
- 边界条件测试

### 8.3 文档要求
- 公共 API 必须带 docstring
- 复杂逻辑必须带注释
- CLI help 文本必须完整

---

## 9. 产出物

| 文件/目录 | 说明 |
|-----------|------|
| `openoctopus/` | Python 包源码 |
| `pyproject.toml` | UV 项目配置 |
| `README.md` | 使用说明 |
| `tests/` | 测试代码 |
| `octopus.yaml` | 示例配置文件 |

---

## 10. 后续阶段衔接点

| 模块 | 阶段二扩展 |
|------|-----------|
| cli/commands/run.py | 实现完整 run 逻辑 |
| core/executor.py | 新增执行器模块 |
| core/session.py | 新增会话管理 |
| core/bus.py | 新增事件总线 |
| core/tmux.py | 新增 TMUX 管理 |
