# OpenOctopus

命令行研发流程编排工具，通过 YAML 配置将文档设计、代码实现、代码审查、测试验证等研发环节串联成可自动执行的工作流。

## 核心价值

| 价值 | 说明 |
|------|------|
| **流程自动化** | 将"人肉编排"转变为"自动驱动" |
| **状态可追溯** | 执行结果、中间产物、决策依据全记录 |
| **人机协作** | 支持自动执行与人工介入的灵活切换 |
| **可复用性** | YAML 配置驱动，团队共享最佳实践 |

## 快速开始

```bash
# 构建
go build -o openoctopus ./cmd/openoctopus

# 校验配置
./openoctopus validate --config ./octopus.yaml

# 执行工作流
./openoctopus run --config ./octopus.yaml

# 等价简写
./openoctopus ./octopus.yaml

# 如果开启 runtime.tmux.enabled=true，且当前是交互式终端，run 会自动进入新建的 tmux 会话
# role pane 会自动拉起对应 role 的交互式 Codex 页面，默认焦点落到第一个 role pane

# 查看状态
./openoctopus status --session <session_id>

# 人工介入
./openoctopus interrupt --session <session_id> --role <role_id> --reason "..."
./openoctopus interrupt-all --session <session_id> --reason "..."
./openoctopus inject --session <session_id> --input ./note.md
./openoctopus switch --session <session_id> --role <role_id>

# 恢复执行
./openoctopus resume --session <session_id>
```

## 项目结构

```
.
├── cmd/openoctopus/       # CLI 入口与子命令实现
│   ├── main.go            # 程序入口
│   ├── root.go            # 根命令
│   ├── validate.go        # 配置校验
│   ├── run.go             # 执行工作流
│   ├── status.go          # 查看状态
│   ├── interrupt.go       # 中断指定角色
│   ├── interrupt_all.go   # 中断全部角色
│   ├── inject.go          # 注入人工输入
│   └── resume.go          # 恢复执行
│
├── internal/
│   ├── config/            # 配置域核心实现
│   │   ├── model/         # 配置模型定义
│   │   ├── loader/        # YAML 加载与合并
│   │   ├── defaults/      # 默认值注入
│   │   ├── validator/     # 配置校验
│   │   ├── service/       # 配置服务
│   │   └── errors/        # 错误定义
│   │
│   ├── session/           # Session 工作目录骨架
│   │
│   ├── eventbus/          # 事件总线（事件存储、锁机制、中断处理）
│   │
│   ├── orchestrator/      # 流程编排引擎
│   │
│   ├── roleruntime/       # 角色运行时
│   │
│   ├── artifact/          # 产物管理
│   │
│   ├── humangate/         # 人工介入机制
│   │
│   ├── tmux/              # TMUX 会话编排与 pane 切换
│   │
│   └── cli/               # CLI 输出格式化
│
├── docs/                  # 产品与模块设计文档
│   ├── prd/              # 平台整体 PRD
│   ├── config/           # 配置模块设计
│   ├── session/          # Session 模块设计
│   ├── event-bus/        # 事件总线设计
│   ├── orchestrator/     # 编排引擎设计
│   ├── role-runtime/     # 角色运行时设计
│   ├── artifact/         # 产物管理设计
│   ├── human-gate/       # 人工介入设计
│   ├── tmux/             # TMUX 设计
│   ├── cli/              # CLI 设计
│   └── timeline.md       # 版本演进时间线
│
├── e2e/                   # E2E 测试（Python + Playwright）
└── Makefile               # 构建与检查入口
```

## 核心概念

| 术语 | 定义 |
|------|------|
| **Workflow** | 完整的研发流程定义，由多个 Stage 组成 |
| **Stage** | 流程阶段，对应具体研发环节（如代码实现、审查） |
| **Role** | 执行 Stage 的角色，包含系统提示词、工具集、LLM 配置 |
| **Transition** | Stage 间流转规则，支持条件分支和循环 |
| **Artifact** | 流程产物（文档、代码、报告等） |
| **Session** | 一次具体的流程执行实例 |

## 核心理念

OpenOctopus 的核心不是"多 Agent 本身"，而是"**基于文件系统的可追溯协作总线**"。

- **单一事实源**：会话状态、调度计划、角色结论、执行时间线全部落在 `.octopus/sessions/{session_id}/` 下的 `.md` 文件
- **文件即协议**：通过 `inbox.md`/`outbox.md` 做任务分发与回执，通过 `context.md` 传递执行上下文
- **先写事件再推进状态**：WAL 思想，保证可恢复性
- **人机协作**：支持在任意阶段打断工作流，注入人工输入后继续执行

## 配置示例

```yaml
version: "2.1"

meta:
  workflow_id: "feature-dev-loop"
  name: "功能开发闭环流程"

llm_profiles:
  claude_code_cli:
    provider: "claude_code"
    cli_path: "claude"

roles:
  planner:
    profile: claude_code_cli
    system_prompt: "你是一个任务规划专家..."

  coder:
    profile: claude_code_cli
    system_prompt: "你是一个代码实现专家..."

stages:
  - id: plan
    role: planner
    transition:
      next: code

  - id: code
    role: coder
    transition:
      next: review

  - id: review
    role: reviewer
```

## 开发指南

详见 [AGENTS.md](./AGENTS.md)。

## License

MIT
