# Config 模块 001 阶段 PRD

## 1. 阶段定位

`config` 是 OpenOctopus 首版的第一落地模块，负责把平台级 `docs/prd/prd-001.md` 中定义的 YAML DSL、环境变量覆盖、命令行参数覆盖、默认值注入和静态校验真正收敛成一套可执行协议。

`001` 阶段不追求“高级配置中心”，只解决首版自动化主链路最关键的问题：**给 `run` / `validate` / 后续 orchestrator、session、role-runtime 一个稳定、可追溯、可阻断的统一输入对象**。

本阶段完成后，配置不再只是文档约定，而是具备以下能力：

- 用户可以用一份 `octopus.yaml` 描述主流程、角色、工具和安全边界。
- 系统可以将 YAML、环境变量和 flags 合并为强类型 `RuntimeConfig`。
- 系统可以在启动前阻断非法配置，避免带病运行。
- 默认值来源明确、可记录、可对外解释。
- 后续 AI 生成配置时，可以依赖 `docs/config/001/yaml-rules.md` 这份规则协议文档进行约束生成。

## 2. 本阶段要解决的问题

### 2.1 当前问题

当前仓库只有平台总 PRD 和模块总纲，尚未把 `config` 收敛为可执行阶段设计，存在以下空白：

1. **输入协议不稳定**：`octopus.yaml` 的顶层结构、引用关系、默认值边界还停留在平台概念层。
2. **优先级不明确**：YAML、环境变量、flags 谁覆盖谁、哪些字段允许被覆盖，还没有阶段化约束。
3. **错误模型不统一**：配置错误只知道“失败”，但不知道失败属于语法、结构、引用还是安全约束。
4. **AI 生成链路缺口**：平台已经要求提供 `yaml-rules.md`，但没有定义它与运行时校验之间如何同步。
5. **首版阻断边界未落地**：`validate` 和 `run` 在遇到非法配置时应该如何停止、输出什么，还未形成闭环。

### 2.2 001 阶段目标

`001` 阶段的核心目标只有一条：**把配置从“说明文档”升级成“运行时协议”**。

具体体现为：

- 明确 `RuntimeConfig` 的顶层结构与消费边界。
- 明确配置来源优先级与允许覆盖的字段范围。
- 明确默认值注入策略、记录方式和输出约定。
- 明确静态校验分类、错误码、阻断规则和面向 AI 的规则映射。
- 明确 `validate` 与 `run` 对 `config` 模块的依赖方式。

## 3. 范围定义

### 3.1 001 范围内

本阶段包含以下能力：

1. **本地配置文件加载**：读取单个 `octopus.yaml`。
2. **环境变量覆盖**：支持固定前缀环境变量覆盖部分配置。
3. **命令行参数覆盖**：支持有限的显式 flags 覆盖关键运行参数。
4. **默认值注入**：对未填写但首版必须存在的保守参数补默认值。
5. **强类型配置对象**：输出统一 `RuntimeConfig`，供 CLI 与后续模块消费。
6. **静态校验**：覆盖结构、引用、流转、环路、安全、监控阈值、只读产物等校验。
7. **错误输出协议**：输出稳定的错误类别、错误码、定位路径和修复建议。
8. **规则文档挂钩**：定义 `yaml-rules.md` 与运行时规则同步的责任边界。

### 3.2 001 范围外

本阶段明确不做：

- 远程配置中心。
- 配置热更新与运行中动态重载。
- 多文件 include / import / inheritance。
- Web 可视化配置编辑器。
- 任意复杂对象的环境变量覆盖能力。
- 插件扩展配置的动态 Schema 装载。

## 4. 核心用户故事

### 4.1 命令行用户

