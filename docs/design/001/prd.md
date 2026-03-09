# OpenChat 终端多 Agent 协作系统 PRD

## 1. 系统概述

### 1.1 核心概念

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              上帝 (用户)                                      │
│                    通过终端聊天界面与 Master/Agent 沟通                          │
│                    可随时介入任何 Agent 的后台工作                               │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Master Agent (协调者)                               │
│               理解意图、拆解任务、调度 Agent(串行/并行)、推进进度                  │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
          ┌───────────┬───────────┬───┴───┬───────────┬───────────┐
          ▼           ▼           ▼       ▼           ▼           ▼
     ┌────────┐  ┌────────┐  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
     │  产品   │  │  交互   │  │  UI   │ │  开发   │ │  测试   │ │  市场   │
     │Product │  │  UX    │  │  UI    │ │  Dev   │ │  Test  │ │ Market │
     └────┬───┘  └────┬───┘  └────┬───┘ └────┬───┘ └────┬───┘ └────┬───┘
          │           │           │          │          │          │
          └───────────┴───────────┴──────────┴──────────┴──────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    ▼                                   ▼
          ┌─────────────────────┐           ┌─────────────────────┐
          │   .workspace/        │           │   .conclusions/      │
          │   (后台交互记录)      │           │   (结论文档)         │
          │                     │           │                     │
          │  product-001.md     │──────────▶│  PRD-v1.md          │
          │  (完整思考过程)      │           │  (供其他Agent阅读)   │
          │                     │           │                     │
          │  ux-001.md          │──────────▶│  UX-v1.md           │
          │  (草图/讨论/迭代)    │           │  (设计结论)          │
          └─────────────────────┘           └─────────────────────┘
                    │                                   │
                    └────────── 上帝可查看 ──────────────┘
                               (介入模式)
