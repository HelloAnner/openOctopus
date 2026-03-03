# Plan 05: CLI 命令实现

## 目标
实现完整的 CLI 命令，包括 init、validate、run（占位）和 version。

## 任务清单

### 5.1 CLI 工具函数 (cli/utils/console.py)

```python
from rich.console import Console
from rich.panel import Panel
from rich.text import Text
from rich.table import Table

# 全局 console 实例
console = Console()

def print_success(message: str) -> None:
    """打印成功信息"""
    console.print(f"[green]✓[/green] {message}")

def print_error(message: str) -> None:
    """打印错误信息"""
    console.print(f"[red]✗[/red] {message}")

def print_warning(message: str) -> None:
    """打印警告信息"""
    console.print(f"[yellow]⚠[/yellow] {message}")

def print_info(message: str) -> None:
    """打印普通信息"""
    console.print(f"[blue]ℹ[/blue] {message}")

def print_header(title: str) -> None:
    """打印标题"""
    console.print(Panel(title, style="bold blue"))
```

### 5.2 格式化工具 (cli/utils/formatters.py)

```python
from rich.table import Table
from rich.json import JSON
import json
import yaml

from ...config.models import ValidationResult

def format_validation_table(result: ValidationResult) -> Table:
    """格式化校验结果为表格"""
    table = Table(title="Validation Result")
    table.add_column("Type", style="cyan")
    table.add_column("Field", style="magenta")
    table.add_column("Message", style="white")

    for error in result.errors:
        table.add_row(
            f"[red]{error.type}[/red]",
            error.field,
            error.message
        )

    for warning in result.warnings:
        table.add_row(
            f"[yellow]{warning.type}[/yellow]",
            warning.field,
            warning.message
        )

    return table

def format_validation_json(result: ValidationResult) -> str:
    """格式化校验结果为 JSON 字符串"""
    return json.dumps(result.model_dump(), indent=2)

def format_validation_yaml(result: ValidationResult) -> str:
    """格式化校验结果为 YAML 字符串"""
    return yaml.dump(result.model_dump(), default_flow_style=False)
```

### 5.3 Version 命令 (cli/commands/version.py)

```python
import typer
from ..utils.console import console
from ...core.constants import VERSION

app = typer.Typer()

@app.callback(invoke_without_command=True)
def version():
    """显示版本信息"""
    console.print(f"OpenOctopus [bold green]{VERSION}[/bold green]")

# 作为 callback 和命令都支持
@app.command()
def show():
    """显示详细版本信息"""
    console.print(f"OpenOctopus [bold green]{VERSION}[/bold green]")
    console.print("A CLI workflow orchestration tool for AI-assisted development.")
```

### 5.4 Init 命令 (cli/commands/init.py)

```python
from pathlib import Path
import typer

from ..utils.console import print_success, print_error, print_warning, print_header
from ...config.templates import get_template, list_templates
from ...core.exceptions import InitError

app = typer.Typer()

@app.command()
def init(
    template: str = typer.Option("default", "--template", "-t", help="Template to use"),
    force: bool = typer.Option(False, "--force", "-f", help="Force overwrite existing config"),
    path: Path = typer.Option(Path("."), "--path", "-p", help="Target directory path"),
):
    """
    Initialize a new OpenOctopus project
    """
    print_header("OpenOctopus Init")

    try:
        # 检查模板存在
        if template not in list_templates():
            print_error(f"Template '{template}' not found. Available: {', '.join(list_templates())}")
            raise typer.Exit(4)

        # 确保目标路径存在
        target_path = path.resolve()
        target_path.mkdir(parents=True, exist_ok=True)

        # 检查 .octopus 目录
        octopus_dir = target_path / ".octopus"
        config_file = target_path / "octopus.yaml"

        if octopus_dir.exists() and not force:
            print_error(f"Project already initialized at {target_path}")
            print_info("Use --force to overwrite existing config")
            raise typer.Exit(4)

        # 创建目录结构
        _create_directory_structure(octopus_dir)

        # 生成配置文件
        template_content = get_template(template)
        if config_file.exists() and not force:
            print_warning(f"Config file exists, skipping (use --force to overwrite)")
        else:
            config_file.write_text(template_content, encoding='utf-8')
            print_success(f"Created config file: {config_file}")

        print_success(f"Initialized OpenOctopus project in {target_path}")
        print_info("Next steps:")
        print_info("  1. Edit octopus.yaml to configure your workflow")
        print_info("  2. Run 'openoctopus validate --config octopus.yaml' to verify")

    except InitError as e:
        print_error(str(e))
        raise typer.Exit(4)
    except Exception as e:
        print_error(f"Unexpected error: {e}")
        raise typer.Exit(1)

def _create_directory_structure(octopus_dir: Path) -> None:
    """创建 .octopus 目录结构"""
    dirs = [
        octopus_dir / "sessions",
        octopus_dir / "artifacts",
        octopus_dir / "logs",
        octopus_dir / "cache",
    ]

    for d in dirs:
        d.mkdir(parents=True, exist_ok=True)
        print_success(f"Created directory: {d}")
```

### 5.5 Validate 命令 (cli/commands/validate.py)

