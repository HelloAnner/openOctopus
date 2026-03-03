# OpenOctopus 阶段一 E2E 测试设计

## 1. 测试概述

### 1.1 目标
- 验证阶段一核心功能的正确性
- 确保 CLI 命令行为符合预期
- 验证配置校验逻辑的正确性
- 确保错误处理和退出码正确

### 1.2 测试范围
| 功能模块 | 覆盖情况 |
|---------|---------|
| openoctopus init | 完整覆盖 |
| openoctopus validate | 完整覆盖 |
| openoctopus run | 占位验证 |
| openoctopus version | 完整覆盖 |
| 配置模型 | 结构验证 |
| 错误处理 | 场景覆盖 |

### 1.3 测试工具
- pytest: 测试框架
- pytest-asyncio: 异步测试（预留）
- pytest-cov: 覆盖率
- typer.testing.CliRunner: CLI 测试

---

## 2. 测试场景

### 场景 1: 项目初始化 - 默认模板

**测试目标**: 验证 `openoctopus init` 命令能正确创建项目结构

**前置条件**:
- 当前目录不存在 `.octopus/` 目录
- 当前目录不存在 `octopus.yaml` 文件

**测试步骤**:
```bash
1. 执行: openoctopus init
2. 检查退出码: 0
3. 检查目录结构:
   - .octopus/ 目录存在
   - .octopus/sessions/ 目录存在
   - .octopus/artifacts/ 目录存在
   - .octopus/logs/ 目录存在
   - .octopus/cache/ 目录存在
4. 检查配置文件:
   - octopus.yaml 存在
   - YAML 语法正确
   - 包含必需的字段: version, meta, runtime, llm_profiles, tool_registry, roles, stages, transitions
```

**期望结果**:
- 退出码: 0
- 目录结构完整创建
- 配置文件可解析
- 输出包含成功信息: "Initialized OpenOctopus project in ..."

**测试文件**: `tests/e2e/test_init.py::test_init_default_template`

---

### 场景 2: 项目初始化 - 指定路径

**测试目标**: 验证 `--path` 参数能正确指定初始化路径

**前置条件**:
- 目标目录 `/tmp/test_octopus` 不存在或可清空

**测试步骤**:
```bash
1. 执行: openoctopus init --path /tmp/test_octopus
2. 检查退出码: 0
3. 检查: /tmp/test_octopus/.octopus/ 目录存在
4. 检查: /tmp/test_octopus/octopus.yaml 存在
```

**期望结果**:
- 退出码: 0
- 项目在指定路径初始化

**测试文件**: `tests/e2e/test_init.py::test_init_with_path`

---

### 场景 3: 项目初始化 - 目录已存在（非 force）

**测试目标**: 验证当项目已存在时的错误处理

**前置条件**:
- 当前目录已存在 `.octopus/` 目录

**测试步骤**:
```bash
1. 创建 .octopus/ 目录
2. 执行: openoctopus init
3. 检查退出码: 4
```

**期望结果**:
- 退出码: 4
- 输出错误信息: "Project already initialized. Use --force to overwrite."
- 不修改现有文件

**测试文件**: `tests/e2e/test_init.py::test_init_existing_project`

---

### 场景 4: 项目初始化 - force 覆盖

**测试目标**: 验证 `--force` 参数能强制覆盖

**前置条件**:
- 当前目录已存在 `.octopus/` 目录和 `octopus.yaml`

**测试步骤**:
```bash
1. 创建初始项目结构
2. 修改 octopus.yaml 内容（添加标记）
3. 执行: openoctopus init --force
4. 检查退出码: 0
5. 检查: 配置文件已重置
```

**期望结果**:
- 退出码: 0
- 目录结构保留
- 配置文件重新生成
- 输出警告信息: "Force mode: existing config may be overwritten"

**测试文件**: `tests/e2e/test_init.py::test_init_force`

---

### 场景 5: 配置校验通过

**测试目标**: 验证有效配置能通过校验

**前置条件**:
- 存在有效的 octopus.yaml 文件

**测试步骤**:
```bash
1. 创建有效配置:
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
     builtin:
       file_read:
         module: "openoctopus.tools.file"
         class: "FileReadTool"
   roles:
     - id: "test_role"
       name: "Test"
       type: "simple"
       llm_profile: "codex"
       system_prompt: "Test prompt"
   stages:
     - id: "test_stage"
       name: "Test Stage"
       role: "test_role"
       input:
         - type: "requirement_file"
           path: "./test.md"
       output:
         - type: "artifact"
           name: "result"
   transitions:
     - from: "test_stage"
       to: "__END__"

2. 执行: openoctopus validate --config ./octopus.yaml
3. 检查退出码: 0
```

**期望结果**:
- 退出码: 0
- 输出成功信息: "✓ Configuration is valid"
- verbose 模式显示详细校验过程

**测试文件**: `tests/e2e/test_validate.py::test_validate_success`

