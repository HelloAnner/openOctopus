# Session 模块 001 阶段 PRD

## 1. 阶段定位

`session 001` 不追求一次实现完整的恢复、调度和多角色协作系统，而是先把 **session 作为文件系统载体的基线** 做实。

在 `config 001` 已经提供稳定 `RuntimeConfig` 的前提下，`session 001` 要解决的问题很明确：当用户执行 `openoctopus run --config ...` 且配置校验通过后，系统必须创建一个 **结构稳定、内容可追溯、后续模块可直接复用** 的 `.octopus/sessions/{session_id}` 工作目录。

这意味着本阶段要从“当前只创建一个 `metadata.md` 的轻量 skeleton”升级到“创建标准目录树 + 初始化关键 Markdown/YAML 文件 + 写入首个状态快照与配置快照”的首版会话基座。

`001` 的目标不是把 session 做复杂，而是给 `event-bus`、`orchestrator`、`role-runtime`、`recovery` 提供一个统一起点，避免后续模块各自发明目录和文件协议。

## 2. 本阶段要解决的问题

### 2.1 当前问题

当前代码里的 session 创建逻辑只有一个非常薄的能力：

- 生成 `session_id`
- 创建 `.octopus/sessions/{id}`
- 写一个最小版 `metadata.md`

这个实现能证明“命令执行到了 session 创建阶段”，但还无法支撑后续模块真正接入，主要问题有：

1. **目录协议不完整**：没有 `session.state.md`、`timeline.md`、`state/checkpoints/`、`planner/`、`bus/` 等约定结构。
2. **配置不可追溯**：没有把有效配置快照落盘，后续无法复盘“这次 session 到底按什么配置启动”。
3. **状态基线缺失**：没有首个当前态和首个 checkpoint，恢复模块没有统一起点。
4. **失败回滚不明确**：如果创建过程中部分文件写成功、部分失败，容易留下半初始化目录。
5. **模块边界不清楚**：session、event-bus、orchestrator 对哪些文件负责创建、哪些文件负责更新，还没有明确分层。

### 2.2 001 阶段目标

`session 001` 的目标是建立一个简单但足够稳定的首版协议：

1. 为每次合法 `run` 创建唯一 session 目录。
2. 初始化后续模块要依赖的基础目录与占位文件。
3. 持久化 `metadata.md`、`session.state.md`、`timeline.md`、首个 checkpoint 和有效配置快照。
4. 保证写入顺序、原子替换与失败回滚可预期。
5. 向 CLI 和后续模块暴露清晰的创建结果对象，而不是让调用方自己拼路径。

## 3. 范围定义

### 3.1 001 范围内

- 基于已通过 `config 001` 校验的 `RuntimeConfig` 创建 session。
- 按 `runtime.workspace.sessions_dir` 解析并创建 session 目录。
- 初始化标准目录骨架和首批占位文件。
- 写入 `metadata.md`、`session.state.md`、`timeline.md`。
- 写入 `state/effective_config.yaml` 与 `state/checkpoints/0000-init.md`。
- 提供统一的原子写文件与失败回滚策略。
- 返回 `session_id`、`session_dir`、关键文件路径等创建结果。

### 3.2 001 范围外

- 不负责主 Agent 调度与 `master_schedule.md` 的业务填充。
- 不负责 `bus/events.md` 的持续事件追加和偏移推进。
- 不负责 `roles/{role_id}/` 子目录创建、回合文件写入和心跳刷新。
- 不负责 `resume`、`stop`、`interrupt`、`reset-session` 等 CLI 命令。
- 不负责增量 checkpoint、回放重建、版本化 artifact 管理。
- 不引入数据库、对象存储或额外守护进程。

## 4. 核心用户故事

### 4.1 命令行用户

- 作为用户，我希望 `run` 成功后能明确看到 session 路径，方便我立即检查 `.octopus/sessions/{id}`。
- 作为用户，我希望就算后续模块还没完全接入，也能在 session 目录里看到完整的初始化痕迹，而不是只有一个零散元数据文件。
- 作为用户，我希望创建失败时不要留下脏目录，避免误判为“已经启动成功”。

### 4.2 下游模块

- 作为 `event-bus`，我希望 session 初始化后就有稳定的 `bus/` 目录和目标文件位置，后续只需要按协议追加内容。
- 作为 `orchestrator`，我希望 `planner/` 与 `session.state.md` 已存在，可以直接开始推进当前态与排程文件。
- 作为 `role-runtime`，我希望 session 根目录、`roles/`、`artifacts/`、`state/checkpoints/` 等基础路径是确定的，不需要在运行时临时补目录。
- 作为 `recovery`，我希望一开始就能拿到有效配置快照和初始化 checkpoint，便于后续恢复链路建立统一入口。

### 4.3 复盘与排障