- 作为用户，我希望执行 `openoctopus validate --config ./octopus.yaml` 时，能在启动前看到清晰的配置错误，而不是运行到一半才发现配置有问题。
- 作为用户，我希望通过少量环境变量或 flags 覆盖环境相关参数，而不需要为每个环境复制一份 YAML。
- 作为用户，我希望最小可运行 YAML 不需要填写所有保守参数，系统可以安全补默认值。

### 4.2 主控与运行时模块

- 作为 CLI，我希望调用一次 `config` 模块，就能拿到已经合并、默认值补齐、校验完成的 `RuntimeConfig`。
- 作为 orchestrator / session / role-runtime，我只消费强类型结果，不再自己读取原始 YAML。
- 作为 recovery，我希望错误信息和有效配置快照具备可追溯性，便于恢复或复盘。

### 4.3 AI 配置生成链路

- 作为后续 AI 生成器，我希望有一份与真实校验逻辑同步的 `yaml-rules.md`，避免“文档允许、运行时报错”的分叉。
- 作为 AI 修复器，我希望错误输出能定位到规则条目，便于自动修正 YAML。

## 5. 核心方案

### 5.1 配置来源与优先级

`001` 阶段采用固定优先级，自低到高为：

1. **系统默认值**
2. **`octopus.yaml` 文件**
3. **环境变量**
4. **命令行 flags**

约束如下：

- YAML 是主配置源，负责描述完整结构。
- 环境变量只覆盖**运行环境相关、敏感信息相关、标量型字段**，不承担主结构定义责任。
- flags 只覆盖**启动期常用控制项**，避免首版做成“任何字段都可 flag 化”的复杂系统。
- 一旦高优先级来源显式给值，低优先级来源必须完全让位，不做隐式合并猜测。

`001` 阶段推荐的环境变量规范：

- 统一前缀：`OPENOCTOPUS_`
- 路径映射：使用双下划线 `__` 表示层级，例如 `OPENOCTOPUS_RUNTIME__TMUX__ENABLED=true`
- 标量转换：由 `config` 模块统一负责布尔、整数、浮点、字符串列表的解析

`001` 阶段推荐支持的 flags 覆盖范围：

- `--workspace-root`
- `--sessions-dir`
- `--artifacts-dir`
- `--logs-dir`
- `--tmux-enabled`
- `--max-parallel-roles`
- `--stage-timeout-seconds`

这样做的目的不是让 flags 替代 YAML，而是让首版在不同机器、不同 CI 环境中保留必要的动态入口。

### 5.2 强类型配置模型

`config` 模块对外只输出一个统一对象：`RuntimeConfig`。

`001` 阶段至少覆盖以下顶层域：

- `meta`
- `runtime`
- `llm_profiles`
- `tool_registry`
- `security`
- `policies`
- `roles`
- `stages`
- `transitions`

消费约束如下：

- CLI 只依赖 `RuntimeConfig`，不直接读取 YAML。
- orchestrator 只依赖 `RuntimeConfig` 中的角色、阶段、流转、策略对象。
- role-runtime 只依赖角色视角所需的 llm、tools、constraints、路径与安全边界。
- session / artifact / recovery 后续如果需要记录有效配置快照，也只记录 `RuntimeConfig` 的序列化结果。

### 5.3 默认值注入策略

默认值注入遵循“**保守、显式、可追踪**”三条原则：

1. **保守**：默认值优先保证稳定，不追求高吞吐。
2. **显式**：只有 PRD 与 `yaml-rules.md` 明确列出的字段允许补默认值。
3. **可追踪**：每个生效默认值都必须能被记录和解释。

`001` 阶段默认值分三类：

#### A. 工作目录类

- `runtime.workspace.root`: `.octopus`
- `runtime.workspace.sessions_dir`: `.octopus/sessions`
- `runtime.workspace.artifacts_dir`: `.octopus/artifacts`
- `runtime.workspace.logs_dir`: `.octopus/logs`

#### B. 运行安全类

