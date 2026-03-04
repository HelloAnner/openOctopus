# Plan 06: CLI 命令实现（含实际执行）

## 目标
实现完整的 CLI 命令，包括 init、validate、run（实际执行）、status、list。

## 任务清单

### 6.1 CLI 工具函数 (cli/utils/console.py)

```python
from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich.progress import Progress, SpinnerColumn, TextColumn

console = Console()

def print_success(message: str) -> None:
    console.print(f"[green]✓[/green] {message}")

def print_error(message: str) -> None:
    console.print(f"[red]✗[/red] {message}")

def print_warning(message: str) -> None:
    console.print(f"[yellow]⚠[/yellow] {message}")

def print_info(message: str) -> None:
    console.print(f"[blue]ℹ[/blue] {message}")

def print_header(title: str) -> None:
    console.print(Panel(title, style="bold blue"))

def create_progress() -> Progress:
    """创建进度条"""
    return Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        console=console
    )
```

### 6.2 Version 命令 (cli/commands/version.py)

```python
import typer
from ..utils.console import console
from ...core.constants import VERSION

app = typer.Typer()

@app.command()
def version():
    """显示版本信息"""
    console.print(f"OpenOctopus [bold green]{VERSION}[/bold green]")
```

### 6.3 Init 命令 (cli/commands/init.py)

保持不变，参见 Plan 05。

### 6.4 Validate 命令 (cli/commands/validate.py)

保持不变，参见 Plan 05。

### 6.5 Run 命令（实际执行）(cli/commands/run.py) ⭐ 核心

```python
from pathlib import Path
import typer

from ..utils.console import (
    print_success, print_error, print_info, print_header,
    create_progress, console
)
from ...config.loader import ConfigLoader
from ...config.validator import ConfigValidator
from ...executor.session import SessionManager
from ...executor.runner import WorkflowRunner
from ...core.exceptions import ConfigError, LoadError

app = typer.Typer()

@app.command()
def run(
    config: Path = typer.Option(Path("octopus.yaml"), "--config", "-c", help="Config file path"),
    requirement: Path = typer.Option(..., "--requirement", "-r", help="Requirement file path"),
    session_id: str = typer.Option(None, "--session", "-s", help="Resume existing session"),
    verbose: bool = typer.Option(False, "--verbose", "-v", help="Verbose output"),
):
    """
    Run OpenOctopus workflow
    """
    print_header("OpenOctopus Runner")

    try:
        # 1. 加载配置
        if verbose:
            console.print("[blue]Loading configuration...[/blue]")

        octopus_config = ConfigLoader.load(config)
        print_success(f"Loaded config: {config}")

        # 2. 校验配置
        if verbose:
            console.print("[blue]Validating configuration...[/blue]")

        validator = ConfigValidator(octopus_config)
        validation = validator.validate()

        if not validation.valid:
            print_error("Configuration validation failed")
            for error in validation.errors:
                console.print(f"  [red]• {error.field}: {error.message}[/red]")
            raise typer.Exit(3)

        print_success("Configuration is valid")

        # 3. 检查需求文件
        if not requirement.exists():
            print_error(f"Requirement file not found: {requirement}")
            raise typer.Exit(2)

        # 4. 创建 SessionManager
        session_manager = SessionManager(octopus_config.runtime.workspace.root)

        # 5. 创建/加载 Session
        if session_id:
            session = session_manager.load_session(session_id)
            if not session:
                print_error(f"Session '{session_id}' not found")
                raise typer.Exit(6)
            print_info(f"Resuming session: {session_id}")
        else:
            session = session_manager.create_session(octopus_config, config)
            print_success(f"Created session: {session.session_id}")

        # 6. 执行工作流
        console.print()
        console.print(f"Executing Workflow: {octopus_config.meta.workflow_id}")
        console.print("─" * 50)

        runner = WorkflowRunner(octopus_config, session_manager)

        with create_progress() as progress:
            task = progress.add_task(
                f"[cyan]Running {len(octopus_config.stages)} stage(s)...",
                total=len(octopus_config.stages)
            )

            # 简化：直接执行，progress 用于显示
            try:
                session = runner.run(requirement, session)
                progress.update(task, completed=len(octopus_config.stages))
            except Exception as e:
                progress.update(task, completed=len(octopus_config.stages))
                raise

        # 7. 输出结果
        console.print()
        console.print("[green]═══════════════════════════════════════════════════[/green]")
        console.print("[green bold]Workflow completed successfully![/green bold]")
        console.print("[green]═══════════════════════════════════════════════════[/green]")
        console.print()
        console.print(f"Session ID: [cyan]{session.session_id}[/cyan]")
        console.print()

        if session.artifacts:
            console.print("Artifacts:")
            for artifact in session.artifacts:
                console.print(f"  • [cyan]{artifact.path}[/cyan]")
            console.print()

        console.print(f"View details: [bold]openoctopus status --session {session.session_id}[/bold]")

    except LoadError as e:
        print_error(str(e))
        raise typer.Exit(2)
    except ConfigError as e:
        print_error(str(e))
        raise typer.Exit(3)
    except RuntimeError as e:
        print_error(f"Execution failed: {e}")
        raise typer.Exit(5)
    except Exception as e:
        print_error(f"Unexpected error: {e}")
        raise typer.Exit(1)
```