- 作为排障者，我希望通过 `metadata.md` 就能看到 `workflow_id`、配置来源、创建时间、状态和快照位置。
- 作为复盘者，我希望 `timeline.md` 能从 session 创建时刻就开始记录，而不是后续某个模块再补第一条记录。

## 5. 核心方案

### 5.1 创建入口与模块接口

`session 001` 的创建入口只接收“已经过配置加载与静态校验”的输入，不再直接读取 YAML。

推荐接口形态：

```go
type CreateOptions struct {
    Config          model.RuntimeConfig
    ConfigPath      string
    AppliedDefaults []defaults.AppliedDefault
}

type CreateResult struct {
    SessionID           string
    SessionDir          string
    MetadataPath        string
    StatePath           string
    TimelinePath        string
    EffectiveConfigPath string
    InitialCheckpoint   string
}
```

约束如下：

1. `session` 模块只消费强类型 `RuntimeConfig`，不自己重读原始 YAML。
2. CLI `run` 在配置错误时必须继续维持“先失败、后退出、不创建 session”的行为。
3. CLI `run` 在配置成功时，只调用 session 创建入口，不自己拼目录或手工写文件。

### 5.2 Session ID 与路径解析规则

首版保持简单直接：

- `session_id` 格式：`sess_{unix_nano}`
- 实际创建目录：`{resolved_sessions_dir}/{session_id}`

路径解析规则：

1. `runtime.workspace.sessions_dir` 为绝对路径时，直接使用。
2. `runtime.workspace.sessions_dir` 为相对路径时，相对 `dirname(configPath)` 解析。
3. `runtime.workspace.root` 在 `001` 阶段主要用于记录工作区语义根路径，不作为 session 真实落盘路径的二次推导来源。

这样做的原因是：`config 001` 已经把 `sessions_dir` 作为显式配置字段与默认值字段产出，`session 001` 不再引入额外的隐式推导，避免路径决策重复分散在多个模块。

### 5.3 目录骨架与初始化文件

`session 001` 需要创建如下首版目录树：

```text
.octopus/sessions/{session_id}/
├── metadata.md
├── session.state.md
├── timeline.md
├── planner/
│   ├── requirement.snapshot.md
│   ├── human_messages.md
│   ├── master_schedule.md
│   ├── global_progress.md
│   └── blockers.md
├── bus/
│   ├── events.md
│   ├── interrupts.md
│   ├── offsets.md
│   └── lock.md
├── roles/
├── artifacts/
│   └── index.md
├── state/
│   ├── effective_config.yaml
│   └── checkpoints/
│       └── 0000-init.md
└── audit/
    ├── lineage.md
    └── replay.md
```

初始化原则：

1. **目录先行**：先把稳定目录骨架建好。
2. **关键文件必须有内容**：`metadata.md`、`session.state.md`、`timeline.md`、`effective_config.yaml`、`0000-init.md` 不能是空文件。
3. **占位文件允许极简**：`planner/`、`bus/`、`audit/` 下的占位文件只写标题和“initialized by session 001”说明，不写业务内容。
4. **角色目录延后**：`roles/` 目录只建根目录，不在 `001` 阶段预创建每个角色子目录，避免 session 模块过早承担 role-runtime 责任。

### 5.4 关键文件协议

#### A. `metadata.md`

必须至少包含：

- `session_id`
- `workflow_id`
- `workflow_name`
- `status`
- `created_at`
- `config_path`
- `sessions_dir`
- `workspace_root`
- `effective_config_path`
- `effective_config_hash`
- `applied_defaults_count`

其中：

- `status` 初始值固定为 `INITIAL`
- `effective_config_hash` 使用 `RuntimeConfig` 的稳定序列化结果计算 `sha256`
- `applied_defaults_count` 用于让用户知道本次启动是否包含默认值注入

#### B. `session.state.md`

必须记录当前 session 的“可覆盖最新态”，至少包括：

- `session_id`
- `status: INITIAL`
- `current_stage_id: ""`
- `current_role_id: ""`
- `checkpoint_seq: 0`
- `last_event: SESSION_CREATED`
- `created_at`
- `updated_at`

#### C. `timeline.md`

首条记录必须由 session 模块写入，表示本次 session 已创建完成。推荐事件名：`SESSION_CREATED`。

#### D. `state/effective_config.yaml`

写入本次启动使用的完整 `RuntimeConfig` 序列化结果。该文件是 session 对 config 001 的直接承接物，用于后续：

- 恢复
- 复盘
- 排查“默认值是否生效”
- 验证 session 与配置输入的一致性

#### E. `state/checkpoints/0000-init.md`

这是首版初始 checkpoint，至少记录：

- `session_id`
- `status`
- `created_at`
- `effective_config_path`
- `timeline_head`

后续 checkpoint 序号从 `0001-*.md` 继续增长。

### 5.5 写入顺序、原子替换与失败回滚

`session 001` 必须显式定义写入顺序，避免“目录存在但状态不完整”的半成品：

