# Plan 02: Pydantic 配置模型实现

## 目标
实现完整的 Pydantic v2 配置模型，覆盖 YAML 配置的所有字段。

## 任务清单

### 2.1 核心常量定义 (core/constants.py)
- [ ] 定义版本常量 `VERSION = "0.1.0"`
- [ ] 定义 `DEFAULT_CONFIG_VERSION = "2.1"`
- [ ] 定义 `END_STAGE = "__END__"`
- [ ] 定义支持的 providers: `["claude_code", "codex", "openai", "anthropic"]`
- [ ] 定义支持的 role types: `["simple", "react", "plan_exec"]`

### 2.2 异常定义 (core/exceptions.py)
- [ ] `OpenOctopusError` - 基础异常
- [ ] `ConfigError` - 配置错误（带 location 字段）
- [ ] `ValidationError` - 校验错误（带 errors 列表）
- [ ] `InitError` - 初始化错误
- [ ] `LoadError` - 加载错误

### 2.3 MetaConfig 模型
```python
class MetaConfig(BaseModel):
    model_config = ConfigDict(extra="forbid")

    workflow_id: str = Field(..., pattern=r"^[a-z0-9_-]+$")
    name: str
    owner: str | None = None
    description: str | None = None
```

### 2.4 Runtime 相关模型
- [ ] `WorkspaceConfig`
- [ ] `TmuxConfig`
- [ ] `CheckpointConfig`
- [ ] `RecoveryConfig`
- [ ] `SchedulerConfig`
- [ ] `RuntimeConfig`（组合以上）

### 2.5 LLM 相关模型
```python
class LLMProfile(BaseModel):
    provider: Literal["claude_code", "codex", "openai", "anthropic"]
    mode: Literal["cli", "api", "sdk"]
    cli_path: str | None = None
    api_key: str | None = None
    base_url: str | None = None
    model: str | None = None
    max_tokens: int = Field(default=4096, gt=0)
    temperature: float = Field(default=0.2, ge=0.0, le=2.0)
```

### 2.6 ToolRegistry 模型
- [ ] `ToolSpec`（处理 alias "class" -> "class_name"）
- [ ] `MCPServerSpec`
- [ ] `ToolRegistry`

### 2.7 Security 模型
- [ ] `ShellSecurityConfig`
- [ ] `PathSecurityConfig`
- [ ] `SecurityConfig`

### 2.8 Policies 模型
- [ ] `RetryPolicy`
- [ ] `TimeoutPolicy`
- [ ] `LoopGuardPolicy`
- [ ] `HumanGatePolicy`
- [ ] `ArtifactPolicy`
- [ ] `PoliciesConfig`

### 2.9 Role 模型
```python
class ReactConfig(BaseModel):
    max_iterations: int = Field(default=8, gt=0)

class Role(BaseModel):
    model_config = ConfigDict(extra="forbid")

    id: str = Field(..., pattern=r"^[a-z0-9_]+$")
    name: str
    type: Literal["simple", "react", "plan_exec"]
    llm_profile: str
    system_prompt: str
    tools: list[str] | None = None
    react_config: ReactConfig | None = None
```

### 2.10 Stage 模型
- [ ] `StageInput`（支持 type/path/ref/optional）
- [ ] `StageOutput`（支持 type/name）
- [ ] `Stage`（包含 input/output 列表）

### 2.11 Transition 模型
```python
class ConditionRule(BaseModel):
    type: Literal["expression", "aggregate", "role_aggregate"]
    expr: str | None = None
    mode: Literal["all", "any", "majority"] | None = None
    rules: list[str] | None = None

class Transition(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    from_: str = Field(..., alias="from")
    to: str | None = None
    condition: ConditionRule | None = None
    on_true: str | None = None
    on_false: str | None = None
```

### 2.12 根模型 OctopusConfig
```python
class OctopusConfig(BaseModel):
    model_config = ConfigDict(extra="forbid")

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

    # 辅助方法
    def get_role(self, role_id: str) -> Role | None
    def get_stage(self, stage_id: str) -> Stage | None
    def get_llm_profile(self, name: str) -> LLMProfile | None
```

### 2.13 校验结果模型
```python
class ValidationErrorItem(BaseModel):
    type: Literal["structure", "reference", "loop", "security"]
    field: str
    message: str
    location: list[str]

class ValidationResult(BaseModel):
    valid: bool
    errors: list[ValidationErrorItem]
    warnings: list[ValidationErrorItem]
```

## 验收标准
- [ ] 所有模型可正确实例化
- [ ] 类型检查通过（`mypy` 或 IDE 检查）
- [ ] 支持 `model.model_dump(by_alias=True)` 正确序列化
- [ ] 单元测试覆盖所有模型类
- [ ] 验证示例配置可正确解析

## 测试要求
```python
# 测试示例
def test_minimal_config():
    config = OctopusConfig(
        version="2.1",
        meta={"workflow_id": "test", "name": "Test"},
        runtime={"workspace": {"root": ".octopus"}},
        llm_profiles={"codex": {"provider": "codex", "mode": "cli"}},
        tool_registry={"builtin": {}},
        roles=[{"id": "test", "name": "Test", "type": "simple",
                "llm_profile": "codex", "system_prompt": "test"}],
        stages=[{"id": "s1", "name": "S1", "role": "test",
                 "input": [{"type": "requirement_file", "path": "./test.md"}],
                 "output": [{"type": "artifact", "name": "result"}]}],
        transitions=[{"from": "s1", "to": "__END__"}]
    )
    assert config.meta.workflow_id == "test"
```

## 预计耗时
60 分钟
