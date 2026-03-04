# Plan 05: 执行器核心实现

## 目标
实现工作流执行的核心模块，包括 Session 管理、Workflow 执行、Role 调用和产物管理。

## 任务清单

### 5.1 Session 管理 (executor/session.py)

```python
"""
Session 管理模块

负责 Session 的创建、加载、更新和生命周期管理。
Session 状态持久化为 Markdown 文件。
"""

import uuid
from datetime import datetime
from pathlib import Path

from pydantic import BaseModel

from ..config.models import OctopusConfig


class Session(BaseModel):
    """会话状态模型"""
    session_id: str
    workflow_id: str
    config_path: Path
    status: str  # CREATED | RUNNING | COMPLETED | FAILED
    created_at: datetime
    updated_at: datetime
    stages_status: dict[str, 'StageExecution']
    artifacts: list['Artifact']


class StageExecution(BaseModel):
    """Stage 执行状态"""
    stage_id: str
    status: str  # PENDING | RUNNING | COMPLETED | FAILED
    started_at: datetime | None = None
    completed_at: datetime | None = None
    error_message: str | None = None


class Artifact(BaseModel):
    """产物模型"""
    name: str
    path: Path
    stage_id: str
    created_at: datetime
    content_hash: str


class SessionManager:
    """会话管理器"""

    def __init__(self, workspace_root: Path = Path(".octopus")):
        self.workspace_root = Path(workspace_root)
        self.sessions_dir = self.workspace_root / "sessions"

    def create_session(self, config: OctopusConfig, config_path: Path) -> Session:
        """
        创建新 Session

        1. 生成唯一 session_id
        2. 创建 session 目录结构
        3. 初始化状态文件
        4. 返回 Session 对象
        """
        session_id = f"sess_{uuid.uuid4().hex[:12]}"
        session_dir = self.sessions_dir / session_id

        # 创建目录
        session_dir.mkdir(parents=True, exist_ok=True)
        (session_dir / "artifacts").mkdir(exist_ok=True)

        # 初始化 Session
        now = datetime.utcnow()
        session = Session(
            session_id=session_id,
            workflow_id=config.meta.workflow_id,
            config_path=config_path,
            status="CREATED",
            created_at=now,
            updated_at=now,
            stages_status={},
            artifacts=[]
        )

        # 写入初始状态
        self._write_state(session)
        self._write_timeline_event(session_id, "SESSION_CREATED", {"config": str(config_path)})

        return session

    def update_session(self, session: Session) -> None:
        """更新 Session 状态"""
        session.updated_at = datetime.utcnow()
        self._write_state(session)

    def load_session(self, session_id: str) -> Session | None:
        """从状态文件加载 Session"""
        state_file = self.sessions_dir / session_id / "session.state.md"
        if not state_file.exists():
            return None
        return self._parse_state_file(state_file)

    def list_sessions(self) -> list[Session]:
        """列出所有 Sessions"""
        sessions = []
        if not self.sessions_dir.exists():
            return sessions

        for session_dir in self.sessions_dir.iterdir():
            if session_dir.is_dir() and session_dir.name.startswith("sess_"):
                session = self.load_session(session_dir.name)
                if session:
                    sessions.append(session)

        return sorted(sessions, key=lambda s: s.created_at, reverse=True)

    def _write_state(self, session: Session) -> None:
        """写入 session.state.md"""
        state_file = self.sessions_dir / session.session_id / "session.state.md"
        content = self._format_state_markdown(session)
        state_file.write_text(content, encoding='utf-8')

    def _write_timeline_event(self, session_id: str, event_type: str, data: dict) -> None:
        """追加 timeline.md 事件"""
        timeline_file = self.sessions_dir / session_id / "timeline.md"
        timestamp = datetime.utcnow().isoformat()

        lines = [
            f"\n### {timestamp} - {event_type}",
        ]
        for key, value in data.items():
            lines.append(f"- {key}: {value}")
        lines.append("")

        with open(timeline_file, "a", encoding='utf-8') as f:
            f.write("\n".join(lines))

    def _format_state_markdown(self, session: Session) -> str:
        """格式化 state 为 Markdown"""
        lines = [
            "# Session State",
            "",
            "## Metadata",
            f"- session_id: {session.session_id}",
            f"- workflow_id: {session.workflow_id}",
            f"- status: {session.status}",
            f"- created_at: {session.created_at.isoformat()}",
            f"- updated_at: {session.updated_at.isoformat()}",
            "",
            "## Stages",
            "| Stage | Status | Started | Completed |",
            "|-------|--------|---------|-----------|",
        ]

        for stage_id, stage_exec in session.stages_status.items():
            started = stage_exec.started_at.isoformat() if stage_exec.started_at else "-"
            completed = stage_exec.completed_at.isoformat() if stage_exec.completed_at else "-"
            lines.append(f"| {stage_id} | {stage_exec.status} | {started} | {completed} |")

        lines.extend([
            "",
            "## Artifacts",
        ])
        for artifact in session.artifacts:
            lines.append(f"- {artifact.name}: `{artifact.path}`")

        return "\n".join(lines)

    def _parse_state_file(self, state_file: Path) -> Session:
        """解析 state 文件（简化实现）"""
        # MVP 版本：直接从目录结构重建 Session
        # 完整解析 Markdown 可作为后续优化
        pass
```