1. 创建 `sessions_dir`
2. 创建 `session_dir`
3. 创建全部子目录
4. 写 `state/effective_config.yaml`
5. 写 `metadata.md`
6. 写 `session.state.md`
7. 写 `timeline.md`
8. 写 `state/checkpoints/0000-init.md`
9. 写其余占位文件

写文件规则：

- 覆盖类文件统一使用 `*.tmp` 写入后 `rename`
- Markdown 内容写入失败即停止后续步骤
- 任一关键步骤失败时，删除整个 `session_dir`
- 返回错误给 CLI，由 CLI 输出失败信息

首版不做复杂事务系统，只做“失败即整目录回滚”的简单策略，保证外部观察结果清晰：**要么成功得到完整 session，要么什么都没有**。

### 5.6 模块边界

`session 001` 与其他模块的责任边界如下：

| 模块 | 负责内容 | 不负责内容 |
| --- | --- | --- |
| `session` | 会话目录、初始化文件、配置快照、首个状态与 checkpoint | 调度算法、事件推进、角色执行 |
| `event-bus` | `bus/events.md`、`interrupts.md` 等持续写入与消费位点推进 | session 根目录创建 |
| `orchestrator` | `planner/master_schedule.md`、`global_progress.md` 等业务内容更新 | session 初始化 |
| `role-runtime` | `roles/{role_id}` 目录、turns 文件、心跳与结论更新 | session 根目录协议定义 |
| `recovery` | 从 `events/checkpoints/state` 重建运行态 | 初始 session 创建 |

边界原则只有一句话：**session 只提供“载体和起点”，不替下游模块做业务推进。**

## 6. 分阶段交付物

### 6.1 必交付物

1. `docs/session/001/prd.md`
2. `docs/session/001/e2e.md`
3. `internal/session` 的首版创建实现
4. `cmd/openoctopus/run.go` 对 session 创建结果的接入
5. 至少一组命令级测试，验证 `run` 成功后创建完整骨架

### 6.2 建议代码落位

建议保持轻量分文件，不把所有逻辑塞回单文件：

- `internal/session/types.go`：创建输入输出模型
- `internal/session/path.go`：路径解析与 `session_id` 生成
- `internal/session/files.go`：模板内容构造、原子写入、回滚
- `internal/session/create.go`：创建流程编排入口
- `internal/session/create_test.go`：session 创建单测

CLI 侧保持最小变更：

- `cmd/openoctopus/run.go`

## 7. 验收标准

以下条件同时满足，才视为 `session 001` 达标：

1. `run` 在合法配置下会创建唯一 session 目录，并输出最终路径。
2. session 目录中存在本 PRD 定义的关键目录与关键文件。
3. `metadata.md`、`session.state.md`、`timeline.md`、`0000-init.md` 的关键信息一致。
4. `state/effective_config.yaml` 成功落盘，且可以反映 `config 001` 处理后的有效配置。
5. 任一写入失败不会留下半初始化 session 目录。
6. 连续两次 `run` 不会覆盖已有 session。

## 8. 依赖关系与风险

### 8.1 上游依赖

- `config 001`：提供合法 `RuntimeConfig` 和 `AppliedDefaults`
- `cli` 当前 `run` 入口：负责先校验、后创建 session

### 8.2 下游影响

- `event-bus 001` 将直接复用 `bus/` 路径与文件约定
- `orchestrator 001` 将直接复用 `planner/` 与 `session.state.md`
- `role-runtime 001` 将直接复用 `roles/`、`artifacts/`、`state/checkpoints/`
- `recovery 001` 将直接复用有效配置快照和初始 checkpoint

### 8.3 主要风险

1. **文件过多但语义过早固化**：首版占位文件如果定义得过深，后续模块可能被不必要束缚。
2. **路径理解不一致**：`workspace.root` 和 `sessions_dir` 的职责如果不写清楚，后续容易出现“创建到了意外目录”的问题。
3. **回滚不彻底**：如果中途失败但目录未清理，会让用户误判运行状态。

对应策略：

- 只初始化“后续模块一定会用到”的文件，不提前铺满全部细枝末节。
- 明确 `sessions_dir` 是创建路径唯一来源。
- 统一通过 session 模块封装写入和回滚，不让 CLI 分散处理。

## 9. 与上一阶段文档的关系

- `docs/session/prd.md` 是模块总纲，定义“session 负责会话载体、状态文件与 checkpoint 体系”的大边界。
- 本文是该总纲的首个版本化落地稿，把“模块级抽象职责”细化为“`run` 成功后到底要落哪些目录、哪些文件、哪些初始化状态”。
- 相比当前仓库里“仅创建 `metadata.md`”的历史实现，`session 001` 的关键变化是：把 session 从“临时证明 run 走通了”升级成“后续所有协作模块都能依赖的标准工作目录基线”。