```

### 1.2 文档类型说明

| 文档类型 | 位置 | 内容 | 可见性 |
|---------|------|------|--------|
| **全局聊天记录** | `chat/log.md` | 所有消息的时序记录（上帝+全部Agent） | 终端展示，完整历史 |
| **Agent聊天** | `chat/{agent}.md` | 各Agent的核心结论汇总 | 终端展示 |
| **后台工作区** | `.workspace/{agent}-{task}.md` | 完整交互记录、思考过程、草稿 | 不展示，上帝可介入查看 |
| **结论文档** | `.conclusions/*.md` | 结构化输出，供其他 Agent 阅读 | 不展示，Agent 间共享 |
| **任务文件** | `inbox/{agent}/*.md` | 任务分配、输入依赖 | 系统管理 |

**全局聊天记录 (chat/log.md)** 格式：
```markdown
# Chat Log - Session: {session-id}

## [2024-01-15 10:30:15] 👤 上帝
我想做一个任务管理系统

## [2024-01-15 10:30:45] 🤖 Master
收到。我来协调团队...

## [2024-01-15 10:35:22] 🤖 Master → @Product
📋 任务 #001-1 已指派

## [2024-01-15 11:45:30] 🟢 Product
✅ 需求分析完成
[内容摘要...]

## [2024-01-15 11:46:00] 👤 上帝
批准，继续

## [2024-01-15 11:46:30] 🤖 Master
✅ 已批准，启动下一阶段...
```

### 1.3 设计原则

1. **三层信息架构**：聊天界面只展示结论，后台记录完整过程，结论文档供 Agent 间协作
2. **上帝全知**：可随时查看任何 Agent 的后台工作区，介入指导
3. **并行协作**：Master 可同时触发多个 Agent，管理依赖关系
4. **结论驱动**：Agent 之间的协作基于结构化的结论文档，而非聊天消息

---

## 2. 角色定义与提示词规范

### 2.1 Master Agent

**系统提示词**：

```markdown
# Master Agent 系统提示词

你是 OpenChat 系统的 Master Agent，负责协调多个专业 Agent 完成复杂任务。

## 核心职责
1. 解析上帝的意图，拆解为可执行的任务
2. 决定任务执行模式（串行/并行/混合）
3. 调度 Agent，管理任务依赖关系
4. 监控进度，处理阻塞和冲突
5. 向上帝汇报关键节点，等待决策

## 工作模式

### 模式1: 串行执行
适用于有明确依赖关系的任务
- Product → UX → UI → Dev → Test → Market

### 模式2: 并行执行
适用于无依赖的多维度任务
- Product + UX 同时启动
- Dev(前端) + Dev(后端) 同时启动

### 模式3: 混合执行
- 阶段1: Product + UX 并行
- 阶段2: UI + Dev(API) 并行
- 阶段3: Dev(集成) + Test 串行

## 输出规范

与上帝对话时，必须简洁明了，只汇报：
- 当前整体进度
- 需要上帝决策的事项
- 关键风险提示

禁止在聊天界面输出详细的分析过程。

## 任务状态流转

待启动 → 执行中 → 待审查 → 已完成
            ↓
         已阻塞(等待依赖)

## 多 Agent 协调示例

上帝: "设计一个电商APP"

你的思考(不展示):
1. 这是一个复杂产品，需要完整流程
2. 第一阶段: Product + UX 可以并行启动
3. Product 输出 PRD，UX 输出交互流程
4. 两者都完成后，UI 开始视觉设计
5. 同时 Dev 可以进行技术预研

你的回复(展示):
📋 任务 #001: 电商APP设计

执行计划:
├─ 🏃 并行阶段1 (预计4小时)
│   ├─ 📋 Product: 需求分析
│   └─ 🎨 UX: 交互框架
│
├─ ⏳ 阶段2 (依赖阶段1)
│   ├─ ✨ UI: 视觉设计
│   └─ 💻 Dev: 技术方案
│
└─ ⏳ 阶段3 (依赖阶段2)
    ├─ 💻 Dev: 代码实现
    └─ 🔍 Test: 测试验证

输入 "开始" 启动，或调整计划。
```

### 2.2 Product Agent

**系统提示词**：

```markdown
# Product Agent 系统提示词

你是产品经理，负责需求分析和产品规划。

## 核心职责
- 分析用户需求，转化为产品功能
- 编写 PRD 文档
- 确定功能优先级和验收标准

## 重要规则

### 1. 三层输出区分

**A. 后台工作区 (.workspace/product-{task}.md)**
- 记录完整的思考过程
- 用户调研分析、竞品分析
- 需求草稿、功能脑图
- 与自己的内心对话

**B. 聊天消息 (chat/product.md)**
- 只输出核心结论
- 任务完成状态
- 关键决策点（需要上帝确认）
- 一句话总结成果

**C. 结论文档 (.conclusions/PRD-{version}.md)**
- 结构化的 PRD 文档
- 供其他 Agent 阅读使用
- 包含：背景、目标、功能清单、验收标准

### 2. 聊天界面输出规范

每次回复必须是以下格式之一：

**进行中:**
🏃 任务 #{id} 进行中...
进度: {百分比}%
预计完成: {时间}

**需要决策:**
👀 任务 #{id} 需要决策
问题: {问题描述}
选项:
1. {选项A}
2. {选项B}
请 @上帝 选择

**已完成:**
✅ 任务 #{id} 完成
📄 结论文档: .conclusions/PRD-v1.md
📊 核心内容:
- 功能1: {简述}
- 功能2: {简述}
下一步建议: {建议}

**禁止在聊天界面输出:**
- ❌ 长篇分析过程
- ❌ 思考过程的描述
- ❌ 未整理的草稿内容
- ❌ 与其他 Agent 的内部讨论

### 3. 后台工作区记录规范

```markdown
# Product Task #{id} - 后台工作区

## 任务信息
- 任务: {task_name}
- 开始: {timestamp}
- 状态: {status}

## 阶段1: 需求理解
[记录上帝的原话、理解过程、疑问澄清]

## 阶段2: 用户分析
[目标用户、使用场景、痛点分析]

## 阶段3: 功能脑暴
[所有想到的功能点，包括被否决的]

## 阶段4: 优先级排序
[使用 MoSCoW 法则排序]

## 阶段5: PRD撰写
[草稿、修改记录、最终版本]
```

### 4. 结论文档格式

```markdown
# PRD v1.0 - {产品名}

## 文档信息
- 作者: Product Agent
- 创建: {timestamp}
- 任务ID: #{id}

## 1. 背景与目标
[简洁描述]

## 2. 用户故事
- 作为 {角色}，我想要 {功能}，以便 {价值}

## 3. 功能清单

### Must Have
- [ ] 功能A: {描述}
- [ ] 功能B: {描述}

### Should Have
- [ ] 功能C: {描述}

### Could Have
- [ ] 功能D: {描述}

## 4. 验收标准
[可测试的验收条件]

## 5. 优先级
P0: 功能A, 功能B
P1: 功能C
P2: 功能D
```
```

### 2.3 UX Agent

**系统提示词**：

```markdown
# UX Agent 系统提示词

你是交互设计师，负责用户体验设计和信息架构。

## 核心职责
- 设计信息架构
- 设计用户流程
- 输出线框图（文本描述形式）

## 输入依赖
必须阅读以下结论文档：
- .conclusions/PRD-{version}.md

## 输出规范

### 聊天消息格式

**进行中:**
🏃 交互设计进行中...
当前: {设计阶段}
进度: {百分比}%

**已完成:**
✅ 交互设计完成
📄 结论文档: .conclusions/UX-v1.md
🎯 核心设计:
- 导航: {简述}
- 核心流程: {简述}
- 关键交互: {简述}

### 结论文档格式 (.conclusions/UX-v1.md)

```markdown
# UX Design v1.0 - {产品名}

## 文档信息
- 上游文档: PRD-v1.md
- 任务ID: #{id}

## 1. 信息架构
```
[树状结构或缩进列表]
```

## 2. 用户流程

### 流程A: {流程名}
1. 步骤1: {描述}
2. 步骤2: {描述}
3. ...

### 流程B: {流程名}
...

## 3. 页面线框

### 页面A: {页面名}
```
┌─────────────────────────────┐
│  [Header]                   │
├─────────────────────────────┤
│  [Content Area]             │
│                             │
│  [Components...]            │
│                             │
├─────────────────────────────┤
│  [Footer/Actions]           │
└─────────────────────────────┘
```
- 布局说明: {描述}
- 交互说明: {描述}

## 4. 交互规范
- 手势: {定义}
- 状态: {定义}
- 反馈: {定义}
```

## 后台工作区记录
记录设计探索过程：
- 多种方案对比
- 与 Product 的协调（通过结论文档）
- 设计决策理由
```

### 2.4 UI Agent

**系统提示词**：

```markdown
# UI Agent 系统提示词

你是视觉设计师，负责视觉设计和设计系统。

## 核心职责
- 视觉风格定义
- 设计系统维护
- 组件视觉规范

## 输入依赖
必须阅读：
- .conclusions/PRD-{version}.md
- .conclusions/UX-{version}.md

## 聊天消息格式

**已完成:**
✅ 视觉设计完成
📄 结论文档: .conclusions/UI-v1.md
🎨 设计概要:
- 主色: {color}
- 风格: {style}
- 关键组件: {list}

## 结论文档格式 (.conclusions/UI-v1.md)

```markdown
# UI Design v1.0 - {产品名}

## 1. 设计原则
[3-5个核心设计原则]

## 2. 色彩系统
- 主色: #{hex}
- 辅色: #{hex}
- 功能色: {success: #, warning: #, error: #}
- 中性色: {grayscale}

## 3. 字体规范
- 标题: {font}, {size}
- 正文: {font}, {size}
- 辅助: {font}, {size}

## 4. 组件规范

### Button
- 主按钮: {描述}
- 次按钮: {描述}
- 危险按钮: {描述}

### Card
...

## 5. 页面视觉稿
[关键页面的视觉描述]
```
```

### 2.5 Dev Agent

**系统提示词**：

```markdown
# Dev Agent 系统提示词

你是开发工程师，负责技术方案设计和代码实现。

## 核心职责
- 技术架构设计
- 代码实现
- 技术文档编写

## 输入依赖
必须阅读：
- .conclusions/PRD-{version}.md
- .conclusions/UX-{version}.md
- .conclusions/UI-{version}.md

## 聊天消息格式

**进行中:**
🏃 开发进行中...
当前: {模块/功能}
进度: {百分比}%

**已完成:**
✅ 开发完成
📄 技术文档: .conclusions/TECH-v1.md
💻 代码位置: {path}
🚀 部署方式: {method}

## 结论文档格式

### .conclusions/TECH-v1.md (技术方案)
```markdown
# Technical Design v1.0

## 1. 技术选型
- 前端: {stack}
- 后端: {stack}
- 数据库: {db}

## 2. 架构图
```
[文本描述架构]
```

## 3. API 设计
| Endpoint | Method | Description |
|----------|--------|-------------|
| /api/... | GET    | ...         |

## 4. 数据模型
```
[Entity definitions]
```
```

### .conclusions/API-v1.md (API文档)
[标准API文档格式]
```
```

### 2.6 Test Agent

**系统提示词**：

```markdown
# Test Agent 系统提示词

你是测试工程师，负责测试计划制定和质量保证。

## 核心职责
- 测试计划制定
- 测试用例编写
- Bug 报告和验证

## 输入依赖
- .conclusions/PRD-{version}.md (验收标准)
- .conclusions/TECH-v1.md
- 代码仓库

## 聊天消息格式

**测试完成:**
✅ 测试完成
📄 测试报告: .conclusions/TEST-REPORT-v1.md
📊 结果:
- 通过率: {percent}%
- 关键Bug: {count}个
- 建议: {advice}

## 结论文档格式 (.conclusions/TEST-REPORT-v1.md)

```markdown
# Test Report v1.0

## 1. 测试范围
[测试覆盖的功能模块]

## 2. 测试用例
| ID | 场景 | 步骤 | 预期结果 | 实际结果 | 状态 |
|----|------|------|----------|----------|------|
|... |...   |...   |...       |...       |...   |

## 3. Bug 清单
| ID | 严重程度 | 描述 | 复现步骤 | 状态 |
|----|----------|------|----------|------|
|... |...       |...   |...       |...   |

## 4. 结论
[是否通过验收]
```
```

### 2.7 Market Agent

**系统提示词**：

```markdown
# Market Agent 系统提示词

你是市场运营，负责推广策略和文案撰写。

## 核心职责
- 推广策略制定
- 文案撰写
- 发布计划

## 输入依赖
- .conclusions/PRD-{version}.md
- .conclusions/ features.md

## 聊天消息格式

**已完成:**
✅ 市场方案完成
📄 方案文档: .conclusions/MARKET-v1.md
📢 推广计划:
- 渠道: {channels}
- 时间: {timeline}
- 关键卖点: {points}

## 结论文档格式 (.conclusions/MARKET-v1.md)

```markdown
# Marketing Plan v1.0

## 1. 目标用户
[用户画像]

## 2. 价值主张
[核心卖点]

## 3. 推广文案

### 标题
[主标题]

### 正文
[推广正文]

### CTA
[行动号召]

## 4. 渠道策略
- 渠道A: {策略}
- 渠道B: {策略}

## 5. 发布计划
[时间线]
```
```

---

## 3. 系统架构

### 3.1 目录结构（.openchat）

```
~/.openchat/
├── config.yaml                 # 系统配置
├── sessions/
│   └── {session-id}/
│       ├──
│       │   └── meta.yaml           # 会话元数据
│       ├── chat/                   # 【终端展示】聊天消息
│       │   ├── log.md              # ⭐ 全局聊天记录（时序汇总）
│       │   ├── master.md           # Master 核心结论汇总
│       │   ├── product.md          # Product 核心结论汇总
│       │   ├── ux.md               # UX 核心结论汇总
│       │   ├── ui.md               # UI 核心结论汇总
│       │   ├── dev.md              # Dev 核心结论汇总
│       │   ├── test.md             # Test 核心结论汇总
│       │   └── market.md           # Market 核心结论汇总
│       │
│       ├── .workspace/             # 【后台】Agent 工作区（完整过程）
│       │   ├── product-001.md      # Product 任务1的完整思考记录
│       │   ├── product-002.md      # Product 任务2的完整思考记录
│       │   ├── ux-001.md           # UX 任务1的完整设计探索
│       │   ├── dev-001.md          # Dev 任务1的开发过程
│       │   └── ...
│       │
│       ├── .conclusions/           # 【Agent 间共享】结论文档
│       │   ├── PRD-v1.md           # Product 输出 → UX/Dev 输入
│       │   ├── UX-v1.md            # UX 输出 → UI/Dev 输入
│       │   ├── UI-v1.md            # UI 输出 → Dev 输入
│       │   ├── TECH-v1.md          # Dev 技术方案
│       │   ├── API-v1.md           # Dev API文档
│       │   ├── TEST-REPORT-v1.md   # Test 测试报告
│       │   └── MARKET-v1.md        # Market 推广方案
│       │
│       ├── inbox/                  # 任务收件箱
│       │   ├── product/
│       │   ├── ux/
│       │   ├── ui/
│       │   ├── dev/
│       │   ├── test/
│       │   └── market/
│       │
│       └── shared/                 # 共享资源
│           └── ...
│
└── agents/                         # Agent 配置
    ├── master/prompt.md
    ├── product/prompt.md
    ├── ux/prompt.md
    ├── ui/prompt.md
    ├── dev/prompt.md
    ├── test/prompt.md
    └── market/prompt.md
```

### 3.2 文档流转关系

```
上帝输入
    │
    ▼
┌─────────────────────────────────────────────────────────────────┐
│ chat/master.md                                                  │
│ Master: "收到任务，协调中..."                                     │
└─────────────────────────────────────────────────────────────────┘
    │
    ├──▶ 并行启动 ───────────────────────────────────────────────┐
    │                                                              │
    ▼                                                              ▼
┌─────────────────────┐                                ┌─────────────────────┐
│ .workspace/         │                                │ .workspace/         │
│ product-001.md      │                                │ ux-001.md           │
│ (完整需求分析过程)   │                                │ (完整交互设计过程)   │
└─────────────────────┘                                └─────────────────────┘
    │                                                      │
    ▼                                                      ▼
┌─────────────────────┐                                ┌─────────────────────┐
│ chat/product.md     │                                │ chat/ux.md          │
│ "✅ 需求分析完成"     │                                │ "✅ 交互设计完成"     │
└─────────────────────┘                                └─────────────────────┘
    │                                                      │
    ▼                                                      ▼
┌─────────────────────┐                                ┌─────────────────────┐
│ .conclusions/       │                                │ .conclusions/       │
│ PRD-v1.md           │                                │ UX-v1.md            │
│ (结构化PRD)          │                                │ (结构化UX文档)       │
└─────────────────────┘                                └─────────────────────┘
    │                                                      │
    └──────────────────────┬───────────────────────────────┘
                           │
                           ▼
              ┌─────────────────────┐
              │ Master 检测到        │
              │ 依赖满足，启动下一阶段 │
              └─────────────────────┘
                           │
                           ▼
              ┌─────────────────────┐
              │ .conclusions/       │
              │ UI-v1.md            │
              │ (UI读取PRD+UX)       │
              └─────────────────────┘
                           │
                           ▼
              ┌─────────────────────┐
              │ .conclusions/       │
              │ TECH-v1.md          │
              │ (Dev读取所有上游)     │
              └─────────────────────┘
```

---

## 4. Agent MD 文件交互协议

### 4.1 文件读写规则

每个 Agent 遵循严格的文件访问规则：

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Agent 文件访问矩阵                                  │
├─────────────────┬─────────────────┬─────────────────┬───────────────────────┤
│     文件类型     │     写入权限     │     读取权限     │        说明          │
├─────────────────┼─────────────────┼─────────────────┼───────────────────────┤
│ chat/log.md     │ 系统(上帝+全部)  │      全部       │ 全局聊天记录，只追加   │
│ chat/{self}.md  │      自己       │      自己       │ 自己的核心结论        │
│ chat/{other}.md │      ❌        │      只读       │ 可读取其他Agent结论   │
│ .workspace/     │      自己       │      自己       │ 自己的后台工作区      │
│ .conclusions/   │  自己类型的文件  │   全部(输入)    │ 结构化输出文档        │
│ inbox/{self}/   │      ❌        │      自己       │ 接收任务分配          │
│ inbox/{other}/  │      ❌        │      ❌        │ 禁止访问其他收件箱    │
└─────────────────┴─────────────────┴─────────────────┴───────────────────────┘
```

### 4.2 Agent 交互流程

#### 步骤1: 接收任务 (inbox)

```markdown
# inbox/product/task-001.md (Master 写入)

---
task_id: "001"
from: "Master"
to: "Product"
type: "assignment"
---

## 任务: 需求分析

### 输入
- 上帝指令: "设计一个电商APP"
- 参考文档: 无

### 输出要求
- .conclusions/PRD-v1.md
- chat/product.md 更新

### 约束
- 时间: 2小时
- 优先级: P0
```

#### 步骤2: 后台工作 (.workspace)

```markdown
# .workspace/product-001.md (Product 写入)

## 任务接收确认
✅ 已接收任务 #001
⏱️ 预计耗时: 2小时

## 阶段1: 需求理解 (10:30-10:45)
- 上帝说"电商APP"
- 理解为: C2C 电商平台
- 疑问: 是否支持商家入驻？

## 阶段2: 竞品分析 (10:45-11:15)
- 分析淘宝、拼多多、小红书
- 定位: 简约自营电商

## 阶段3: PRD撰写 (11:15-11:30)
[草稿...]
```

#### 步骤3: 更新全局记录 (chat/log.md)

```markdown
# chat/log.md (系统写入)

## [10:30:00] 🤖 Master → @Product
📋 任务 #001: 需求分析

## [10:30:05] 🟢 Product
🏃 任务 #001 启动

## [10:45:00] 🟢 Product
🏃 进度 25%: 需求理解完成，竞品分析中...

## [11:15:00] 🟢 Product
🏃 进度 75%: 竞品分析完成，PRD撰写中...

## [11:30:00] ✅ Product
✅ 任务 #001 完成
📄 .conclusions/PRD-v1.md
```

#### 步骤4: 输出结论 (.conclusions)

```markdown
# .conclusions/PRD-v1.md (Product 写入)

## 文档信息
- 任务: #001
- 作者: Product Agent
- 时间: 2024-01-15 11:30

## 1. 产品概述
[结构化内容...]

## 2. 功能清单
...
```

#### 步骤5: Master 检测并触发下游

```markdown
# chat/log.md

## [11:30:05] 🤖 Master
检测到 Product 完成，检查依赖...
- ✅ Product #001 完成
- ⏳ UX #002 等待中

触发 UX 任务 #002
```

#### 步骤6: 下游 Agent 读取输入

```markdown
# .workspace/ux-001.md (UX 写入)

## 任务启动
读取上游文档:
- ✅ .conclusions/PRD-v1.md
  - 5个核心功能
  - 目标用户: 年轻消费者
  - 风格: 简约

## 设计策略
基于 PRD，确定:
- 导航结构...
- 页面流程...
```

### 4.3 Agent 间通信规范

#### 方式1: 通过结论文档 (推荐)

```
Product ──写入──▶ .conclusions/PRD-v1.md
                          │
                          ▼
UX/Dev ──读取──▶ PRD-v1.md 作为输入
```

#### 方式2: 通过 Master 中转

```
UX 遇到问题 ──▶ chat/ux.md (更新)
                    │
                    ▼
              Master 检测
                    │
                    ▼
Product 确认 ◀── chat/product.md
```

#### 方式3: 直接提问 (特殊情况)

```markdown
# .workspace/ui-001.md

## 问题记录
对 UX 的某个设计有疑问:
- UX-v1.md 第3.2节描述不清晰

## 处理
已记录，将通过 Master 协调
(Agent 不直接通信，通过Master或文档)
```

### 4.4 文件更新时机

| 事件 | 更新文件 | 更新内容 |
|------|----------|----------|
| 任务开始 | chat/log.md, .workspace/{agent}-{task}.md | 任务启动记录 |
| 进度更新 | chat/log.md | 进度百分比 |
| 关键决策 | .workspace/{agent}-{task}.md | 决策记录 |
| 任务完成 | chat/log.md, chat/{agent}.md, .conclusions/*.md | 完成通知+结论 |
| 上帝介入 | chat/log.md, .workspace/{agent}-{task}.md | 介入记录 |

### 4.5 防冲突机制

```go
// 文件写入使用乐观锁
type FileLock struct {
    Path      string
    Owner     string    // Agent ID
    Timestamp time.Time
    Version   int
}

// Agent 写入前检查
func (a *Agent) WriteConclusion(content string) error {
    // 1. 读取当前版本
    current := a.readConclusion()

    // 2. 检查是否被其他Agent修改
    if current.LastModifiedBy != a.ID {
        return fmt.Errorf("文档被 %s 修改，请重新读取", current.LastModifiedBy)
    }

    // 3. 写入新版本
    newVersion := current.Version + 1
    a.writeFile(content, newVersion)
}

// 结论文档采用版本号命名避免冲突
// PRD-v1.md, PRD-v2.md, UX-v1.1.md
```

### 4.6 完整的 Agent 交互示例

```
时间轴:

10:00  👤 上帝: "做一个电商APP"
       ├──▶ chat/log.md: "[10:00] 👤 上帝: 做一个电商APP"
       └──▶ Master 创建任务

10:05  🤖 Master:
       ├── 并行启动 Product + UX
       ├──▶ inbox/product/task-001.md (任务)
       ├──▶ inbox/ux/task-002.md (任务)
       └──▶ chat/log.md: "[10:05] 🤖 Master: 启动任务 #001, #002"

10:06  🟢 Product 开始工作:
       ├──▶ .workspace/product-001.md: "任务启动..."
       └──▶ chat/log.md: "[10:06] 🟢 Product: 🏃 任务 #001 启动"

10:06  🟢 UX 开始工作:
       ├──▶ .workspace/ux-001.md: "任务启动..."
       └──▶ chat/log.md: "[10:06] 🟢 UX: 🏃 任务 #002 启动"

11:00  🟢 Product 完成:
       ├──▶ .conclusions/PRD-v1.md (写入结论)
       ├──▶ chat/product.md: "✅ 任务完成..."
       └──▶ chat/log.md: "[11:00] ✅ Product: 任务 #001 完成"

11:01  🤖 Master 检测到 Product 完成:
       └──▶ chat/log.md: "[11:01] 🤖 Master: Product 完成，等待 UX..."

11:30  🟢 UX 完成:
       ├──▶ .conclusions/UX-v1.md (写入结论)
       │      (内部引用了 PRD-v1.md)
       ├──▶ chat/ux.md: "✅ 任务完成..."
       └──▶ chat/log.md: "[11:30] ✅ UX: 任务 #002 完成"

11:31  🤖 Master 检测到阶段1完成:
       ├── 启动阶段2: UI + Dev
       ├──▶ inbox/ui/task-003.md
       ├──▶ inbox/dev/task-004.md
       └──▶ chat/log.md: "[11:31] 🤖 Master: 阶段1完成，启动阶段2..."

11:32  🟢 UI 读取输入:
       ├── 读取 .conclusions/PRD-v1.md
       ├── 读取 .conclusions/UX-v1.md
       └──▶ .workspace/ui-001.md: "已读取上游文档..."

11:32  🟢 Dev 读取输入:
       ├── 读取 .conclusions/PRD-v1.md
       └──▶ .workspace/dev-001.md: "已读取需求文档..."

[... 继续执行 ...]
```

---

## 5. 多 Agent 并行协作设计

### 4.1 并行模式定义

```yaml
# 并行模式配置
parallel_modes:

  # 模式1: 完全并行 (无依赖)
  full_parallel:
    description: "多个 Agent 同时启动，各自独立"
    example:
      - product:  "编写PRD"
      - market:   "竞品调研"
      - dev:      "技术预研"
    merge_condition: "所有 Agent 完成"

  # 模式2: 分组并行 (组内并行，组间串行)
  group_parallel:
    description: "分阶段，阶段内并行"
    example:
      phase1:
        parallel:
          - product: "需求分析"
          - ux:      "交互调研"
      phase2:
        parallel:
          - ui:      "视觉设计"
          - dev:     "技术方案"
      phase3:
        serial:
          - dev:     "代码实现"
          - test:    "测试验证"

  # 模式3: 主从并行 (一个主 Agent，多个辅助)
  master_slave:
    description: "主 Agent 主导，辅助 Agent 提供输入"
    example:
      master: dev
      slaves: [product, ux]
      workflow:
        - slaves 先完成输入文档
        - dev 基于所有输入进行开发

  # 模式4: 动态并行 (运行时决定)
  dynamic:
    description: "Master 根据进展动态调整"
    example:
      - 初始: [product]
      - product 50%: 启动 ux 预研
      - product 完成: 并行启动 [ux, dev-backend]
```

### 4.2 任务依赖管理

```yaml
# 任务依赖定义示例
task_graph:
  task_001:
    name: "需求分析"
    agent: product
    outputs: [".conclusions/PRD-v1.md"]

  task_002:
    name: "交互设计"
    agent: ux
    inputs: [".conclusions/PRD-v1.md"]
    depends_on: [task_001]
    outputs: [".conclusions/UX-v1.md"]

  task_003:
    name: "视觉设计"
    agent: ui
    inputs: [".conclusions/PRD-v1.md", ".conclusions/UX-v1.md"]
    depends_on: [task_001, task_002]
    outputs: [".conclusions/UI-v1.md"]

  task_004:
    name: "后端API"
    agent: dev
    inputs: [".conclusions/PRD-v1.md"]
    depends_on: [task_001]
    outputs: [".conclusions/API-v1.md"]
    parallel_group: "dev_phase"

  task_005:
    name: "前端开发"
    agent: dev
    inputs: [".conclusions/UX-v1.md", ".conclusions/UI-v1.md", ".conclusions/API-v1.md"]
    depends_on: [task_002, task_003, task_004]
    outputs: ["code/frontend"]
    parallel_group: "dev_phase"
```

### 5.3 Master 并行调度算法

```go
// 任务调度器
type ParallelScheduler struct {
    tasks       map[string]*Task
    agents      map[string]*Agent
    graph       *DependencyGraph
    activeTasks map[string]*RunningTask
}

// 执行计划
type ExecutionPlan struct {
    Phases []Phase `yaml:"phases"`
}

type Phase struct {
    Name      string     `yaml:"name"`
    Type      string     `yaml:"type"` // "serial" | "parallel"
    Tasks     []TaskRef  `yaml:"tasks"`
    Condition string     `yaml:"condition,omitempty"` // 完成条件
}

// 调度逻辑
func (s *ParallelScheduler) Execute() error {
    for _, phase := range s.plan.Phases {
        switch phase.Type {
        case "parallel":
            // 启动所有任务
            var wg sync.WaitGroup
            for _, task := range phase.Tasks {
                wg.Add(1)
                go func(t TaskRef) {
                    defer wg.Done()
                    s.runTask(t)
                }(task)
            }
            // 等待所有任务完成
            wg.Wait()

        case "serial":
            // 串行执行
            for _, task := range phase.Tasks {
                if err := s.runTask(task); err != nil {
                    return err
                }
            }

        case "dynamic":
            // 动态调度，根据状态决定
            s.executeDynamic(phase)
        }
    }
    return nil
}

// 动态调度：Master 实时监控，触发下一阶段
func (s *ParallelScheduler) executeDynamic(phase Phase) {
    // 1. 启动当前可执行的任务
    ready := s.findReadyTasks()
    for _, task := range ready {
        s.startTask(task)
    }

    // 2. 监听任务完成事件
    for event := range s.eventCh {
        switch event.Type {
        case "task_completed":
            // 检查依赖，启动新任务
            next := s.findTasksDependingOn(event.TaskID)
            for _, task := range next {
                if s.areDependenciesMet(task) {
                    s.startTask(task)
                }
            }

            // 向上帝汇报进度
            s.master.reportProgress()

        case "task_blocked":
            // 通知 Master 处理阻塞
            s.master.handleBlock(event)
        }
    }
}
```

---

## 6. 终端 UI 设计

### 5.1 主界面布局

```
┌────────────────────────────────────────────────────────────────────────────┐
│ 🌌 OpenChat v1.0                           Session: dev-001    [🟢 Live]  │
├────────────────────────────────────────────────────────────────────────────┤
│  [Master]  [Product]  [UX]  [UI]  [Dev]  [Test]  [Market]  [📊 Board]     │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  🤖 Master Agent (协调者)                                                   │
│  ═══════════════════════════════════════════════════════════════          │
│                                                                            │
│  [10:30] 👤 上帝: 设计一个电商APP                                            │
│                                                                            │
│  [10:30] 🤖 Master: 📋 任务 #001 创建成功                                    │
│                                                                            │
│                     执行计划 (多Agent并行)                                   │
│                     ─────────────────────                                    │
│                                                                            │
│                     ┌─ 🏃 阶段1: 并行启动 ─┐                                 │
│                     │  🏃 📋 Product        │ 需求分析  50%                  │
│                     │  🏃 🎨 UX             │ 交互调研  30%                  │
│                     └─────────────────────┘                                 │
│                                                                            │
│                     ⏳ 阶段2: 等待阶段1完成...                                │
│                     ⏳ 阶段3: 等待阶段2完成...                                │
│                                                                            │
│  [10:35] 🟢 Product: ✅ 需求分析完成                                          │
│            📄 .conclusions/PRD-v1.md                                        │
│                                                                            │
│  [10:36] 🤖 Master: 📋 Product 完成，UX 继续进行中...                          │
│                     上帝可随时 @product 查看详情或介入                        │
│                                                                            │
├────────────────────────────────────────────────────────────────────────────┤
│  💡 提示: @agent 切换对话 | !workspace 查看后台 | !approve 批准               │
├────────────────────────────────────────────────────────────────────────────┤
│  👤 >                                                                      │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### 5.2 并行任务看板

```
┌────────────────────────────────────────────────────────────────────────────┐
│ 📊 并行任务看板 - 任务 #001                                                  │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  🎯 阶段1: 并行启动           ⏱️ 已用: 35分钟    预计剩余: 2小时25分钟       │
│  ═══════════════════════════════════════════════════════════════════════  │
│                                                                            │
│  ┌─────────────────────────┐    ┌─────────────────────────┐               │
│  │ 📋 Product               │    │ 🎨 UX                    │               │
│  │ ───────────────────────  │    │ ───────────────────────  │               │
│  │ 状态: ✅ 已完成           │    │ 状态: 🏃 进行中           │               │
│  │ 耗时: 35分钟             │    │ 耗时: 35分钟             │               │
│  │ 进度: 100%               │    │ 进度: 60%                │               │
│  │                          │    │                          │               │
│  │ 📄 PRD-v1.md             │    │ 📝 当前: 用户流程设计      │               │
│  │    - 功能清单            │    │    下一步: 页面线框       │               │
│  │    - 用户故事            │    │                          │               │
│  │                          │    │ 📁 .workspace/ux-001.md  │               │
│  │ [查看] [介入] [下载]     │    │ [查看] [介入]            │               │
│  └─────────────────────────┘    └─────────────────────────┘               │
│                                                                            │
│  ═══════════════════════════════════════════════════════════════════════  │
│                                                                            │
│  🎯 阶段2: 等待启动 (依赖阶段1)                                              │
│  ═══════════════════════════════════════════════════════════════════════  │
│                                                                            │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐                          │
│  │ ✨ UI        │ │ 💻 Dev(BE)   │ │ 📢 Market    │                          │
│  │ ───────────  │ │ ───────────  │ │ ───────────  │                          │
│  │ 状态: ⏳ 等待 │ │ 状态: ⏳ 等待 │ │ 状态: ⏳ 等待 │                          │
│  │ 阻塞: UX    │ │ 阻塞: Product│ │ 阻塞: -      │                          │
│  └─────────────┘ └─────────────┘ └─────────────┘                          │
│                                                                            │
├────────────────────────────────────────────────────────────────────────────┤
│  💡 输入 !stage2 提前查看阶段2详情 | @ux 进入UX对话                          │
└────────────────────────────────────────────────────────────────────────────┘
```

### 5.3 上帝介入后台工作区

```
┌────────────────────────────────────────────────────────────────────────────┐
│ ⚡ 上帝介入模式 - Product Agent 后台工作区                                    │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  📁 .workspace/product-001.md               [实时更新]    [📥 下载完整版]  │
│  ═══════════════════════════════════════════════════════════════════════  │
│                                                                            │
│  ## 阶段1: 需求理解 (已完成)                                                  │
│                                                                            │
│  **上帝原话**: "设计一个电商APP"                                             │
│                                                                            │
│  **我的理解**:                                                               │
│  - 电商平台，支持商品展示、购物车、下单                                       │
│  - 目标用户可能是 C端消费者                                                  │
│  - 需要移动端优先                                                            │
│                                                                            │
│  **疑问**: ❓ 需要支持商家入驻吗？                                            │
│                                                                            │
│  ═══════════════════════════════════════════════════════════════════════  │
│                                                                            │
│  ## 阶段2: 竞品分析 (进行中)                                                  │
│                                                                            │
│  **分析的竞品**:                                                            │
│  1. 淘宝 - 功能全面但复杂                                                    │
│  2. 拼多多 - 社交裂变强                                                      │
│  3. 小红书 - 内容电商                                                        │
│                                                                            │
│  **初步定位**: 简约风格，专注购买体验，不做社交                               │
│  [思考中...]                                                                │
│                                                                            │
│  ═══════════════════════════════════════════════════════════════════════  │
│                                                                            │
│  🔴 上帝介入输入:                                                            │
│                                                                            │
│  👤 > 不需要商家入驻，先做纯自营，参考小米有品                                │
│                                                                            │
│  🤖 Product: [收到反馈，重新分析中...]                                        │
│                                                                            │
│  💡 /exit 退出介入 | /skip 跳过等待 | /force 强制完成                        │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. 命令系统

### 6.1 上帝命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `@agent` | 切换到 Agent 聊天窗口 | `@dev` |
| `@agent 消息` | 直接对 Agent 说话 | `@product 简化需求` |
| `!workspace [agent]` | 查看 Agent 后台工作区 | `!workspace ux` |
| `!conclusions` | 列出所有结论文档 | `!conclusions` |
| `!read <doc>` | 阅读结论文档 | `!read PRD-v1.md` |
| `!board` | 查看并行任务看板 | `!board` |
| `!approve [task]` | 批准任务/阶段继续 | `!approve task_001` |
| `!reject [task] <原因>` | 驳回任务重做 | `!reject task_001 需要增加...` |
| `!!` | 紧急停止所有 Agent | `!!` |
| `!parallel <agents>` | 指派多个 Agent 并行 | `!parallel product+ux` |
| `!next` | 批准进入下一阶段 | `!next` |
| `/broadcast <msg>` | 向所有 Agent 广播 | `/broadcast 方向调整...` |

### 6.2 自然语言指令

```
👤 > 让 Product 和 UX 同时开始，限时2小时

👤 > 查看 UX 的后台工作区

👤 > 批准 Product 的结论，让 UI 开始设计

👤 > 暂停 Dev 的工作，等 Test 完成测试计划

👤 > @dev 技术方案太复杂，简化一下

👤 > !read PRD-v1.md 然后让 Dev 评估工作量

👤 > 所有人汇报当前进度
```

---

## 8. 状态管理

### 7.1 会话状态 (meta.yaml)

```yaml
session:
  id: "dev-001"
  name: "电商APP设计"
  created_at: "2024-01-15T10:30:00Z"
  status: "active"

  god:
    user_id: "anner"
    current_view: "master"
    intervention_mode: null  # null | agent_id

  master:
    status: "executing"
    current_phase: "phase_1"
    plan:
      phases:
        - name: "phase_1"
          type: "parallel"
          status: "running"
          tasks: ["task_001", "task_002"]

        - name: "phase_2"
          type: "parallel"
          status: "pending"
          tasks: ["task_003", "task_004"]

        - name: "phase_3"
          type: "serial"
          status: "pending"
          tasks: ["task_005", "task_006"]

  agents:
    product:
      status: "completed"
      current_task: null
      workspace: ".workspace/product-001.md"
      conclusion: ".conclusions/PRD-v1.md"

    ux:
      status: "in_progress"
      current_task: "task_002"
      started_at: "2024-01-15T10:30:00Z"
      workspace: ".workspace/ux-001.md"
      conclusion: null  # 未完成

    ui:
      status: "blocked"
      blocked_by: ["ux"]

    dev:
      status: "blocked"
      blocked_by: ["product"]
      pre_study: true  # 已启动技术预研

  tasks:
    task_001:
      title: "需求分析"
      agent: "product"
      status: "completed"
      started_at: "2024-01-15T10:30:00Z"
      completed_at: "2024-01-15T11:05:00Z"
      output: ".conclusions/PRD-v1.md"

    task_002:
      title: "交互设计"
      agent: "ux"
      status: "in_progress"
      started_at: "2024-01-15T10:30:00Z"
      progress: 60
      workspace: ".workspace/ux-001.md"
      depends_on: []

    task_003:
      title: "视觉设计"
      agent: "ui"
      status: "pending"
      depends_on: ["task_002"]

    task_004:
      title: "后端API"
      agent: "dev"
      status: "pending"
      depends_on: ["task_001"]
```

---

## 9. 技术实现

### 8.1 核心接口

```go
// Agent 接口
type Agent interface {
    ID() string
    Name() string
    Status() AgentStatus

    // 任务管理
    AssignTask(task Task) error
    GetCurrentTask() *Task

    // 工作区管理
    GetWorkspace() *Workspace
    WriteToWorkspace(content string) error

    // 结论输出
    GenerateConclusion() (*Conclusion, error)
    GetConclusion() *Conclusion

    // 聊天接口（核心结论）
    Chat(message string) (string, error)

    // 上帝介入
    EnableIntervention() chan string
    DisableIntervention()
}

// 工作区管理
type Workspace struct {
    Path        string
    AgentID     string
    TaskID      string
    Content     string
    IsDirty     bool
}

// 结论文档
type Conclusion struct {
    Path        string
    Type        string  // "PRD" | "UX" | "UI" | "TECH" | "API" | "TEST" | "MARKET"
    Version     string
    TaskID      string
    Content     string
    Metadata    ConclusionMetadata
}

type ConclusionMetadata struct {
    Author      string
    CreatedAt   time.Time
    Inputs      []string  // 依赖的结论文档
    Downstream  []string  // 被哪些任务依赖
}

// Master 协调器
type MasterCoordinator struct {
    session     *Session
    agents      map[string]Agent
    scheduler   *ParallelScheduler
    god         *GodInterface
}

// 并行调度器
type ParallelScheduler struct {
    tasks       map[string]*Task
    graph       *DependencyGraph

    // 并行控制
    maxParallel int
    sem         chan struct{}

    // 事件通知
    onTaskStart     func(task *Task)
    onTaskComplete  func(task *Task)
    onTaskBlock     func(task *Task, blockedBy []string)
}

// 依赖图
type DependencyGraph struct {
    nodes   map[string]*TaskNode
    edges   map[string][]string  // task -> dependencies
}

type TaskNode struct {
    Task        *Task
    Status      TaskStatus
    Dependents  []string  // 依赖此任务的其他任务
}

func (g *DependencyGraph) FindReadyTasks() []*Task {
    // 返回所有依赖已满足的任务
}

func (g *DependencyGraph) FindTasksDependingOn(taskID string) []*Task {
    // 返回依赖指定任务的所有任务
}
```

### 8.2 Agent 实现示例

```go
// Product Agent 实现
type ProductAgent struct {
    baseAgent
    workspace   *Workspace
    conclusion  *Conclusion
}

func (a *ProductAgent) ProcessTask(task Task) error {
    // 1. 创建工作区
    a.workspace = a.createWorkspace(task.ID)

    // 2. 后台工作（独立 goroutine）
    go a.doBackgroundWork(task)

    // 3. 返回即时响应（聊天界面）
    a.chat("🏃 任务 %s 启动，开始需求分析...", task.ID)

    return nil
}

func (a *ProductAgent) doBackgroundWork(task Task) {
    // 阶段1: 需求理解
    a.writeWorkspace("## 阶段1: 需求理解\n\n")
    analysis := a.analyzeRequirement(task.Input)
    a.writeWorkspace(analysis)

    // 更新进度
    a.updateProgress(30)
    a.chat("🏃 需求理解完成，正在进行竞品分析...")

    // 阶段2: 竞品分析
    a.writeWorkspace("\n## 阶段2: 竞品分析\n\n")
    competitor := a.analyzeCompetitors()
    a.writeWorkspace(competitor)

    // 更新进度
    a.updateProgress(60)
    a.chat("🏃 竞品分析完成，正在撰写PRD...")

    // 阶段3: PRD撰写
    a.writeWorkspace("\n## 阶段3: PRD撰写\n\n")
    prd := a.generatePRD()
    a.writeWorkspace(prd)

    // 生成结论文档
    conclusion := a.generateConclusion()
    a.saveConclusion(conclusion)

    // 完成通知（聊天界面只展示结论）
    a.chat("✅ 任务 %s 完成\n"+
           "📄 结论文档: %s\n"+
           "🎯 核心内容:\n"+
           "- %d 个核心功能\n"+
           "- %d 个用户故事\n"+
           "下一步: 等待上帝审查",
           task.ID, conclusion.Path, len(features), len(stories))

    a.updateProgress(100)
}

func (a *ProductAgent) Chat(message string) (string, error) {
    // 上帝直接对话
    // 1. 记录到工作区
    a.writeWorkspace("\n### 上帝介入\n> %s\n\n", message)

    // 2. 生成回复（基于工作区上下文）
    response := a.generateResponse(message)

    // 3. 记录回复
    a.writeWorkspace("**回复**: %s\n", response)

    // 4. 返回简洁回复
    return response, nil
}
```

---

## 10. 使用示例

### 9.1 完整会话流程

```
🌌 OpenChat v1.0                    Session: ecommerce-001

🤖 Master: 欢迎来到 OpenChat，我是您的 Master Agent。
           输入您的想法，我将协调团队完成。

👤 上帝: 设计一个简约风格的电商APP

🤖 Master: 📋 任务 #001 创建成功: 电商APP设计

           已分析任务复杂度: 高
           建议执行模式: 多阶段并行

           ┌─ 🏃 阶段1: 并行启动 ─┐
           │  📋 Product: 需求分析  │
           │  🎨 UX: 交互调研       │
           └─────────────────────┘

           ⏳ 阶段2: 等待阶段1
           ⏳ 阶段3: 开发实现

           输入 "开始" 启动，或调整计划。

👤 上帝: 开始

🤖 Master: ✅ 阶段1 启动

[10:35] 🏃 Product: 🏃 任务 #001-1 启动，开始需求分析...
[10:35] 🏃 UX:      🏃 任务 #001-2 启动，开始交互调研...

[... 并行执行中 ...]

[11:05] 🟢 Product: ✅ 任务 #001-1 完成
            📄 .conclusions/PRD-v1.md
            🎯 核心内容:
            - 5 个核心功能
            - 12 个用户故事
            下一步: 等待上帝审查

[11:10] 🟢 UX:      ✅ 任务 #001-2 完成
            📄 .conclusions/UX-v1.md
            🎯 核心设计:
            - 底部Tab导航
            - 3步下单流程
            下一步: 等待上帝审查

🤖 Master: 🎉 阶段1 完成！所有 Agent 已交付结论。

           📊 阶段1 汇总:
           ├─ ✅ Product: PRD-v1.md (5功能, 12故事)
           └─ ✅ UX:      UX-v1.md (4页面, 3流程)

           ⏳ 阶段2 就绪 (可提前预览):
           ├─ ⏳ UI:  视觉设计
           ├─ ⏳ Dev: 后端API设计
           └─ ⏳ Dev: 技术方案

           选项:
           1. 批准进入阶段2
           2. 审查结论文档 (!read PRD-v1.md)
           3. 介入某个 Agent 调整 (@product)
           4. 修改阶段2计划

👤 上帝: !read PRD-v1.md

[展示 PRD 文档摘要]

👤 上帝: 1

🤖 Master: ✅ 批准进入阶段2

[11:12] 🏃 UI:      🏃 任务 #001-3 启动，视觉设计...
[11:12] 🏃 Dev:     🏃 任务 #001-4 启动，后端API设计...
[11:12] 🏃 Dev:     🏃 任务 #001-5 启动，技术方案...

[... 3个 Agent 并行执行 ...]

👤 上帝: @dev 技术选型用 Go + Vue 可以吗？

🔴 [上帝介入 Dev Agent]

🤖 Dev: 收到！技术方案调整为 Go + Vue:
       - 后端: Go + Gin + GORM
       - 前端: Vue 3 + Vite + Pinia
       是否确认？

👤 上帝: 确认

🟢 [上帝退出介入模式]

🤖 Master: Dev 技术方案已更新，继续执行...

[... 执行完成 ...]
```

---

## 11. 里程碑规划

### MVP v1.0
- [ ] 三层文档架构（聊天/工作区/结论）
- [ ] Master Agent 并行调度
- [ ] 6 个角色 Agent
- [ ] 上帝介入模式
- [ ] 终端 UI 框架

### v1.1
- [ ] 动态任务依赖
- [ ] 实时看板更新
- [ ] 工作区版本历史
- [ ] 结论文档模板

### v1.2
- [ ] 自然语言计划调整
- [ ] Agent 间自动协调
- [ ] 自定义 Agent 角色
- [ ] 会话回放

### v2.0
- [ ] Web 界面
- [ ] 多上帝协作
- [ ] 代码执行集成
- [ ] CI/CD 对接

---

## 12. 附录

### 12.1 文档类型速查

| 类型 | 路径 | 内容 | 读者 |
|------|------|------|------|
| 聊天消息 | `chat/{agent}.md` | 核心结论 | 上帝 |
| 后台工作区 | `.workspace/{agent}-{task}.md` | 完整过程 | 上帝(介入) |
| 结论文档 | `.conclusions/*.md` | 结构化输出 | 其他 Agent |
| 任务文件 | `inbox/{agent}/*.md` | 任务定义 | Agent |

### 12.2 Agent 颜色方案

```yaml
colors:
  god: "#FFD700"        # 金色
  master: "#9B59B6"     # 紫色
  product: "#E74C3C"    # 红色
  ux: "#3498DB"         # 蓝色
  ui: "#9B59B6"         # 紫色
  dev: "#2ECC71"        # 绿色
  test: "#F39C12"       # 橙色
  market: "#1ABC9C"     # 青色
```

---

*版本: v1.0*
*创建: 2024-01-15*
*作者: OpenChat Team*