### 6.6 Status 命令 (cli/commands/status.py)

```python
from pathlib import Path
import typer

from ..utils.console import console, print_error, print_success
from ...executor.session import SessionManager

app = typer.Typer()

@app.command()
def status(
    session: str = typer.Option(..., "--session", "-s", help="Session ID"),
    workspace: Path = typer.Option(Path(".octopus"), "--workspace", "-w", help="Workspace directory"),
):
    """
    Show session status
    """
    session_manager = SessionManager(workspace)
    session_obj = session_manager.load_session(session)

    if not session_obj:
        print_error(f"Session '{session}' not found")
        raise typer.Exit(6)

    # 输出状态
    status_color = {
        "COMPLETED": "green",
        "RUNNING": "yellow",
        "FAILED": "red",
        "CREATED": "blue"
    }.get(session_obj.status, "white")

    console.print(f"Session: [cyan]{session_obj.session_id}[/cyan]")
    console.print(f"Workflow: {session_obj.workflow_id}")
    console.print(f"Status: [{status_color}]{session_obj.status}[/{status_color}]")
    console.print(f"Created: {session_obj.created_at}")
    console.print()

    if session_obj.stages_status:
        console.print("Stages:")
        for stage_id, stage_exec in session_obj.stages_status.items():
            stage_color = {
                "COMPLETED": "green",
                "RUNNING": "yellow",
                "FAILED": "red"
            }.get(stage_exec.status, "white")
            console.print(f"  [{'✓' if stage_exec.status == 'COMPLETED' else '○'}] [{stage_color}]{stage_id}[/{stage_color}]")

    console.print()

    if session_obj.artifacts:
        console.print("Artifacts:")
        for artifact in session_obj.artifacts:
            console.print(f"  • [cyan]{artifact.path}[/cyan]")
```

### 6.7 List 命令 (cli/commands/list.py)

```python
from pathlib import Path
import typer
from rich.table import Table

from ..utils.console import console
from ...executor.session import SessionManager

app = typer.Typer()

@app.command()
def list_sessions(
    workspace: Path = typer.Option(Path(".octopus"), "--workspace", "-w", help="Workspace directory"),
    limit: int = typer.Option(10, "--limit", "-n", help="Maximum number of sessions to show"),
):
    """
    List all sessions
    """
    session_manager = SessionManager(workspace)
    sessions = session_manager.list_sessions()

    if not sessions:
        console.print("No sessions found.")
        return

    table = Table(title="Sessions")
    table.add_column("Session ID", style="cyan")
    table.add_column("Workflow", style="magenta")
    table.add_column("Status", style="green")
    table.add_column("Created", style="dim")

    for session in sessions[:limit]:
        status_color = {
            "COMPLETED": "green",
            "RUNNING": "yellow",
            "FAILED": "red",
            "CREATED": "blue"
        }.get(session.status, "white")

        table.add_row(
            session.session_id,
            session.workflow_id,
            f"[{status_color}]{session.status}[/{status_color}]",
            session.created_at.strftime("%Y-%m-%d %H:%M")
        )

    console.print(table)
    console.print(f"\nTotal: {len(sessions)} session(s)")
```

### 6.8 主 CLI 入口 (cli/main.py)

```python
import typer

from .commands import init, validate, run, status, list_cmd, version
from ..core.constants import VERSION

app = typer.Typer(
    name="openoctopus",
    help="OpenOctopus - AI-assisted development workflow orchestration tool",
    no_args_is_help=True,
)

# 注册子命令
app.add_typer(init.app, name="init")
app.add_typer(validate.app, name="validate")
app.add_typer(run.app, name="run")
app.add_typer(status.app, name="status")
app.add_typer(list_cmd.app, name="list")
app.add_typer(version.app, name="version")

@app.callback()
def main(
    version: bool = typer.Option(False, "--version", help="Show version"),
):
    if version:
        console.print(f"OpenOctopus {VERSION}")
        raise typer.Exit()

def main_entry():
    app()

if __name__ == "__main__":
    main_entry()
```

## 验收标准
- [ ] `openoctopus --version` 输出版本号
- [ ] `openoctopus init` 创建正确目录结构
- [ ] `openoctopus validate` 正确校验配置
- [ ] **`openoctopus run` 能实际执行工作流**
- [ ] **`openoctopus status` 显示执行状态**
- [ ] **`openoctopus list` 列出所有 sessions**
- [ ] Rich 美化输出正常
- [ ] 进度显示清晰
- [ ] E2E 测试全部通过

## 预计耗时
90 分钟
