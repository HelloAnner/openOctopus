# Plan 07: 模板系统和 E2E 测试

## 目标
实现内置模板系统和完整的 E2E 测试覆盖。

## 任务清单

### 7.1 模板系统 (config/templates.py)

```python
"""内置模板系统"""

from typing import Final

# 默认模板 - 可立即运行的 Hello World 配置
DEFAULT_TEMPLATE: Final[str] = '''version: "2.1"

meta:
  workflow_id: "hello-world"
  name: "Hello World"
  description: "A minimal executable OpenOctopus workflow"

runtime:
  workspace:
    root: ".octopus"

llm_profiles:
  default:
    provider: "claude_code"
    mode: "cli"

tool_registry:
  builtin: {}

roles:
  - id: "analyzer"
    name: "需求分析器"
    type: "simple"
    llm_profile: "default"
    system_prompt: |
      你是一位需求分析师。请分析给定的需求文档，输出：
      1. 核心功能点
      2. 技术要点
      3. 实现建议

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
'''

# 可用模板注册表
TEMPLATES: Final[dict[str, str]] = {
    "default": DEFAULT_TEMPLATE,
    "hello-world": DEFAULT_TEMPLATE,
}

def get_template(name: str) -> str:
    if name not in TEMPLATES:
        available = ", ".join(TEMPLATES.keys())
        raise KeyError(f"Template '{name}' not found. Available: {available}")
    return TEMPLATES[name]

def list_templates() -> list[str]:
    return list(TEMPLATES.keys())
```

### 7.2 测试配置 (tests/conftest.py)

```python
import pytest
from pathlib import Path
from openoctopus.config.models import OctopusConfig

@pytest.fixture
def minimal_config_data():
    return {
        "version": "2.1",
        "meta": {"workflow_id": "test", "name": "Test"},
        "runtime": {"workspace": {"root": ".octopus"}},
        "llm_profiles": {
            "default": {"provider": "claude_code", "mode": "cli"}
        },
        "tool_registry": {"builtin": {}},
        "roles": [
            {
                "id": "test_role",
                "name": "Test",
                "type": "simple",
                "llm_profile": "default",
                "system_prompt": "Test"
            }
        ],
        "stages": [
            {
                "id": "test_stage",
                "name": "Test",
                "role": "test_role",
                "input": [{"type": "requirement_file", "path": "./test.md"}],
                "output": [{"type": "artifact", "name": "result"}]
            }
        ],
        "transitions": [{"from": "test_stage", "to": "__END__"}]
    }

@pytest.fixture
def minimal_config(minimal_config_data):
    return OctopusConfig.model_validate(minimal_config_data)

@pytest.fixture
def mock_llm_executor(monkeypatch):
    """Mock LLM 执行"""
    def mock_execute(*args, **kwargs):
        from openoctopus.executor.role_executor import ExecutionResult
        return ExecutionResult(
            status="completed",
            output="# Test Result\n\nMock analysis output.",
            artifacts=["result"]
        )

    monkeypatch.setattr(
        "openoctopus.executor.role_executor.RoleExecutor.execute",
        mock_execute
    )
```

### 7.3 E2E 测试文件结构

```
tests/e2e/
├── __init__.py
├── test_init.py          # init 命令测试
├── test_validate.py      # validate 命令测试
├── test_run.py           # run 命令测试（核心）
├── test_status.py        # status 命令测试
├── test_list.py          # list 命令测试
├── test_version.py       # version 命令测试
└── test_full_workflow.py # 完整流程测试
```

### 7.4 核心 E2E 测试

