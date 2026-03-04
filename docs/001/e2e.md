# OpenOctopus 阶段一 E2E 测试设计

## 1. 测试概述

### 1.1 目标
- 验证阶段一 MVP 功能的正确性
- 确保 CLI 命令行为符合预期
- 验证配置校验和执行逻辑
- 确保错误处理和退出码正确
- **验证完整工作流：init -> validate -> run -> status**

### 1.2 测试范围
| 功能模块 | 覆盖情况 |
|---------|---------|
| openoctopus init | 完整覆盖 |
| openoctopus validate | 完整覆盖 |
| **openoctopus run** | **实际执行覆盖** |
| **openoctopus status** | **完整覆盖** |
| **openoctopus list** | **完整覆盖** |
| openoctopus version | 完整覆盖 |
| 配置模型 | 结构验证 |
| **Session 管理** | **完整覆盖** |
| **产物管理** | **完整覆盖** |

### 1.3 测试工具
- pytest: 测试框架
- pytest-cov: 覆盖率
- typer.testing.CliRunner: CLI 测试
- unittest.mock: Mock LLM 调用

---

## 2. 测试场景

### 场景 1-12: init 和 validate 测试（保持原有）

参见原 e2e.md 场景 1-12。

---

### 场景 13: 完整工作流执行 - 单 Stage

**测试目标**: 验证 `openoctopus run` 能正确执行单 stage 工作流

**前置条件**:
- 有效配置和角色定义
- 需求文件存在

**测试步骤**:
```bash
1. 初始化项目: openoctopus init
2. 创建需求文件: echo "# Test" > requirement.md
3. 执行: openoctopus run --config octopus.yaml --requirement requirement.md
4. 检查退出码: 0
5. 检查输出包含 session_id
6. 检查产物文件生成
7. 检查 session 状态文件
```

**期望结果**:
- 退出码: 0
- 输出显示执行进度和完成信息
- 生成 session_id（如 sess_xxx）
- 产物文件 `.octopus/sessions/sess_xxx/artifacts/{name}.md` 存在
- session.state.md 状态为 COMPLETED

**测试文件**: `tests/e2e/test_run.py::test_run_single_stage`

---

### 场景 14: 工作流执行 - 多 Stage 顺序

**测试目标**: 验证多个 stage 按顺序执行

**前置条件**:
- 配置包含 2-3 个 stage

**测试步骤**:
```bash
1. 创建多 stage 配置
2. 执行: openoctopus run -c config.yaml -r req.md
3. 验证 stages 按数组顺序执行
4. 验证每个 stage 的产物都生成
```

**期望结果**:
- 退出码: 0
- timeline.md 记录各 stage 开始/完成时间
- 最终状态 COMPLETED

**测试文件**: `tests/e2e/test_run.py::test_run_multiple_stages`

---

### 场景 15: Session 状态查询

**测试目标**: 验证 `openoctopus status` 显示正确状态

**前置条件**:
- 已完成一次工作流执行

**测试步骤**:
```bash
1. 执行 run 命令获取 session_id
2. 执行: openoctopus status --session sess_xxx
3. 检查输出
```

**期望结果**:
- 显示 session_id, workflow_id, status
- 显示所有 stages 状态
- 显示产物列表和路径
- 退出码: 0

**测试文件**: `tests/e2e/test_status.py::test_status_completed_session`

---

### 场景 16: Session 列表查询

**测试目标**: 验证 `openoctopus list` 显示所有 sessions

**前置条件**:
- 已执行多次工作流

**测试步骤**:
```bash
1. 执行多次 run 命令
2. 执行: openoctopus list
```

**期望结果**:
- 列出所有 sessions
- 显示创建时间、状态、workflow_id
- 退出码: 0

**测试文件**: `tests/e2e/test_list.py::test_list_sessions`

---

### 场景 17: 执行失败处理

**测试目标**: 验证 stage 执行失败时的处理

**前置条件**:
- 配置有效但 role 执行会失败（如 LLM CLI 不存在）

**测试步骤**:
```bash
1. 配置使用不存在的 CLI 路径: cli_path: "/nonexistent/claude"
2. 执行: openoctopus run -c config.yaml -r req.md
3. 检查退出码和错误信息
4. 检查 session 状态
```

**期望结果**:
- 退出码: 5（执行失败）
- 错误信息清晰：指出 CLI 未找到
- session 状态为 FAILED
- timeline 记录失败原因

**测试文件**: `tests/e2e/test_run.py::test_run_llm_not_found`

---

### 场景 18: 产物文件验证

**测试目标**: 验证产物文件格式正确

**前置条件**:
- 已完成一次执行

**测试步骤**:
```bash
1. 执行 run 命令
2. 读取产物文件: .octopus/sessions/sess_xxx/artifacts/{name}.md
3. 验证格式
```

**期望结果**:
- 文件存在且可读
- 包含 Metadata 区块
- 包含 Content 区块
- Metadata 包含 artifact_name, stage_id, created_at 等

**测试文件**: `tests/e2e/test_run.py::test_artifact_format`

---

### 场景 19: Session State 文件验证

**测试目标**: 验证 session.state.md 格式

**前置条件**:
- 已完成一次执行

**测试步骤**:
```bash
1. 执行 run 命令
2. 读取: .octopus/sessions/sess_xxx/session.state.md
```

