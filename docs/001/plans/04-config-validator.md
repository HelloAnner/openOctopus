# Plan 04: 配置校验器实现

## 目标
实现完整的配置校验逻辑，包括引用校验、环路检测和安全校验。

## 任务清单

### 4.1 校验器类结构 (config/validator.py)

```python
from typing import Any
from collections import defaultdict

from .models import (
    OctopusConfig, ValidationResult, ValidationErrorItem,
    Stage, Transition, Role
)
from ..core.constants import END_STAGE


class ConfigValidator:
    """
    配置校验器

    执行以下校验：
    1. 引用校验：stage.role, transition.from/to/on_true/on_false, role.llm_profile, role.tools
    2. 环路检测：transition 图中是否存在环路
    3. 安全校验：shell_exec 工具是否配置了安全措施
    """

    def __init__(self, config: OctopusConfig, raw_data: dict[str, Any] | None = None):
        self.config = config
        self.raw_data = raw_data or {}
        self.errors: list[ValidationErrorItem] = []
        self.warnings: list[ValidationErrorItem] = []

        # 构建索引加速查询
        self._role_ids = {r.id for r in config.roles}
        self._stage_ids = {s.id for s in config.stages}
        self._llm_profile_names = set(config.llm_profiles.keys())
        self._builtin_tools = set(config.tool_registry.builtin.keys()) if config.tool_registry.builtin else set()

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

    def _add_error(self, type_: str, field: str, message: str, location: list[str]) -> None:
        """添加错误"""
        self.errors.append(ValidationErrorItem(
            type=type_,
            field=field,
            message=message,
            location=location
        ))

    def _add_warning(self, type_: str, field: str, message: str, location: list[str]) -> None:
        """添加警告"""
        self.warnings.append(ValidationErrorItem(
            type=type_,
            field=field,
            message=message,
            location=location
        ))
```

### 4.2 引用校验实现

```python
    def _validate_references(self) -> None:
        """校验所有引用关系"""
        self._validate_role_references()
        self._validate_llm_profile_references()
        self._validate_tool_references()
        self._validate_stage_references()
        self._validate_transition_references()

    def _validate_role_references(self) -> None:
        """校验 stage.role 引用"""
        for i, stage in enumerate(self.config.stages):
            if stage.role not in self._role_ids:
                self._add_error(
                    type_="reference",
                    field=f"stages[{i}].role",
                    message=f"Stage '{stage.id}' references undefined role '{stage.role}'",
                    location=["stages", i, "role"]
                )

    def _validate_llm_profile_references(self) -> None:
        """校验 role.llm_profile 引用"""
        for i, role in enumerate(self.config.roles):
            if role.llm_profile not in self._llm_profile_names:
                self._add_error(
                    type_="reference",
                    field=f"roles[{i}].llm_profile",
                    message=f"Role '{role.id}' references undefined llm_profile '{role.llm_profile}'",
                    location=["roles", i, "llm_profile"]
                )

    def _validate_tool_references(self) -> None:
        """校验 role.tools 引用"""
        for i, role in enumerate(self.config.roles):
            if not role.tools:
                continue
            for j, tool in enumerate(role.tools):
                if tool not in self._builtin_tools:
                    self._add_error(
                        type_="reference",
                        field=f"roles[{i}].tools[{j}]",
                        message=f"Role '{role.id}' references undefined tool '{tool}'",
                        location=["roles", i, "tools", j]
                    )

    def _validate_stage_references(self) -> None:
        """校验 stage 相关的 artifact 引用（预留）"""
        # 阶段一暂不实现 artifact 引用校验
        # 后续阶段实现 input.ref 指向已定义 artifact 的校验
        pass

    def _validate_transition_references(self) -> None:
        """校验 transition 引用"""
        valid_targets = self._stage_ids | {END_STAGE}

        for i, trans in enumerate(self.config.transitions):
            # 校验 from
            if trans.from_ not in self._stage_ids:
                self._add_error(
                    type_="reference",
                    field=f"transitions[{i}].from",
                    message=f"Transition references undefined stage '{trans.from_}'",
                    location=["transitions", i, "from"]
                )

            # 校验 to
            if trans.to is not None and trans.to not in valid_targets:
                self._add_error(
                    type_="reference",
                    field=f"transitions[{i}].to",
                    message=f"Transition references undefined target '{trans.to}'",
                    location=["transitions", i, "to"]
                )

            # 校验 on_true/on_false
            if trans.on_true is not None and trans.on_true not in valid_targets:
                self._add_error(
                    type_="reference",
                    field=f"transitions[{i}].on_true",
                    message=f"Transition on_true references undefined target '{trans.on_true}'",
                    location=["transitions", i, "on_true"]
                )

            if trans.on_false is not None and trans.on_false not in valid_targets:
                self._add_error(
                    type_="reference",
                    field=f"transitions[{i}].on_false",
                    message=f"Transition on_false references undefined target '{trans.on_false}'",
                    location=["transitions", i, "on_false"]
                )

            # 条件校验
            if trans.condition is not None:
                if trans.on_true is None and trans.on_false is None:
                    self._add_warning(
                        type_="reference",
                        field=f"transitions[{i}].condition",
                        message=f"Transition has condition but no on_true/on_false targets",
                        location=["transitions", i, "condition"]
                    )
```