---

### 场景 6: 配置校验失败 - YAML 语法错误

**测试目标**: 验证 YAML 语法错误的检测

**前置条件**:
- 存在语法错误的 YAML 文件

**测试步骤**:
```bash
1. 创建无效 YAML:
   version: "2.1
   meta:
     workflow_id: test
     # 缺少闭合引号，缩进错误等

2. 执行: openoctopus validate --config ./invalid.yaml
3. 检查退出码: 2
```

**期望结果**:
- 退出码: 2
- 输出解析错误信息，包含行号和列号
- 指出具体语法问题

**测试文件**: `tests/e2e/test_validate.py::test_validate_yaml_syntax_error`

---

### 场景 7: 配置校验失败 - 结构错误（必填字段缺失）

**测试目标**: 验证必填字段缺失的检测

**前置条件**:
- 配置缺少必填字段

**测试步骤**:
```bash
1. 创建配置（缺少 meta.workflow_id）:
   version: "2.1"
   meta:
     name: "Test"  # 缺少 workflow_id

2. 执行: openoctopus validate --config ./incomplete.yaml
3. 检查退出码: 3
```

**期望结果**:
- 退出码: 3
- 输出字段错误: "Field required: meta.workflow_id"
- 错误类型: structure

**测试文件**: `tests/e2e/test_validate.py::test_validate_missing_required_field`

---

### 场景 8: 配置校验失败 - 引用错误（Role 不存在）

**测试目标**: 验证 stage.role 引用不存在的 role 时出错

**前置条件**:
- stage 引用了未定义的 role

**测试步骤**:
```bash
1. 创建配置:
   # ... 基础配置 ...
   roles:
     - id: "role_a"
       # ...
   stages:
     - id: "stage_1"
       role: "nonexistent_role"  # 不存在的 role
       # ...

2. 执行: openoctopus validate --config ./bad_ref.yaml
3. 检查退出码: 3
```

**期望结果**:
- 退出码: 3
- 错误信息: "Stage 'stage_1' references undefined role 'nonexistent_role'"
- 错误类型: reference

**测试文件**: `tests/e2e/test_validate.py::test_validate_invalid_role_reference`

---

### 场景 9: 配置校验失败 - 引用错误（Stage 不存在）

**测试目标**: 验证 transition 引用不存在的 stage

**前置条件**:
- transition 引用了未定义的 stage

**测试步骤**:
```bash
1. 创建配置:
   # ... 基础配置 ...
   stages:
     - id: "stage_a"
       # ...
   transitions:
     - from: "stage_a"
       to: "nonexistent_stage"

2. 执行: openoctopus validate --config ./bad_transition.yaml
3. 检查退出码: 3
```

**期望结果**:
- 退出码: 3
- 错误信息: "Transition references undefined stage 'nonexistent_stage'"
- 错误类型: reference

**测试文件**: `tests/e2e/test_validate.py::test_validate_invalid_stage_reference`

---

### 场景 10: 配置校验失败 - 环路检测

**测试目标**: 验证 transition 环路的检测

**前置条件**:
- transition 形成环路

**测试步骤**:
```bash
1. 创建配置:
   # ... 基础配置 ...
   stages:
     - id: "stage_a"
       # ...
     - id: "stage_b"
       # ...
   transitions:
     - from: "stage_a"
       to: "stage_b"
     - from: "stage_b"
       to: "stage_a"  # 形成环路

2. 执行: openoctopus validate --config ./loop.yaml
3. 检查退出码: 3
```

**期望结果**:
- 退出码: 3
- 错误信息: "Cycle detected in transitions: stage_a -> stage_b -> stage_a"
- 错误类型: loop

**测试文件**: `tests/e2e/test_validate.py::test_validate_loop_detection`

---

### 场景 11: 配置校验 - 安全警告

**测试目标**: 验证 shell_exec 工具的安全警告

**前置条件**:
- role 使用 shell_exec 但未配置 security.shell

**测试步骤**:
```bash
1. 创建配置:
   # ... 基础配置 ...
   roles:
     - id: "danger_role"
       # ...
       tools: ["shell_exec"]
   # 缺少 security.shell 配置

2. 执行: openoctopus validate --config ./no_security.yaml
3. 检查退出码: 0（警告不阻止通过）
```

**期望结果**:
- 退出码: 0
- 警告信息: "Role 'danger_role' uses 'shell_exec' without security.shell configuration"
- 错误类型: warning

**测试文件**: `tests/e2e/test_validate.py::test_validate_security_warning`

---

### 场景 12: 配置校验 - 多种格式输出

**测试目标**: 验证不同输出格式

**前置条件**:
- 存在包含多个错误的配置文件

**测试步骤**:
```bash
1. 执行: openoctopus validate --config ./errors.yaml --format table
2. 执行: openoctopus validate --config ./errors.yaml --format json
3. 执行: openoctopus validate --config ./errors.yaml --format yaml
```