```python
# tests/e2e/test_run.py
import pytest
import yaml
from typer.testing import CliRunner
from openoctopus.cli.main import app

runner = CliRunner()

class TestRunCommand:
    def test_run_single_stage(self, tmp_path, mock_llm_executor):
        """测试执行单 stage 工作流"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            # 创建配置
            config = Path("octopus.yaml")
            config.write_text(yaml.dump({
                "version": "2.1",
                "meta": {"workflow_id": "test", "name": "Test"},
                "runtime": {"workspace": {"root": ".octopus"}},
                "llm_profiles": {
                    "default": {"provider": "claude_code", "mode": "cli"}
                },
                "tool_registry": {"builtin": {}},
                "roles": [
                    {
                        "id": "analyzer",
                        "name": "Analyzer",
                        "type": "simple",
                        "llm_profile": "default",
                        "system_prompt": "Analyze."
                    }
                ],
                "stages": [
                    {
                        "id": "analyze",
                        "name": "Analyze",
                        "role": "analyzer",
                        "input": [{"type": "requirement_file", "path": "./req.md"}],
                        "output": [{"type": "artifact", "name": "report"}]
                    }
                ],
                "transitions": [{"from": "analyze", "to": "__END__"}]
            }))

            # 创建需求文件
            Path("req.md").write_text("# Test requirement")

            # 执行 run
            result = runner.invoke(app, ["run", "-c", "octopus.yaml", "-r", "req.md"])

            assert result.exit_code == 0
            assert "completed successfully" in result.output
            assert "sess_" in result.output

            # 验证产物生成
            sessions_dir = tmp_path / ".octopus" / "sessions"
            assert sessions_dir.exists()

            session_dirs = list(sessions_dir.iterdir())
            assert len(session_dirs) == 1

            artifact_file = session_dirs[0] / "artifacts" / "report.md"
            assert artifact_file.exists()

    def test_run_invalid_config(self, tmp_path):
        """测试无效配置不执行"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            Path("bad.yaml").write_text("invalid: yaml: [")
            Path("req.md").write_text("test")

            result = runner.invoke(app, ["run", "-c", "bad.yaml", "-r", "req.md"])

            assert result.exit_code == 2
            assert "not found" in result.output.lower() or "syntax" in result.output.lower()
```

```python
# tests/e2e/test_full_workflow.py
class TestFullWorkflow:
    def test_complete_user_journey(self, tmp_path, mock_llm_executor):
        """测试完整用户旅程"""
        with runner.isolated_filesystem(temp_dir=tmp_path):
            # 1. init
            result = runner.invoke(app, ["init"])
            assert result.exit_code == 0

            # 2. 创建需求
            Path("requirement.md").write_text("# Feature\n\n实现登录功能")

            # 3. validate
            result = runner.invoke(app, ["validate", "-c", "octopus.yaml"])
            assert result.exit_code == 0

            # 4. run
            result = runner.invoke(app, ["run", "-c", "octopus.yaml", "-r", "requirement.md"])
            assert result.exit_code == 0
            assert "sess_" in result.output

            # 提取 session_id
            import re
            match = re.search(r'sess_\w+', result.output)
            assert match
            session_id = match.group()

            # 5. status
            result = runner.invoke(app, ["status", "-s", session_id])
            assert result.exit_code == 0
            assert "COMPLETED" in result.output

            # 6. list
            result = runner.invoke(app, ["list"])
            assert result.exit_code == 0
            assert session_id in result.output
```

### 7.5 Makefile

```makefile
.PHONY: install test test-e2e test-unit lint format

install:
	uv sync

test:
	uv run pytest tests/ -v

test-e2e:
	uv run pytest tests/e2e/ -v

test-unit:
	uv run pytest tests/unit/ -v

lint:
	uv run ruff check openoctopus tests

format:
	uv run ruff format openoctopus tests

coverage:
	uv run pytest tests/ --cov=openoctopus --cov-report=html
```

### 7.6 CI 配置

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

      - uses: astral-sh/setup-uv@v1

      - run: uv sync

      - run: uv run ruff check openoctopus tests

      - run: uv run pytest tests/ -v --cov=openoctopus --cov-report=xml

      - uses: codecov/codecov-action@v3
        with:
          files: ./coverage.xml
```

## 验收标准
- [ ] 模板系统可正常工作
- [ ] E2E 测试覆盖 init → validate → run → status → list 完整流程
- [ ] 单元测试覆盖率 ≥ 85%
- [ ] CI 配置可正常运行
- [ ] 代码格式化通过 Ruff
- [ ] 无 lint 错误

## 预计耗时
60 分钟
