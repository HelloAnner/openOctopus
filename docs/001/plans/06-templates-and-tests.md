# Plan 06: 模板系统和 E2E 测试

## 目标
实现内置模板系统和完整的 E2E 测试覆盖。

## 任务清单

### 6.1 模板系统 (config/templates.py)

```python
"""
内置模板系统
"""

from typing import Final

# 默认模板 - 最小可用配置
DEFAULT_TEMPLATE: Final[str] = '''version: "2.1"

meta:
  workflow_id: "example-workflow"
  name: "Example Workflow"
  description: "A minimal OpenOctopus workflow configuration"

runtime:
  workspace:
    root: ".octopus"
    sessions_dir: ".octopus/sessions"
    artifacts_dir: ".octopus/artifacts"
    logs_dir: ".octopus/logs"
  tmux:
    enabled: false

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"
    max_tokens: 4096
    temperature: 0.1

tool_registry:
  builtin:
    file_read:
      module: "openoctopus.tools.file"
      class: "FileReadTool"
    file_write:
      module: "openoctopus.tools.file"
      class: "FileWriteTool"

roles:
  - id: "review_agent"
    name: "代码审查员"
    type: "simple"
    llm_profile: "codex_cli"
    system_prompt: |
      你是一位严格的代码审查员，负责检查代码质量和潜在问题。
    tools:
      - "file_read"
      - "file_write"

stages:
  - id: "code_review"
    name: "代码审查"
    role: "review_agent"
    input:
      - type: "requirement_file"
        path: "./review.diff"
    output:
      - type: "artifact"
        name: "review_result"

transitions:
  - from: "code_review"
    to: "__END__"
'''

# Feature Dev 模板 - 与 default 相同，预留扩展
FEATURE_DEV_TEMPLATE: Final[str] = DEFAULT_TEMPLATE

# Bug Fix 模板 - 预留
BUG_FIX_TEMPLATE: Final[str] = DEFAULT_TEMPLATE

# 模板注册表
TEMPLATES: Final[dict[str, str]] = {
    "default": DEFAULT_TEMPLATE,
    "feature_dev": FEATURE_DEV_TEMPLATE,
    "bug_fix": BUG_FIX_TEMPLATE,
}

def get_template(name: str) -> str:
    """
    获取模板内容

    Args:
        name: 模板名称

    Returns:
        模板 YAML 字符串

    Raises:
        KeyError: 模板不存在
    """
    if name not in TEMPLATES:
        available = ", ".join(TEMPLATES.keys())
        raise KeyError(f"Template '{name}' not found. Available: {available}")
    return TEMPLATES[name]

def list_templates() -> list[str]:
    """返回所有可用模板名称列表"""
    return list(TEMPLATES.keys())

def is_valid_template(name: str) -> bool:
    """检查模板名称是否有效"""
    return name in TEMPLATES
```

### 6.2 测试配置 (tests/conftest.py)

```python
"""
Pytest 配置和 fixtures
"""

import pytest
from pathlib import Path

from openoctopus.config.models import OctopusConfig


@pytest.fixture
def minimal_config_data():
    """最小配置数据字典"""
    return {
        "version": "2.1",
        "meta": {
            "workflow_id": "test-workflow",
            "name": "Test Workflow"
        },
        "runtime": {
            "workspace": {
                "root": ".octopus",
                "sessions_dir": ".octopus/sessions",
                "artifacts_dir": ".octopus/artifacts",
                "logs_dir": ".octopus/logs"
            }
        },
        "llm_profiles": {
            "codex": {
                "provider": "codex",
                "mode": "cli"
            }
        },
        "tool_registry": {
            "builtin": {
                "file_read": {
                    "module": "openoctopus.tools.file",
                    "class": "FileReadTool"
                }
            }
        },
        "roles": [
            {
                "id": "test_role",
                "name": "Test Role",
                "type": "simple",
                "llm_profile": "codex",
                "system_prompt": "Test prompt",
                "tools": ["file_read"]
            }
        ],
        "stages": [
            {
                "id": "test_stage",
                "name": "Test Stage",
                "role": "test_role",
                "input": [
                    {
                        "type": "requirement_file",
                        "path": "./test.md"
                    }
                ],
                "output": [
                    {
                        "type": "artifact",
                        "name": "result"
                    }
                ]
            }
        ],
        "transitions": [
            {
                "from": "test_stage",
                "to": "__END__"
            }
        ]
    }


@pytest.fixture
def minimal_config(minimal_config_data):
    """最小配置对象"""
    return OctopusConfig.model_validate(minimal_config_data)


@pytest.fixture
def tmp_config_file(tmp_path, minimal_config_data):
    """创建临时配置文件"""
    import yaml

    config_file = tmp_path / "octopus.yaml"
    config_file.write_text(yaml.dump(minimal_config_data))
    return config_file
```