- `runtime.checkpoint.enabled`: `true`
- `runtime.checkpoint.on_stage_boundary`: `true`
- `runtime.role_runtime.idle_poll_seconds`: `2`
- `policies.retry.max_retry_per_stage`: `2`
- `policies.retry.backoff_seconds`: `[5, 20]`
- `policies.timeout.stage_timeout_seconds`: `1800`
- `policies.timeout.role_heartbeat_timeout_seconds`: `120`
- `policies.loop_guard.max_rounds_per_task`: `6`
- `policies.loop_guard.min_quality_gain`: `0.05`

#### C. 可观测性类

- `runtime.master_watch.enabled`: `true`
- `runtime.master_watch.progress_file`: `planner/global_progress.md`
- `runtime.master_watch.blocker_file`: `planner/blockers.md`
- `runtime.master_watch.max_no_progress_rounds`: `3`

默认值记录要求：

- `config` 模块返回 `AppliedDefaults[]` 列表。
- `validate` 命令需要能展示默认值摘要。
- `run` 命令需要把默认值摘要写入启动日志或后续 session 元数据。

### 5.4 静态校验模型

`001` 阶段的校验按“从易错到高风险”分五层执行：

#### 第一层：语法与反序列化校验

- YAML 语法是否合法。
- 字段类型是否能映射到 `RuntimeConfig`。
- 未知字段是否出现。

#### 第二层：结构与必填校验

- 顶层对象是否合法。
- `roles.id`、`stages.id` 是否唯一。
- 必填字段是否缺失。
- 合法枚举值是否越界。

#### 第三层：引用与流转校验

- `stage.role` 是否引用已定义角色。
- `input.ref` 是否引用已产出的 artifact。
- `transitions.from/to/on_true/on_false` 是否引用存在节点或 `__END__`。
- `session_reset` 阶段是否声明合法的目标角色与保留策略。

#### 第四层：策略与安全校验

- 角色 `tools` 是否能在 `tool_registry` 中解析。
- 启用 `shell_exec` 的角色是否存在 `security.shell` 配置。
- `immutable_artifacts.paths` 是否与未授权写路径冲突。
- 角色读写边界、只读输入、E2E 产物不可写约束是否冲突。

#### 第五层：运行保护校验

- `loop_guard`、`deadlock_guard`、`master_watch` 阈值是否为正数。
- 超时、重试、并发度等参数是否在保守区间。
- 关键文件路径是否为空或存在明显危险配置。

### 5.5 错误输出协议

`001` 阶段所有错误必须归一到统一模型：

- `code`：错误码，例如 `CFG-SYNTAX-001`
- `category`：错误分类，例如 `syntax` / `schema` / `reference` / `security`
- `path`：字段路径，例如 `roles[1].tools[0]`
- `message`：面向用户的简洁错误说明
- `suggestion`：修复建议
- `rule_id`：可选，映射到 `yaml-rules.md` 中的规则条目

推荐错误分类：

- `syntax`：YAML 语法与反序列化失败
- `schema`：必填、类型、枚举、唯一性问题
- `reference`：角色、阶段、artifact、transition 引用错误
- `security`：工具权限、路径、只读冲突、安全策略缺失
- `policy`：超时、重试、循环保护、监控阈值非法

阻断规则：

- `validate` 遇到任意错误，返回非 0。
- `run` 在配置阶段遇到任意错误，禁止创建或推进 session。
- 同一份配置允许一次性返回多条错误，避免用户“修一条冒一条”。

### 5.6 `yaml-rules.md` 同步责任

`docs/config/001/yaml-rules.md` 是 `config 001` 的正式交付物之一，负责把运行时校验规则转成可供人工与 AI 共同消费的配置生成协议。

同步原则如下：

1. **先补规则，再开能力**：新增 YAML 字段或能力时，先更新 `yaml-rules.md`，再开放运行时支持。
2. **错误可回链**：关键校验错误需要映射到 `rule_id`。
3. **示例可复制**：规则文档中的 YAML 示例必须可通过 `validate`。
4. **边界一致**：文档允许什么、运行时就允许什么；文档禁止什么、运行时就必须阻断。

