# 阶段一实现计划（MVP 可运行版）

## 概述

本文档列出 OpenOctopus 阶段一（MVP 可运行版本）的所有实现计划。

**核心目标：第一个版本就要"用起来"，而非仅静态配置。**

## 计划列表

| 编号 | 计划 | 目标 | 预计耗时 | 依赖 |
|------|------|------|----------|------|
| 01 | [项目初始化与环境搭建](01-project-setup.md) | UV 项目、依赖管理、目录结构 | 30min | - |
| 02 | [Pydantic 配置模型](02-config-models.md) | 完整的 Pydantic v2 模型 | 60min | 01 |
| 03 | [配置加载器实现](03-config-loader.md) | YAML 加载与解析 | 45min | 02 |
| 04 | [配置校验器实现](04-config-validator.md) | 引用校验、环路检测 | 60min | 02, 03 |
| 05 | [执行器核心](05-executor-core.md) | Session、Artifact、Runner | 90min | 02-04 |
| 06 | [CLI 命令实现](06-cli-commands.md) | init, validate, run, status, list | 90min | 02-05 |
| 07 | [模板系统和测试](07-templates-and-tests.md) | 内置模板、完整测试覆盖 | 60min | 01-06 |

**总预计耗时**: 约 7.5 小时

## 执行顺序

```
01 项目初始化与环境搭建
    ↓
02 Pydantic 配置模型
    ↓
03 配置加载器实现
    ↓
04 配置校验器实现
    ↓
05 执行器核心（Session、Runner、Executor）
    ↓
06 CLI 命令实现（含 run 实际执行）
    ↓
07 模板系统和测试
```

## 新增模块说明

### executor/ 模块（Plan 05）
阶段一核心新增模块，负责实际工作流执行：

```
executor/
├── __init__.py
├── session.py          # SessionManager - 会话生命周期
├── runner.py           # WorkflowRunner - 工作流执行编排
├── role_executor.py    # RoleExecutor - 单角色执行逻辑
└── artifact_manager.py # ArtifactManager - 产物管理
```

**执行流程**：
```
CLI run 命令
    ↓
WorkflowRunner.run()
    ↓
创建 Session（生成目录结构）
    ↓
遍历 stages（顺序执行）
    ↓
RoleExecutor.execute()
    ↓
调用 LLM CLI（claude/codex）
    ↓
ArtifactManager.save_artifact()
    ↓
更新 Session 状态
    ↓
输出结果摘要
```

## 验收清单（更新）

### 功能验收
- [ ] `openoctopus init` 能正确创建项目结构和可运行配置
- [ ] `openoctopus validate` 能正确校验配置
- [ ] **`openoctopus run` 能实际执行单 stage 工作流**
- [ ] **`openoctopus status` 能查询执行状态**
- [ ] **`openoctopus list` 能列出所有 sessions**
- [ ] **执行完成后产物文件正确生成**
- [ ] **session 状态文件正确维护**

### 质量验收
- [ ] 单元测试覆盖率 ≥ 85%
- [ ] E2E 测试覆盖 init → validate → run → status 完整流程
- [ ] Ruff 代码检查无错误
- [ ] 所有 CLI 命令有帮助文档

### 用户体验
- [ ] Rich 终端美化输出
- [ ] 执行进度实时显示
- [ ] 清晰的产物路径提示
- [ ] 完整的错误处理和信息提示

## 快速开始（目标体验）

```bash
# 1. 安装
pip install openoctopus

# 2. 初始化项目
mkdir my-project && cd my-project
openoctopus init

# 3. 创建需求
echo "# Feature Request" > requirement.md

# 4. 运行工作流（实际执行！）
openoctopus run -c octopus.yaml -r requirement.md

# 5. 查看结果
cat .octopus/sessions/sess_xxx/artifacts/*.md
```

## 关键实现决策

### 1. 阶段一简化执行模型
- **单线程顺序执行**：不支持并行 stages
- **simple role 类型**：单次调用，无循环/反思
- **无 TMUX**：直接在主进程执行
- **无 LangGraph**：简单数组遍历执行

### 2. LLM 调用方式
- 通过 `subprocess` 调用 CLI（claude/codex）
- 支持 stdin 输入 prompt
- 捕获 stdout 作为输出
- 预留 API 模式接口

### 3. 状态持久化
- Markdown 文件格式（便于人工查看）
- 原子写入（先写 .tmp 再重命名）
- Session 目录隔离（`.octopus/sessions/{id}/`）