### 6.3 E2E Init 测试 (tests/e2e/test_init.py)

```python
"""
E2E 测试: init 命令
"""

import pytest
from typer.testing import CliRunner
from pathlib import Path

from openoctopus.cli.main import app

runner = CliRunner()


class TestInitCommand:
    def test_init_default_template(self, tmp_path):
        """测试场景 1: 使用默认模板初始化"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            result = runner.invoke(app, ["init"])

            assert result.exit_code == 0
            assert ".octopus/" in result.output
            assert "octopus.yaml" in result.output

            # 验证目录结构
            assert (tmp_path / ".octopus").exists()
            assert (tmp_path / ".octopus" / "sessions").exists()
            assert (tmp_path / ".octopus" / "artifacts").exists()
            assert (tmp_path / ".octopus" / "logs").exists()
            assert (tmp_path / ".octopus" / "cache").exists()

            # 验证配置文件
            config_file = tmp_path / "octopus.yaml"
            assert config_file.exists()

            # 验证 YAML 可解析
            import yaml
            data = yaml.safe_load(config_file.read_text())
            assert data["version"] == "2.1"
            assert "meta" in data
            assert "roles" in data

    def test_init_with_path(self, tmp_path):
        """测试场景 2: 指定路径初始化"""
        target = tmp_path / "subdir"

        result = runner.invoke(app, ["init", "--path", str(target)])

        assert result.exit_code == 0
        assert (target / ".octopus").exists()
        assert (target / "octopus.yaml").exists()

    def test_init_existing_project_no_force(self, tmp_path):
        """测试场景 3: 目录已存在，不使用 force"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            # 先初始化一次
            (tmp_path / ".octopus").mkdir()

            result = runner.invoke(app, ["init"])

            assert result.exit_code == 4
            assert "already initialized" in result.output.lower()

    def test_init_force(self, tmp_path):
        """测试场景 4: 使用 force 覆盖"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            # 先初始化
            runner.invoke(app, ["init"])

            # 修改配置文件
            config = tmp_path / "octopus.yaml"
            original_content = config.read_text()
            config.write_text(original_content + "\n# MODIFIED")

            # 强制重新初始化
            result = runner.invoke(app, ["init", "--force"])

            assert result.exit_code == 0
            assert "force" in result.output.lower() or "Force" in result.output

    def test_init_invalid_template(self, tmp_path):
        """测试: 无效模板名称"""
        result = runner.invoke(app, ["init", "--template", "nonexistent"])

        assert result.exit_code == 4
        assert "not found" in result.output.lower()
```

### 6.4 E2E Validate 测试 (tests/e2e/test_validate.py)