### 5.7 模块接口边界

`001` 阶段建议对外暴露两个稳定入口：

- `LoadForValidate(...)`：返回 `RuntimeConfig`、`AppliedDefaults[]`、`ValidationErrors[]`
- `LoadForRun(...)`：返回 `RuntimeConfig`、`AppliedDefaults[]`，如校验失败则直接阻断

边界要求：

- `config` 模块不负责启动 tmux、创建 session、写 bus 事件。
- `config` 模块负责把“配置是否合法”在运行前说清楚。
- 下游模块不得绕过 `config` 直接消费原始 YAML。

## 6. 分阶段交付物

### 6.1 必交付物

1. `RuntimeConfig` 首版结构定义。
2. YAML + env + flags 的加载与合并逻辑。
3. 默认值注入逻辑与默认值记录输出。
4. 静态校验器与统一错误模型。
5. `validate` 命令接入规范。
6. `run` 启动前配置阻断规范。
7. `docs/config/001/yaml-rules.md` 规则文档与 `rule_id` 编号体系。
8. 配套黑盒 E2E 方案文档：`docs/config/001/e2e.md`。
9. `docs/config/001/plans/` 拆分实现计划。

### 6.2 建议代码落位

当进入实现阶段时，建议优先按以下边界落位，避免一个超大类承载全部逻辑：

- `internal/config/model`
- `internal/config/loader`
- `internal/config/defaults`
- `internal/config/validator`
- `internal/config/errors`

原因：`config` 是首版输入协议，逻辑天然包含解析、补全、校验、报错四类职责，必须从一开始就拆开，避免后续演化成巨型 God Object。

## 7. 验收标准

满足以下条件，才视为 `config 001` 完成：

1. `openoctopus validate --config ./octopus.yaml` 可以对合法配置返回成功，对非法配置返回非 0。
2. YAML、环境变量、flags 的优先级稳定且可被黑盒验证。
3. 最小可运行 YAML 可以依赖默认值补齐，不需要手写全部保守参数。
4. 结构、引用、流转、安全、只读产物、监控阈值等核心错误都能在启动前被发现。
5. 错误输出包含字段路径、错误类别和修复建议。
6. `run` 在配置非法时必须在 session 创建前失败。
7. `yaml-rules.md` 已补齐规则编号、示例与回链约束，后续实现必须严格对齐该文档。

## 8. 依赖关系与风险

### 8.1 上游依赖

`config 001` 基本没有强上游模块依赖，它本身就是首个落地模块。

### 8.2 下游影响

以下模块直接依赖 `config 001`：

- `cli`
- `session`
- `orchestrator`
- `role-runtime`
- `artifact`
- `recovery`

### 8.3 主要风险

1. **过度开放环境变量覆盖范围**：会让首版配置行为变得不可预期。
2. **默认值过多且不可见**：会导致用户误以为 YAML 生效，实际上吃的是隐藏默认值。
3. **错误信息不可定位**：会让 AI 修复和人工排查都变慢。
4. **`yaml-rules.md` 与真实校验脱节**：会直接破坏“AI 生成 YAML -> validate -> run”的链路。

## 9. 与上一阶段文档的关系

相对于 `docs/config/prd.md` 这个模块总纲，`001` 的关键变化是：

- 从“模块职责摘要”推进到“首版配置协议设计”。
- 从“支持 Koanf + validator”推进到“明确来源优先级、错误模型、默认值策略和阻断规则”。
- 从“只有目标描述”推进到“有验收标准、有交付物、有 E2E 闭环”。

这意味着 `docs/config/prd.md` 继续作为模块总纲存在，而 `docs/config/001/prd.md` 才是当前可评审、可执行、可继续迭代的首版阶段文档。