```python
from pathlib import Path
import typer

from ..utils.console import print_success, print_error, console
from ..utils.formatters import format_validation_table, format_validation_json, format_validation_yaml
from ...config.loader import ConfigLoader
from ...config.validator import ConfigValidator
from ...core.exceptions import ConfigError, LoadError

app = typer.Typer()

@app.command()
def validate(
    config: Path = typer.Option(..., "--config", "-c", help="Path to config file"),
    verbose: bool = typer.Option(False, "--verbose", "-v", help="Verbose output"),
    format: str = typer.Option("table", "--format", help="Output format: table, json, yaml"),
):
    """
    Validate OpenOctopus configuration file
    """
    # 验证 format 参数
    if format not in ("table", "json", "yaml"):
        print_error(f"Invalid format: {format}. Use 'table', 'json', or 'yaml'")
        raise typer.Exit(1)

    try:
        if verbose:
            console.print(f"[blue]Loading config from {config}...[/blue]")

        # 加载配置
        raw_data = ConfigLoader.load_raw(config)
        octopus_config = ConfigLoader.load(config)

        if verbose:
            console.print("[blue]Running validations...[/blue]")

        # 校验配置
        validator = ConfigValidator(octopus_config, raw_data)
        result = validator.validate()

        # 输出结果
        if format == "table":
            if result.valid and not result.warnings:
                print_success("Configuration is valid")
            elif result.valid:
                print_success("Configuration is valid (with warnings)")
                console.print(format_validation_table(result))
            else:
                print_error(f"Configuration validation failed ({len(result.errors)} error(s))")
                console.print(format_validation_table(result))
        elif format == "json":
            console.print(format_validation_json(result))
        else:  # yaml
            console.print(format_validation_yaml(result))

        # 返回适当的退出码
        if not result.valid:
            raise typer.Exit(3)

    except LoadError as e:
        print_error(str(e))
        raise typer.Exit(2)
    except ConfigError as e:
        print_error(str(e))
        if e.location:
            console.print(f"[dim]Location: {' -> '.join(e.location)}[/dim]")
        raise typer.Exit(2)
    except Exception as e:
        print_error(f"Unexpected error: {e}")
        raise typer.Exit(1)
```

### 5.6 Run 命令占位 (cli/commands/run.py)

```python
import typer
from pathlib import Path

from ..utils.console import print_info, print_warning

app = typer.Typer()

@app.command()
def run(
    config: Path = typer.Option(Path("octopus.yaml"), "--config", "-c", help="Path to config file"),
    requirement: Path = typer.Option(None, "--requirement", "-r", help="Path to requirement file"),
    session: str = typer.Option(None, "--session", "-s", help="Session ID to resume"),
    dry_run: bool = typer.Option(False, "--dry-run", help="Show what would be executed"),
):
    """
    Run OpenOctopus workflow (placeholder)
    """
    print_warning("Run command is not implemented yet. Coming in Phase 2.")
    print_info("Use 'openoctopus validate --config <file>' to check your configuration.")
```

### 5.7 主 CLI 入口 (cli/main.py)

```python
import typer

from .commands import init, validate, run, version
from ..core.constants import VERSION

app = typer.Typer(
    name="openoctopus",
    help="OpenOctopus - AI-assisted development workflow orchestration tool",
    no_args_is_help=True,
)

# 注册子命令
app.add_typer(init.app, name="init", help="Initialize a new project")
app.add_typer(validate.app, name="validate", help="Validate configuration")
app.add_typer(run.app, name="run", help="Run workflow (placeholder)")
app.add_typer(version.app, name="version", help="Show version")

@app.callback()
def main(
    verbose: bool = typer.Option(False, "--verbose", "-v", help="Enable verbose output"),
):
    """OpenOctopus CLI"""
    pass

# 支持 --version 全局选项
@app.callback(invoke_without_command=True)
def show_version(
    version: bool = typer.Option(False, "--version", help="Show version and exit"),
):
    if version:
        print(f"OpenOctopus {VERSION}")
        raise typer.Exit()

def main_entry():
    """入口函数"""
    app()

if __name__ == "__main__":
    main_entry()
```

### 5.8 包入口 (cli/__init__.py 和 cli/commands/__init__.py)

```python
# cli/__init__.py
from .main import app, main_entry

__all__ = ["app", "main_entry"]
```

```python
# cli/commands/__init__.py
from . import init, validate, run, version

__all__ = ["init", "validate", "run", "version"]
```

### 5.9 根包入口 (__main__.py 和 __init__.py)

```python
# openoctopus/__init__.py
from .core.constants import VERSION

__version__ = VERSION
__all__ = ["__version__"]
```

```python
# openoctopus/__main__.py
from .cli import main_entry

if __name__ == "__main__":
    main_entry()
```

## 验收标准
- [ ] `openoctopus --version` 输出版本号
- [ ] `openoctopus init` 创建正确目录结构
- [ ] `openoctopus validate` 正确校验配置文件
- [ ] 错误时返回正确的退出码
- [ ] Rich 美化输出正常
- [ ] 帮助信息完整且格式美观
- [ ] E2E 测试全部通过

## 测试用例

```python
# tests/e2e/test_cli.py
from typer.testing import CliRunner
from openoctopus.cli.main import app

runner = CliRunner()

class TestCLI:
    def test_version(self):
        result = runner.invoke(app, ["--version"])
        assert result.exit_code == 0
        assert "OpenOctopus" in result.output

    def test_init_default(self, tmp_path):
        with runner.isolated_filesystem(temp_dir=tmp_path):
            result = runner.invoke(app, ["init"])
            assert result.exit_code == 0
            assert (tmp_path / ".octopus").exists()
            assert (tmp_path / "octopus.yaml").exists()

    def test_validate_success(self, tmp_path):
        # 创建有效配置
        with runner.isolated_filesystem(temp_dir=tmp_path):
            # ... 创建配置文件
            result = runner.invoke(app, ["validate", "-c", "octopus.yaml"])
            assert result.exit_code == 0
            assert "valid" in result.output.lower()
```

## 预计耗时
75 分钟