```python
"""
E2E 测试: validate 命令
"""

import pytest
import yaml
from typer.testing import CliRunner
from pathlib import Path

from openoctopus.cli.main import app

runner = CliRunner()


class TestValidateCommand:
    def test_validate_success(self, tmp_path, minimal_config_data):
        """测试场景 5: 校验通过"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            config_file = Path("octopus.yaml")
            config_file.write_text(yaml.dump(minimal_config_data))

            result = runner.invoke(app, ["validate", "--config", str(config_file)])

            assert result.exit_code == 0
            assert "valid" in result.output.lower()

    def test_validate_yaml_syntax_error(self, tmp_path):
        """测试场景 6: YAML 语法错误"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            config_file = Path("invalid.yaml")
            config_file.write_text("invalid: yaml: [unclosed")

            result = runner.invoke(app, ["validate", "--config", str(config_file)])

            assert result.exit_code == 2
            assert "yaml" in result.output.lower() or "syntax" in result.output.lower()

    def test_validate_missing_required_field(self, tmp_path):
        """测试场景 7: 必填字段缺失"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            config_file = Path("incomplete.yaml")
            config_file.write_text(yaml.dump({
                "version": "2.1",
                "meta": {
                    "name": "Missing workflow_id"  # 缺少 workflow_id
                }
            }))

            result = runner.invoke(app, ["validate", "--config", str(config_file)])

            assert result.exit_code == 2
            assert "workflow_id" in result.output.lower() or "required" in result.output.lower()

    def test_validate_invalid_role_reference(self, tmp_path):
        """测试场景 8: Role 引用错误"""
        config_data = {
            "version": "2.1",
            "meta": {"workflow_id": "test", "name": "Test"},
            "runtime": {"workspace": {"root": ".octopus"}},
            "llm_profiles": {"codex": {"provider": "codex", "mode": "cli"}},
            "tool_registry": {"builtin": {}},
            "roles": [],  # 没有定义任何 role
            "stages": [
                {
                    "id": "stage1",
                    "name": "Stage 1",
                    "role": "nonexistent_role",  # 引用不存在的 role
                    "input": [{"type": "requirement_file", "path": "./test.md"}],
                    "output": [{"type": "artifact", "name": "result"}]
                }
            ],
            "transitions": []
        }

        with runner.isolated_filesystem(temp_dir=tmp_path):
            config_file = Path("bad_ref.yaml")
            config_file.write_text(yaml.dump(config_data))

            result = runner.invoke(app, ["validate", "--config", str(config_file)])

            assert result.exit_code == 3
            assert "nonexistent" in result.output.lower()

    def test_validate_loop_detection(self, tmp_path):
        """测试场景 10: 环路检测"""
        config_data = {
            "version": "2.1",
            "meta": {"workflow_id": "test", "name": "Test"},
            "runtime": {"workspace": {"root": ".octopus"}},
            "llm_profiles": {"codex": {"provider": "codex", "mode": "cli"}},
            "tool_registry": {"builtin": {}},
            "roles": [
                {
                    "id": "r1", "name": "R1", "type": "simple",
                    "llm_profile": "codex", "system_prompt": "test"
                }
            ],
            "stages": [
                {"id": "a", "name": "A", "role": "r1", "input": [], "output": []},
                {"id": "b", "name": "B", "role": "r1", "input": [], "output": []}
            ],
            "transitions": [
                {"from": "a", "to": "b"},
                {"from": "b", "to": "a"}  # 形成环路
            ]
        }

        with runner.isolated_filesystem(temp_dir=tmp_path):
            config_file = Path("loop.yaml")
            config_file.write_text(yaml.dump(config_data))

            result = runner.invoke(app, ["validate", "--config", str(config_file)])

            assert result.exit_code == 3
            assert "cycle" in result.output.lower() or "loop" in result.output.lower()

    def test_validate_output_json(self, tmp_path, minimal_config_data):
        """测试场景 12: JSON 格式输出"""
        import json

        with runner.isolated_filesystem(temp_dir=tmp_path):
            config_file = Path("test.yaml")
            config_file.write_text(yaml.dump(minimal_config_data))

            result = runner.invoke(app, ["validate", "--config", str(config_file), "--format", "json"])

            assert result.exit_code == 0
            # 验证是有效的 JSON
            data = json.loads(result.output)
            assert "valid" in data
            assert "errors" in data

    def test_validate_verbose(self, tmp_path, minimal_config_data):
        """测试: verbose 模式"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            config_file = Path("test.yaml")
            config_file.write_text(yaml.dump(minimal_config_data))

            result = runner.invoke(app, ["validate", "--config", str(config_file), "--verbose"])

            assert result.exit_code == 0
            assert "loading" in result.output.lower() or "running" in result.output.lower()
```

### 6.5 E2E 其他命令测试

```python
# tests/e2e/test_version.py
from typer.testing import CliRunner
from openoctopus.cli.main import app

runner = CliRunner()


class TestVersionCommand:
    def test_version_command(self):
        """测试场景 13: Version 命令"""
        result = runner.invoke(app, ["version"])
        assert result.exit_code == 0
        assert "OpenOctopus" in result.output

    def test_version_flag(self):
        """测试 --version 全局选项"""
        result = runner.invoke(app, ["--version"])
        assert result.exit_code == 0
        assert "OpenOctopus" in result.output
```