**期望结果**:
- table 格式: 表格形式显示错误
- json 格式: 可解析的 JSON 输出
- yaml 格式: YAML 格式的错误列表

**测试文件**: `tests/e2e/test_validate.py::test_validate_output_formats`

---

### 场景 13: Version 命令

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

### 场景 14: Run 命令占位

**测试目标**: 验证 run 命令占位提示

**测试步骤**:
```bash
1. 执行: openoctopus run
2. 执行: openoctopus run --config ./octopus.yaml
```

**期望结果**:
- 输出: "Run command is not implemented yet. Coming in Phase 2."
- 退出码: 0（占位命令不报错）

**测试文件**: `tests/e2e/test_run.py::test_run_placeholder`

---

### 场景 15: 帮助信息

**测试目标**: 验证所有命令的帮助信息

**测试步骤**:
```bash
1. 执行: openoctopus --help
2. 执行: openoctopus init --help
3. 执行: openoctopus validate --help
4. 执行: openoctopus run --help
5. 执行: openoctopus version --help
```

**期望结果**:
- 每个命令都显示完整的 help 信息
- 包含参数说明和示例
- 格式美观（Rich 渲染）

**测试文件**: `tests/e2e/test_help.py::test_all_commands_help`

---

## 3. 测试数据结构

### 3.1 最小有效配置

```yaml
version: "2.1"

meta:
  workflow_id: "minimal-test"
  name: "Minimal Test"

runtime:
  workspace:
    root: ".octopus"
    sessions_dir: ".octopus/sessions"
    artifacts_dir: ".octopus/artifacts"
    logs_dir: ".octopus/logs"

llm_profiles:
  codex_cli:
    provider: "codex"
    mode: "cli"
    cli_path: "codex"

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
    system_prompt: "你是一位严格的代码审查员..."
    tools: ["file_read", "file_write"]

stages:
  - id: "review"
    name: "代码审查"
    role: "review_agent"
    input:
      - type: "requirement_file"
        path: "./review.diff"
    output:
      - type: "artifact"
        name: "review_result"

transitions:
  - from: "review"
    to: "__END__"
```

### 3.2 无效配置示例

**结构错误**:
```yaml
version: "2.1"
meta:
  name: "Missing workflow_id"  # 缺少 workflow_id
```

**引用错误**:
```yaml
# roles 未定义 test_role，但 stage 引用了
stages:
  - id: "test"
    role: "test_role"  # 错误
```

**环路配置**:
```yaml
stages:
  - id: "a"
  - id: "b"
transitions:
  - from: "a"
    to: "b"
  - from: "b"
    to: "a"  # 环路
```

---

## 4. 测试辅助工具

### 4.1 配置生成器

```python
def create_minimal_config(tmp_path: Path) -> Path:
    """创建最小有效配置"""
    config = tmp_path / "octopus.yaml"
    config.write_text(MINIMAL_CONFIG)
    return config

def create_invalid_config(tmp_path: Path, error_type: str) -> Path:
    """创建特定类型的无效配置"""
    configs = {
        "syntax": INVALID_SYNTAX,
        "missing_field": MISSING_FIELD,
        "bad_reference": BAD_REFERENCE,
        "loop": LOOP_CONFIG,
    }
    config = tmp_path / "invalid.yaml"
    config.write_text(configs[error_type])
    return config
```

### 4.2 结果验证器

```python
def assert_exit_code(result, expected: int):
    """验证退出码"""
    assert result.exit_code == expected, f"Expected {expected}, got {result.exit_code}"

def assert_output_contains(result, text: str):
    """验证输出包含指定文本"""
    assert text in result.output

def assert_valid_json_output(result):
    """验证输出为有效 JSON"""
    import json
    data = json.loads(result.output)
    assert "valid" in data
    assert "errors" in data
```

---

## 5. 测试执行

### 5.1 执行全部 E2E 测试

```bash
pytest tests/e2e/ -v --cov=openoctopus --cov-report=html
```

### 5.2 执行特定测试

```bash
# 仅 init 测试
pytest tests/e2e/test_init.py -v

# 仅 validate 测试
pytest tests/e2e/test_validate.py -v

# 特定场景
pytest tests/e2e/test_validate.py::test_validate_loop_detection -v
```

### 5.3 覆盖率要求

| 模块 | 目标覆盖率 |
|------|-----------|
| cli/commands/ | 90% |
| config/ | 85% |
| core/ | 80% |
| 整体 | 85% |

---

## 6. 持续集成

### 6.1 CI 配置

```yaml
# .github/workflows/test.yml
name: E2E Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: astral-sh/setup-uv@v1
      - run: uv sync
      - run: uv run pytest tests/e2e/ -v --cov=openoctopus
```

### 6.2 测试报告

- 生成 HTML 覆盖率报告
- 生成 JUnit XML 格式结果
- 失败时截图/日志归档