### 5.2 产物管理 (executor/artifact_manager.py)

```python
"""产物管理器"""

import hashlib
from datetime import datetime
from pathlib import Path

from .session import Artifact


class ArtifactManager:
    """产物管理器"""

    def __init__(self, session_dir: Path):
        self.artifacts_dir = session_dir / "artifacts"
        self.artifacts_dir.mkdir(parents=True, exist_ok=True)

    def save_artifact(
        self,
        name: str,
        content: str,
        stage_id: str,
        metadata: dict | None = None
    ) -> Artifact:
        """
        保存产物

        产物格式为 Markdown，包含 Metadata 和 Content 区块。
        """
        # 计算 hash
        content_hash = hashlib.sha256(content.encode()).hexdigest()[:16]

        # 构建产物内容
        lines = [
            f"# {name}",
            "",
            "## Metadata",
            f"- artifact_name: {name}",
            f"- stage_id: {stage_id}",
            f"- created_at: {datetime.utcnow().isoformat()}",
            f"- content_hash: {content_hash}",
        ]

        if metadata:
            for key, value in metadata.items():
                lines.append(f"- {key}: {value}")

        lines.extend([
            "",
            "## Content",
            "",
            content,
        ])

        artifact_content = "\n".join(lines)

        # 写入文件
        artifact_path = self.artifacts_dir / f"{name}.md"
        artifact_path.write_text(artifact_content, encoding='utf-8')

        return Artifact(
            name=name,
            path=artifact_path,
            stage_id=stage_id,
            created_at=datetime.utcnow(),
            content_hash=content_hash
        )

    def load_artifact(self, name: str) -> str:
        """读取产物内容"""
        artifact_path = self.artifacts_dir / f"{name}.md"
        if not artifact_path.exists():
            raise FileNotFoundError(f"Artifact '{name}' not found")
        return artifact_path.read_text(encoding='utf-8')

    def list_artifacts(self) -> list[str]:
        """列出所有产物名称"""
        artifacts = []
        for path in self.artifacts_dir.glob("*.md"):
            artifacts.append(path.stem)
        return sorted(artifacts)
```

### 5.3 Role 执行器 (executor/role_executor.py)