**期望结果**:
- 包含 Metadata 区块（session_id, workflow_id, status）
- 包含 Stages 表格
- 包含 Artifacts 列表

**测试文件**: `tests/e2e/test_run.py::test_session_state_format`

---

### 场景 20: 无效 Session ID 查询

**测试目标**: 验证查询不存在的 session

**测试步骤**:
```bash
1. 执行: openoctopus status --session nonexistent
```

**期望结果**:
- 退出码: 6
- 错误信息: "Session 'nonexistent' not found"

**测试文件**: `tests/e2e/test_status.py::test_status_invalid_session`

---

### 场景 21: 执行时配置校验失败

**测试目标**: 验证 run 命令执行前会校验配置

**前置条件**:
- 存在无效配置

**测试步骤**:
```bash
1. 创建引用错误的配置
2. 执行: openoctopus run -c invalid.yaml -r req.md
```

**期望结果**:
- 退出码: 3
- 错误信息指出配置错误
- 不创建 session

**测试文件**: `tests/e2e/test_run.py::test_run_invalid_config`

---

### 场景 22: Version 命令

**测试目标**: 验证 version 命令

**测试步骤**:
```bash
1. 执行: openoctopus version
2. 执行: openoctopus --version
```

**期望结果**:
- 输出版本号: "OpenOctopus 0.1.0"
- 退出码: 0

**测试文件**: `tests/e2e/test_version.py::test_version_command`

---

### 场景 23: 完整用户旅程

**测试目标**: 验证从 init 到获取结果的完整流程

**测试步骤**:
```bash
1. mkdir test_project && cd test_project
2. openoctopus init
3. echo "# Feature" > requirement.md
4. openoctopus validate -c octopus.yaml
5. openoctopus run -c octopus.yaml -r requirement.md
6. 记录输出的 session_id
7. openoctopus status -s {session_id}
8. openoctopus list
9. cat .octopus/sessions/{session_id}/artifacts/*.md
```

**期望结果**:
- 所有命令成功（退出码 0）
- 最终产物文件包含分析内容
- session 状态为 COMPLETED

**测试文件**: `tests/e2e/test_full_workflow.py::test_complete_user_journey`

---

## 3. 测试辅助工具

### 3.1 Mock LLM 执行

```python
@pytest.fixture
def mock_llm_executor(monkeypatch):
    """Mock LLM 执行，避免实际调用 CLI"""
    def mock_execute(*args, **kwargs):
        return ExecutionResult(
            status="completed",
            output="# Analysis Result\n\nThis is a mock analysis.",
            artifacts=["analysis_report"],
            error_message=None
        )

    monkeypatch.setattr(
        "openoctopus.executor.role_executor.RoleExecutor.execute",
        mock_execute
    )
```

### 3.2 配置生成器（增强）

```pythonndef create_executable_config(tmp_path: Path) -> Path:
    """创建可直接运行的配置"""
    config = tmp_path / "octopus.yaml"
    config.write_text("""
version: "2.1"
meta:
  workflow_id: "test"
  name: "Test"
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
    name: "Analyzer"
    type: "simple"
    llm_profile: "default"
    system_prompt: "Analyze the requirement."
stages:
  - id: "analyze"
    name: "Analyze"
    role: "analyzer"
    input:
      - type: "requirement_file"
        path: "./requirement.md"
    output:
      - type: "artifact"
        name: "analysis"
transitions:
  - from: "analyze"
    to: "__END__"
""")
    return config
```

### 3.3 Session 验证器

```python
def assert_session_exists(tmp_path: Path, session_id: str):
    """验证 session 目录结构"""
    session_dir = tmp_path / ".octopus" / "sessions" / session_id
    assert session_dir.exists()
    assert (session_dir / "session.state.md").exists()
    assert (session_dir / "timeline.md").exists()
    assert (session_dir / "artifacts").exists()

def assert_artifact_exists(tmp_path: Path, session_id: str, artifact_name: str):
    """验证产物存在"""
    artifact_path = tmp_path / ".octopus" / "sessions" / session_id / "artifacts" / f"{artifact_name}.md"
    assert artifact_path.exists()
    content = artifact_path.read_text()
    assert "## Metadata" in content
    assert "## Content" in content
```

---

## 4. 测试执行

### 4.1 全部 E2E 测试

```bash
pytest tests/e2e/ -v --cov=openoctopus --cov-report=html
```

### 4.2 分类执行

```bash
# Init 相关
pytest tests/e2e/test_init.py -v

# Validate 相关
pytest tests/e2e/test_validate.py -v

# Run 相关（核心）
pytest tests/e2e/test_run.py -v

# Status/List 相关
pytest tests/e2e/test_status.py tests/e2e/test_list.py -v

# 完整流程
pytest tests/e2e/test_full_workflow.py -v
```

### 4.3 覆盖率要求

| 模块 | 目标覆盖率 |
|------|-----------|
| cli/commands/ | 90% |
| config/ | 85% |
| core/ | 80% |
| **executor/** | **85%** |
| 整体 | 85% |

---

## 5. 持续集成

```yaml
# .github/workflows/test.yml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: astral-sh/setup-uv@v1
      - run: uv sync
      - run: uv run pytest tests/ -v --cov=openoctopus --cov-report=xml
      - uses: codecov/codecov-action@v3
        with:
          files: ./coverage.xml
```