```python
# tests/e2e/test_run.py
from typer.testing import CliRunner
from openoctopus.cli.main import app

runner = CliRunner()


class TestRunCommand:
    def test_run_placeholder(self):
        """测试场景 14: Run 命令占位"""
        result = runner.invoke(app, ["run"])
        assert result.exit_code == 0
        assert "not implemented" in result.output.lower() or "placeholder" in result.output.lower()
```

```python
# tests/e2e/test_help.py
import pytest
from typer.testing import CliRunner
from openoctopus.cli.main import app

runner = CliRunner()


class TestHelpCommand:
    @pytest.mark.parametrize("args", [
        [],
        ["--help"],
        ["init", "--help"],
        ["validate", "--help"],
        ["run", "--help"],
        ["version", "--help"],
    ])
    def test_help_output(self, args):
        """测试场景 15: 所有命令的帮助信息"""
        result = runner.invoke(app, args)
        assert result.exit_code == 0
        assert "Usage:" in result.output or "usage:" in result.output.lower()
```

### 6.6 单元测试

```python
# tests/unit/test_models.py
import pytest
from openoctopus.config.models import OctopusConfig


class TestOctopusConfig:
    def test_minimal_config_creation(self, minimal_config_data):
        """测试创建最小配置"""
        config = OctopusConfig.model_validate(minimal_config_data)
        assert config.version == "2.1"
        assert config.meta.workflow_id == "test-workflow"

    def test_config_serialization(self, minimal_config):
        """测试配置序列化"""
        data = minimal_config.model_dump(by_alias=True)
        assert data["version"] == "2.1"
```

```python
# tests/unit/test_loader.py
import pytest
import yaml
from pathlib import Path

from openoctopus.config.loader import ConfigLoader
from openoctopus.core.exceptions import LoadError, ConfigError


class TestConfigLoader:
    def test_load_valid_config(self, tmp_path, minimal_config_data):
        """测试加载有效配置"""
        config_file = tmp_path / "test.yaml"
        config_file.write_text(yaml.dump(minimal_config_data))

        config = ConfigLoader.load(config_file)
        assert config.version == "2.1"

    def test_load_file_not_found(self):
        """测试文件不存在"""
        with pytest.raises(LoadError) as exc_info:
            ConfigLoader.load("/nonexistent/file.yaml")
        assert "not found" in str(exc_info.value)

    def test_load_yaml_syntax_error(self, tmp_path):
        """测试 YAML 语法错误"""
        config_file = tmp_path / "bad.yaml"
        config_file.write_text("invalid: yaml: [")

        with pytest.raises(ConfigError) as exc_info:
            ConfigLoader.load(config_file)
        assert "syntax" in str(exc_info.value).lower()
```

### 6.7 Makefile 和 CI 配置

```makefile
# Makefile
.PHONY: install test test-e2e test-unit lint format clean

install:
	uv sync

test:
	uv run pytest -v

test-e2e:
	uv run pytest tests/e2e/ -v

test-unit:
	uv run pytest tests/unit/ -v

lint:
	uv run ruff check openoctopus tests

format:
	uv run ruff format openoctopus tests

clean:
	rm -rf .pytest_cache
	rm -rf htmlcov
	rm -rf .ruff_cache
```

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        python-version: ["3.10", "3.11", "3.12"]

    steps:
      - uses: actions/checkout@v4

      - name: Setup UV
        uses: astral-sh/setup-uv@v1

      - name: Setup Python
        run: uv python install ${{ matrix.python-version }}

      - name: Install dependencies
        run: uv sync

      - name: Run linting
        run: uv run ruff check openoctopus tests

      - name: Run tests
        run: uv run pytest tests/ -v --cov=openoctopus --cov-report=xml

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.xml
```

## 验收标准
- [ ] 模板系统可正常工作
- [ ] E2E 测试覆盖所有 15 个场景
- [ ] 单元测试覆盖率达到 85%+
- [ ] CI 配置可正常运行
- [ ] 代码格式化通过 Ruff
- [ ] 无 lint 错误

## 预计耗时
60 分钟