```python
"""Role 执行器"""

import subprocess
from dataclasses import dataclass
from pathlib import Path

from ..config.models import Role, Stage, LLMProfile


@dataclass
class ExecutionResult:
    """执行结果"""
    status: str  # completed | failed
    output: str
    artifacts: list[str]
    error_message: str | None = None


class RoleExecutor:
    """Role 执行器 - 阶段一只实现 simple 类型"""

    def execute(
        self,
        role: Role,
        stage: Stage,
        requirement_content: str,
        llm_profile: LLMProfile
    ) -> ExecutionResult:
        """
        执行 Role

        阶段一简化逻辑：
        1. 构建完整 prompt（system_prompt + input）
        2. 调用 LLM CLI
        3. 返回输出
        """
        if role.type != "simple":
            return ExecutionResult(
                status="failed",
                output="",
                artifacts=[],
                error_message=f"Role type '{role.type}' not supported in Phase 1"
            )

        # 构建 prompt
        prompt = self._build_prompt(role, stage, requirement_content)

        # 执行 LLM 调用
        try:
            output = self._call_llm(llm_profile, prompt)
            return ExecutionResult(
                status="completed",
                output=output,
                artifacts=[output.name for output in stage.output]
            )
        except Exception as e:
            return ExecutionResult(
                status="failed",
                output="",
                artifacts=[],
                error_message=str(e)
            )

    def _build_prompt(self, role: Role, stage: Stage, requirement_content: str) -> str:
        """构建完整 prompt"""
        lines = [
            role.system_prompt,
            "",
            "---",
            "",
            "## Task",
            f"Stage: {stage.name}",
            "",
            "## Input",
            requirement_content,
            "",
            "## Expected Output",
            "Please provide your output in markdown format.",
        ]
        return "\n".join(lines)

    def _call_llm(self, profile: LLMProfile, prompt: str) -> str:
        """
        调用 LLM

        阶段一只支持 CLI 模式。
        """
        if profile.mode != "cli":
            raise NotImplementedError(f"Mode '{profile.mode}' not supported in Phase 1")

        cli_path = profile.cli_path or profile.provider

        # 构建命令
        cmd = [cli_path]

        # 添加常用参数
        if profile.provider == "claude_code":
            cmd.extend(["--print"])  # 非交互模式
        elif profile.provider == "codex":
            cmd.extend(["--no-interactive"])

        # 执行命令
        try:
            result = subprocess.run(
                cmd,
                input=prompt,
                capture_output=True,
                text=True,
                timeout=300  # 5分钟超时
            )

            if result.returncode != 0:
                raise RuntimeError(f"LLM CLI failed: {result.stderr}")

            return result.stdout

        except FileNotFoundError:
            raise RuntimeError(
                f"LLM CLI not found: '{cli_path}'. "
                f"Please install {profile.provider} CLI."
            )
        except subprocess.TimeoutExpired:
            raise RuntimeError("LLM execution timed out")
```

### 5.4 Workflow 运行器 (executor/runner.py)