### 4.3 环路检测实现

```python
    def _validate_loops(self) -> None:
        """检测 transition 图中的环路"""
        # 构建邻接表
        graph: dict[str, set[str]] = defaultdict(set)

        for trans in self.config.transitions:
            if trans.from_ in self._stage_ids:
                # 收集所有可能的目标
                targets = []
                if trans.to:
                    targets.append(trans.to)
                if trans.on_true:
                    targets.append(trans.on_true)
                if trans.on_false:
                    targets.append(trans.on_false)

                for target in targets:
                    if target in self._stage_ids:
                        graph[trans.from_].add(target)

        # DFS 检测环路
        visited: set[str] = set()
        rec_stack: set[str] = set()
        path: list[str] = []

        def has_cycle(node: str) -> bool:
            visited.add(node)
            rec_stack.add(node)
            path.append(node)

            for neighbor in graph.get(node, []):
                if neighbor not in visited:
                    if has_cycle(neighbor):
                        return True
                elif neighbor in rec_stack:
                    # 发现环路
                    cycle_start = path.index(neighbor)
                    cycle_path = path[cycle_start:] + [neighbor]
                    self._add_error(
                        type_="loop",
                        field="transitions",
                        message=f"Cycle detected: {' -> '.join(cycle_path)}",
                        location=["transitions"]
                    )
                    return True

            path.pop()
            rec_stack.remove(node)
            return False

        for stage_id in self._stage_ids:
            if stage_id not in visited:
                has_cycle(stage_id)
```

### 4.4 安全校验实现

```python
    def _validate_security(self) -> None:
        """安全相关校验"""
        has_shell_security = (
            self.config.security is not None and
            self.config.security.shell is not None
        )

        for i, role in enumerate(self.config.roles):
            if not role.tools:
                continue

            # 检查是否使用了 shell_exec
            if "shell_exec" in role.tools:
                if not has_shell_security:
                    self._add_warning(
                        type_="security",
                        field=f"roles[{i}].tools",
                        message=f"Role '{role.id}' uses 'shell_exec' without security.shell configuration",
                        location=["roles", i, "tools"]
                    )

            # 检查 denylist 是否配置
            if has_shell_security and self.config.security.shell:
                denylist = self.config.security.shell.denylist_keywords or []
                if "shell_exec" in role.tools and not denylist:
                    self._add_warning(
                        type_="security",
                        field=f"roles[{i}].tools",
                        message=f"Role '{role.id}' uses 'shell_exec' but security.shell.denylist_keywords is empty",
                        location=["roles", i, "tools"]
                    )
```

### 4.5 便捷函数

```python
def validate_config(config: OctopusConfig, raw_data: dict | None = None) -> ValidationResult:
    """便捷函数：校验配置"""
    validator = ConfigValidator(config, raw_data)
    return validator.validate()
```

## 验收标准
- [ ] 能正确检测所有引用错误
- [ ] 能正确检测 transition 环路
- [ ] 能正确发出安全警告
- [ ] 错误信息包含准确的位置信息
- [ ] 单元测试覆盖率达到 90%
- [ ] 支持并发执行多个校验（无状态设计）

## 测试用例

```python
# tests/unit/test_validator.py

class TestConfigValidator:
    def test_valid_config(self, minimal_config):
        """测试有效配置"""
        validator = ConfigValidator(minimal_config)
        result = validator.validate()
        assert result.valid is True
        assert len(result.errors) == 0

    def test_invalid_role_reference(self):
        """测试 role 引用错误"""
        config = create_config_with(
            stages=[{"id": "s1", "role": "nonexistent", ...}]
        )
        validator = ConfigValidator(config)
        result = validator.validate()
        assert result.valid is False
        assert any("nonexistent" in e.message for e in result.errors)

    def test_transition_loop(self):
        """测试环路检测"""
        config = create_config_with(
            stages=[{"id": "a"}, {"id": "b"}],
            transitions=[
                {"from": "a", "to": "b"},
                {"from": "b", "to": "a"}
            ]
        )
        validator = ConfigValidator(config)
        result = validator.validate()
        assert result.valid is False
        assert any("Cycle" in e.message for e in result.errors)

    def test_security_warning(self):
        """测试安全警告"""
        config = create_config_with(
            roles=[{"id": "r1", "tools": ["shell_exec"], ...}],
            security=None
        )
        validator = ConfigValidator(config)
        result = validator.validate()
        assert result.valid is True  # 警告不阻止通过
        assert len(result.warnings) > 0
```

## 预计耗时
60 分钟
