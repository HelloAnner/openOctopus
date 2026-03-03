# Plan 03: 配置加载器实现

## 目标
实现 YAML 配置加载器，支持从文件加载并解析为 Pydantic 模型。

## 任务清单

### 3.1 基础加载功能 (config/loader.py)

#### ConfigLoader 类
```python
import yaml
from pathlib import Path
from typing import Any

from .models import OctopusConfig
from ..core.exceptions import LoadError, ConfigError


class ConfigLoader:
    """配置加载器，负责从 YAML 文件加载配置"""

    @staticmethod
    def load(path: Path | str) -> OctopusConfig:
        """
        加载并解析 YAML 配置文件

        Args:
            path: 配置文件路径

        Returns:
            OctopusConfig: 解析后的配置对象

        Raises:
            LoadError: 文件不存在或无法读取
            ConfigError: YAML 解析错误或结构错误
        """
        path = Path(path).resolve()

        # 检查文件存在
        if not path.exists():
            raise LoadError(f"Config file not found: {path}")

        # 检查是文件
        if not path.is_file():
            raise LoadError(f"Config path is not a file: {path}")

        # 读取原始内容
        try:
            content = path.read_text(encoding='utf-8')
        except UnicodeDecodeError as e:
            raise LoadError(f"File encoding error (expected UTF-8): {e}")
        except IOError as e:
            raise LoadError(f"Cannot read file: {e}")

        # 解析 YAML
        try:
            raw_data = yaml.safe_load(content)
        except yaml.YAMLError as e:
            raise ConfigError(
                f"YAML syntax error: {e}",
                location=[str(path)]
            )

        # 检查空文件
        if raw_data is None:
            raise ConfigError("Empty config file", location=[str(path)])

        # 检查是否是字典
        if not isinstance(raw_data, dict):
            raise ConfigError(
                f"Config must be a YAML object, got {type(raw_data).__name__}",
                location=[str(path)]
            )

        # 解析为 Pydantic 模型
        try:
            config = OctopusConfig.model_validate(raw_data)
        except ValidationError as e:
            # 转换 Pydantic 错误为 ConfigError
            errors = []
            for err in e.errors():
                location = list(err['loc'])
                errors.append({
                    'type': 'structure',
                    'field': '.'.join(str(x) for x in location),
                    'message': err['msg'],
                    'location': location
                })
            raise ConfigError(
                f"Config validation failed with {len(errors)} error(s)",
                location=[str(path)]
            ) from e

        return config

    @staticmethod
    def load_raw(path: Path | str) -> dict[str, Any]:
        """
        加载原始 YAML 数据（不进行模型验证）

        Args:
            path: 配置文件路径

        Returns:
            dict: 原始 YAML 数据
        """
        path = Path(path).resolve()
        content = path.read_text(encoding='utf-8')
        return yaml.safe_load(content)
```

### 3.2 错误转换工具
```python
def convert_pydantic_errors(validation_error: ValidationError) -> list[dict]:
    """
    将 Pydantic ValidationError 转换为结构化错误列表

    示例输出:
    [
        {
            "type": "structure",
            "field": "meta.workflow_id",
            "message": "Field required",
            "location": ["meta", "workflow_id"]
        }
    ]
    """
    errors = []
    for err in validation_error.errors():
        errors.append({
            "type": "structure",
            "field": ".".join(str(x) for x in err['loc']),
            "message": err['msg'],
            "location": list(err['loc'])
        })
    return errors
```

### 3.3 环境变量支持（预留接口）
```python
import os
import re

ENV_VAR_PATTERN = re.compile(r'\$\{([A-Z_][A-Z0-9_]*)\}')

def expand_env_vars(obj: Any) -> Any:
    """
    递归展开对象中的环境变量引用
    支持 ${VAR_NAME} 语法

    示例:
    "${HOME}/.octopus" -> "/Users/xxx/.octopus"
    """
    if isinstance(obj, str):
        def replacer(match: re.Match) -> str:
            var_name = match.group(1)
            return os.environ.get(var_name, match.group(0))
        return ENV_VAR_PATTERN.sub(replacer, obj)
    elif isinstance(obj, dict):
        return {k: expand_env_vars(v) for k, v in obj.items()}
    elif isinstance(obj, list):
        return [expand_env_vars(item) for item in obj]
    return obj
```

### 3.4 加载器工厂函数
```python
def load_config(path: Path | str | None = None) -> OctopusConfig:
    """
    便捷函数：加载配置

    Args:
        path: 配置文件路径，默认查找 ./octopus.yaml

    Returns:
        OctopusConfig

    Raises:
        LoadError: 加载失败
    """
    if path is None:
        path = Path("octopus.yaml")
    return ConfigLoader.load(path)
```

## 验收标准
- [ ] 能正确加载有效 YAML 文件
- [ ] 文件不存在时抛出 LoadError
- [ ] YAML 语法错误时抛出 ConfigError 并包含位置信息
- [ ] Pydantic 验证失败时提供详细错误信息
- [ ] 空文件处理正确
- [ ] 非 UTF-8 编码文件给出明确错误
- [ ] 单元测试覆盖率达到 90%

## 测试用例

```python
# tests/unit/test_loader.py

class TestConfigLoader:
    def test_load_valid_config(self, tmp_path):
        """测试加载有效配置"""
        config_file = tmp_path / "octopus.yaml"
        config_file.write_text("""
version: "2.1"
meta:
  workflow_id: "test"
  name: "Test"
runtime:
  workspace:
    root: ".octopus"
llm_profiles:
  codex:
    provider: "codex"
    mode: "cli"
tool_registry:
  builtin: {}
roles:
  - id: "test"
    name: "Test"
    type: "simple"
    llm_profile: "codex"
    system_prompt: "test"
stages:
  - id: "s1"
    name: "S1"
    role: "test"
    input:
      - type: "requirement_file"
        path: "./test.md"
    output:
      - type: "artifact"
        name: "result"
transitions:
  - from: "s1"
    to: "__END__"
""")
        config = ConfigLoader.load(config_file)
        assert config.version == "2.1"
        assert config.meta.workflow_id == "test"

    def test_load_file_not_found(self):
        """测试文件不存在"""
        with pytest.raises(LoadError) as exc_info:
            ConfigLoader.load("/nonexistent/config.yaml")
        assert "not found" in str(exc_info.value)

    def test_load_yaml_syntax_error(self, tmp_path):
        """测试 YAML 语法错误"""
        config_file = tmp_path / "bad.yaml"
        config_file.write_text("invalid: yaml: [unclosed")

        with pytest.raises(ConfigError) as exc_info:
            ConfigLoader.load(config_file)
        assert "YAML syntax error" in str(exc_info.value)

    def test_load_empty_file(self, tmp_path):
        """测试空文件"""
        config_file = tmp_path / "empty.yaml"
        config_file.write_text("")

        with pytest.raises(ConfigError) as exc_info:
            ConfigLoader.load(config_file)
        assert "Empty" in str(exc_info.value)

    def test_load_raw(self, tmp_path):
        """测试原始加载"""
        config_file = tmp_path / "test.yaml"
        config_file.write_text("key: value\nlist:\n  - a\n  - b")

        raw = ConfigLoader.load_raw(config_file)
        assert raw["key"] == "value"
        assert raw["list"] == ["a", "b"]
```

## 预计耗时
45 分钟