```python
"""Workflow 运行器"""

from pathlib import Path
from datetime import datetime

from ..config.models import OctopusConfig, Stage
from .session import SessionManager, Session, StageExecution
from .role_executor import RoleExecutor, ExecutionResult
from .artifact_manager import ArtifactManager


class WorkflowRunner:
    """工作流运行器"""

    def __init__(
        self,
        config: OctopusConfig,
        session_manager: SessionManager,
        role_executor: RoleExecutor | None = None
    ):
        self.config = config
        self.session_manager = session_manager
        self.role_executor = role_executor or RoleExecutor()

    def run(self, requirement_path: Path, session: Session | None = None) -> Session:
        """
        执行工作流

        阶段一简化逻辑：按 stages 数组顺序执行
        """
        # 创建或复用 session
        if session is None:
            session = self.session_manager.create_session(
                self.config,
                Path("octopus.yaml")  # 简化处理
            )

        # 初始化 artifact manager
        artifact_manager = ArtifactManager(
            self.session_manager.sessions_dir / session.session_id
        )

        # 读取需求文件
        requirement_content = requirement_path.read_text(encoding='utf-8')

        # 更新状态为运行中
        session.status = "RUNNING"
        self.session_manager.update_session(session)
        self._log_event(session.session_id, "WORKFLOW_STARTED", {})

        try:
            # 按顺序执行 stages
            for stage in self.config.stages:
                self._execute_stage(
                    session,
                    stage,
                    requirement_content,
                    artifact_manager
                )

            # 标记完成
            session.status = "COMPLETED"
            self._log_event(session.session_id, "WORKFLOW_COMPLETED", {
                "total_stages": len(self.config.stages)
            })

        except Exception as e:
            session.status = "FAILED"
            self._log_event(session.session_id, "WORKFLOW_FAILED", {
                "error": str(e)
            })
            raise

        finally:
            self.session_manager.update_session(session)

        return session

    def _execute_stage(
        self,
        session: Session,
        stage: Stage,
        requirement_content: str,
        artifact_manager: ArtifactManager
    ) -> None:
        """执行单个 Stage"""
        # 获取 role
        role = self.config.get_role(stage.role)
        if not role:
            raise ValueError(f"Role '{stage.role}' not found")

        # 获取 llm_profile
        llm_profile = self.config.llm_profiles.get(role.llm_profile)
        if not llm_profile:
            raise ValueError(f"LLM profile '{role.llm_profile}' not found")

        # 更新 stage 状态为运行中
        stage_exec = StageExecution(
            stage_id=stage.id,
            status="RUNNING",
            started_at=datetime.utcnow()
        )
        session.stages_status[stage.id] = stage_exec
        self.session_manager.update_session(session)
        self._log_event(session.session_id, "STAGE_STARTED", {
            "stage_id": stage.id,
            "role_id": role.id
        })

        try:
            # 执行 role
            result = self.role_executor.execute(
                role,
                stage,
                requirement_content,
                llm_profile
            )

            if result.status == "completed":
                # 保存产物
                for output in stage.output:
                    artifact = artifact_manager.save_artifact(
                        name=output.name,
                        content=result.output,
                        stage_id=stage.id
                    )
                    session.artifacts.append(artifact)

                # 更新状态为完成
                stage_exec.status = "COMPLETED"
                stage_exec.completed_at = datetime.utcnow()
                self._log_event(session.session_id, "STAGE_COMPLETED", {
                    "stage_id": stage.id,
                    "artifacts": [a.name for a in session.artifacts if a.stage_id == stage.id]
                })
            else:
                # 执行失败
                stage_exec.status = "FAILED"
                stage_exec.error_message = result.error_message
                raise RuntimeError(f"Stage '{stage.id}' failed: {result.error_message}")

        except Exception as e:
            stage_exec.status = "FAILED"
            stage_exec.error_message = str(e)
            self._log_event(session.session_id, "STAGE_FAILED", {
                "stage_id": stage.id,
                "error": str(e)
            })
            raise

        finally:
            self.session_manager.update_session(session)

    def _log_event(self, session_id: str, event_type: str, data: dict) -> None:
        """记录时间线事件"""
        # 通过 SessionManager 的 timeline 方法记录
        pass  # 实现在 SessionManager 中
```

## 验收标准
- [ ] SessionManager 能正确创建/加载/更新 Session
- [ ] ArtifactManager 能正确保存/加载产物
- [ ] RoleExecutor 能调用 LLM CLI 并返回结果
- [ ] WorkflowRunner 能按顺序执行 stages
- [ ] Session 状态文件正确生成
- [ ] Timeline 事件正确记录
- [ ] 单元测试覆盖率 ≥ 85%

## 测试用例

```python
# tests/unit/executor/test_session.py
def test_create_session(tmp_path):
    manager = SessionManager(tmp_path / ".octopus")
    config = create_test_config()

    session = manager.create_session(config, tmp_path / "config.yaml")

    assert session.session_id.startswith("sess_")
    assert session.status == "CREATED"
    assert (tmp_path / ".octopus" / "sessions" / session.session_id).exists()

# tests/unit/executor/test_role_executor.py
def test_execute_simple_role(mock_subprocess):
    executor = RoleExecutor()
    role = create_test_role(type="simple")
    stage = create_test_stage()
    profile = create_test_profile()

    result = executor.execute(role, stage, "test input", profile)

    assert result.status == "completed"
    assert "mock output" in result.output
```

## 预计耗时
90 分钟
