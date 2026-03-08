# Artifact 模块 001 PRD

## 1. 目标

`artifact 001` 的目标，不是让 OpenOctopus 立刻拥有一个通用对象存储系统，而是先把**会话内产物的版本化落盘、稳定引用、输入回流和审计链路**做稳。

首版必须解决四个现实问题：

1. 角色已经能产出 `conclusion.md.output_refs`，但系统还没有把这些 ref 提升为可复用的正式 artifact。
2. 后续 stage 虽然在 YAML 中声明了 `input.ref`，但 orchestrator 还没有把“逻辑 artifact 名称”解析成“本次 session 内可直接读取的真实文件”。
3. session 目录虽然已经预留了 `artifacts/index.md` 与 `audit/lineage.md`，但仍是 placeholder，尚未形成真实协议。
4. 真实 Codex CLI 已经是产品主路径之一，artifact 首版需要至少支撑一个“真实 Codex 生成产物 → 注册版本 → 下游继续消费”的最小自动化闭环。

因此，`artifact 001` 的核心是：**把角色输出从“回执字符串”升级为“有版本、有索引、有血缘、可再次输入的正式产物”**。

## 2. 首版边界

### 2.1 本阶段必须覆盖

1. bootstrap `artifacts/index.md` 与 `audit/lineage.md`，替换 session placeholder。
2. 支持把 stage 成功后的输出注册为正式 artifact version。
3. 支持按 artifact 名称解析“最新版本”的 `content_ref` / `manifest_ref`，并注入到下游 `context.md`。
4. 支持文本 / Markdown 文件 artifact 的版本快照与 diff 摘要。
5. 支持目录 artifact 的快照与 manifest 落盘。
6. 支持把发布动作记录到 `audit/lineage.md` 与 bus 事件里。
7. 支持真实 Codex 在 session 内写一个简单 Markdown 产物，并被 artifact 模块正式接管。

### 2.2 本阶段明确不做

1. 不做跨 session 的全局 artifact 仓库去重。
2. 不做二进制大对象管理。
3. 不做复杂权限判定引擎；只复用 `config 001` 已经完成的静态约束。
4. 不做基于表达式的 artifact 条件路由。
5. 不做 Web UI 或下载接口。

首版坚持第一性原理：**先把 session 内版本化协议做实，再考虑跨 session 共享与复杂分发。**

## 3. 关键用户故事

### 3.1 作为 orchestrator

- 我希望当某个 stage 成功时，能够把该 stage 声明的 `output.artifact` 自动注册成正式版本，而不是继续只读 `conclusion.md`。
- 我希望在分发下一个 stage 前，能把 `input.ref` 解析成稳定的 `content_ref` / `manifest_ref`，并写入 `context.md`。

### 3.2 作为 role-runtime / executor

- 我希望输出时有一个明确、稳定、低心智负担的建议落盘位置，这样真实 Codex 不需要猜文件应该写到哪里。
- 我希望即便模型漏写 `output_refs`，系统也能先尝试使用约定的 staging ref 或 turn 输出做兜底，而不是直接丢失产物链路。

### 3.3 作为 recovery / audit

- 我希望知道某个 artifact version 是由哪个 stage、哪个 role、哪次 task、哪份 conclusion 与哪份 turn 输出产生的。
- 我希望能从 `artifacts/index.md` 快速知道某个 artifact 当前最新版本在哪里，而不是扫全 session。

## 4. 核心方案

### 4.1 目录结构

`artifact 001` 在 session 内使用如下结构：

```text
artifacts/
├── index.md
├── _staging/
│   └── {stage_id}/
│       └── {artifact_name}.md
└── {artifact_name}/
    └── 0001/
        ├── manifest.md
        ├── diff.md
        └── content.md
```

补充说明：

1. `_staging/` 是角色可写的“原始输出暂存区”，供 Codex / deterministic 先写原始结果。
2. `{artifact_name}/{version}/` 才是 artifact 模块正式管理的版本目录。
3. 如果 source ref 指向目录，则版本目录中使用 `content/` 保存完整快照，而不是 `content.md`。

### 4.2 bootstrap 协议

`session 001` 已创建 `artifacts/index.md` 与 `audit/lineage.md` placeholder；`artifact 001` 负责把它们升级成正式协议文件。

bootstrap 后要求：

#### A. `artifacts/index.md`

```markdown
# Artifact Index

- artifact_count: 0
- version_count: 0
- updated_at: 2026-03-08T10:00:00Z
```

#### B. `audit/lineage.md`

```markdown
# Artifact Lineage

- record_count: 0
- updated_at: 2026-03-08T10:00:00Z
```

幂等要求：

1. 如果文件已经是合法协议，则再次 bootstrap 不应清空历史记录。
2. 如果仍是 placeholder，则必须升级为正式协议内容。

### 4.3 输出契约与 staging 约定

artifact 首版不直接生成内容，但要给角色一个**明确的默认写入建议**。

对于 stage 中每一个 `output` 且 `type=artifact` 的条目，orchestrator 在 `context.md` 中写入：

- `artifact_name`
- `suggested_ref`
- `publish_rule`

默认 `suggested_ref` 规则：

```text
artifacts/_staging/{stage_id}/{artifact_name}.md
```

使用约束：

1. 如果角色按建议路径写出文件，则即使 `output_refs` 留空，artifact 仍应优先尝试接管该 staging 文件。
2. 如果角色需要输出目录、代码树或非 `.md` 文件，可以返回自定义 `output_refs`，artifact 模块按真实路径接管。
3. 如果既没有 `output_refs`，建议 staging ref 也不存在，则最后才回退到 `outbox.turn_output_ref`，确保首版兼容已有 deterministic / turn 输出路径。

最终兜底优先级如下：

```text
conclusion.output_refs
  ↓
context suggested_ref
  ↓
outbox.turn_output_ref
```

### 4.4 发布时机

正式发布动作不放在 role-runtime，而放在 orchestrator 消费 `SUCCESS` 结论时执行。

原因：

1. orchestrator 拥有 stage 配置，知道“当前 task 对应哪些逻辑 artifact name”。
2. orchestrator 已持有 bus lease，方便把 artifact 发布与 stage 完成事件串成同一条审计链。
3. role-runtime 只负责真实执行与回执，不负责跨 stage 的正式存储索引。

发布顺序：

1. 读取 stage 的 artifact 输出声明。
2. 解析 `conclusion.md.output_refs`。
3. 结合默认 `suggested_ref` 与 `outbox.turn_output_ref` 生成最终 source refs。
4. 对每个 artifact 递增版本号。
5. 快照 source 内容到正式版本目录。
6. 写 `manifest.md` 与 `diff.md`。
7. 追加更新 `artifacts/index.md`。
8. 追加更新 `audit/lineage.md`。
9. 追加 bus 事件 `ARTIFACT_PUBLISHED`。

只要任一步失败，本轮 Tick 直接失败，不做静默跳过。

### 4.5 `artifacts/index.md` 协议

`index.md` 是首版唯一的 artifact registry，按 block 记录每一个 version。

示例：

```markdown
# Artifact Index

- artifact_count: 1
- version_count: 2
- updated_at: 2026-03-08T10:05:00Z

## artifact: solution_doc@0001
- artifact_name: solution_doc
- version: 1
- stage_id: design_solution
- role_id: agent_a
- task_id: task-design_solution-01
- source_ref: artifacts/_staging/design_solution/solution_doc.md
- content_ref: artifacts/solution_doc/0001/content.md
- manifest_ref: artifacts/solution_doc/0001/manifest.md
- diff_ref: artifacts/solution_doc/0001/diff.md
- content_hash: sha256:ab12...
- created_at: 2026-03-08T10:02:00Z

## artifact: solution_doc@0002
- artifact_name: solution_doc
- version: 2
...
```

约束如下：

1. 同名 artifact 按 version 严格递增，从 `0001` 开始。
2. 最新版本的判定规则固定为“同名下最大 version”。
3. `content_ref` / `manifest_ref` / `diff_ref` 必须是 session 相对路径。
4. `content_hash` 必须带算法前缀，例如 `sha256:`。

### 4.6 `manifest.md` 协议

每个 artifact version 都必须有一份 `manifest.md`，至少包含：

- `artifact_name`
- `version`
- `source_ref`
- `source_kind: file | directory`
- `stage_id`
- `role_id`
- `task_id`
- `conclusion_ref`
- `turn_output_ref`
- `content_ref`
- `content_hash`
- `previous_version`
- `created_at`

如果 source 是文件，还应补充：

- `file_ext`
- `size_bytes`
- `line_count`

如果 source 是目录，还应补充：

- `file_count`

### 4.7 `diff.md` 协议

首版的 diff 不追求复杂到可以直接 code review，而是先保证“版本差异有证据、有摘要”。

要求如下：

1. 第一个版本必须写明 `initial version`。
2. 文件 artifact 至少写出：`previous_hash`、`current_hash`、`changed`、`previous_content_ref`、`current_content_ref`。
3. 目录 artifact 至少写出：`previous_file_count`、`current_file_count`、`added_or_removed` 摘要。
4. 首版允许 diff 只做摘要，不强求完整 unified diff。

### 4.8 输入解析协议

当某个 stage 的 `input` 中存在 `type=artifact` 且 `ref=artifact_name` 时，orchestrator 在分发前必须解析该 artifact 的最新版本，并把以下信息写入 `context.md`：

- `ref`
- `resolved_version`
- `content_ref`
- `manifest_ref`

如果找不到对应 artifact：

1. 不允许继续分发；
2. 直接返回稳定错误，避免让下游角色拿着悬空输入继续执行。

### 4.9 真实 Codex 兼容策略

artifact 首版要求至少支持一个真实 Codex 闭环。为了降低首版复杂度，约束如下：

1. `codex` 执行工作目录固定为当前 session 根目录。
2. role prompt 必须明确要求读取 `context.md` / `inbox.md`，并在结束时输出标准 `role_result` block。
3. 如果 `context.md` 中声明了 `output_artifacts.suggested_ref`，Codex 必须优先把产物写到该位置。
4. artifact 模块不依赖 Codex 的自然语言正文，只依赖文件副作用与 `role_result` 结构块。

这样可以保证：即便模型措辞波动，artifact 仍能通过文件路径与 version 协议稳定工作。

## 5. 模块边界

| 模块 | 负责内容 | 不负责内容 |
| --- | --- | --- |
| `session` | 提供 `artifacts/` 与 `audit/` 骨架 | 版本索引与输入解析 |
| `role-runtime` | 执行任务、写 turn / conclusion / outbox | 正式发布 artifact version |
| `orchestrator` | 成功后触发发布、分发前解析输入 | 内容生成本身 |
| `event-bus` | 记录 `ARTIFACT_PUBLISHED` 事件 | artifact 内容快照 |
| `artifact` | bootstrap、版本化、索引、diff 摘要、lineage、输入解析 | 判定 stage 成功失败 |

## 6. 必交付物

1. `docs/artifact/001/prd.md`
2. `docs/artifact/001/e2e.md`
3. `docs/artifact/001/plans/`
4. `internal/artifact` 首版实现
5. `orchestrator` 对 artifact 输入解析与成功发布的接入
6. `role-runtime` 对建议输出 ref / 真实 Codex 执行的兼容补齐
7. `e2e/artifact/` 黑盒用例与真实 Codex 简单闭环

## 7. 验收标准

以下条件同时满足，才视为 `artifact 001` 达标：

1. `run` 在带 artifact 输出的合法配置下，能把 stage 成功结果注册到 `artifacts/index.md`。
2. 同名 artifact 重复发布时，version 会从 `0001`、`0002` 按顺序递增。
3. `manifest.md`、`diff.md`、`audit/lineage.md` 与 bus 事件都能留下稳定证据。
4. 下游 stage 的 `context.md` 可以拿到上游 artifact 的稳定 `content_ref` / `manifest_ref`。
5. deterministic 场景与真实 Codex 场景都能跑通最小闭环。
6. `make check` 全量通过。

## 8. 依赖与风险

### 8.1 上游依赖

- `config 001`：提供 artifact 声明、输入引用和基础策略配置
- `session 001`：提供 `artifacts/` / `audit/` 根骨架
- `event-bus 001`：提供追加事件与 lease 能力
- `orchestrator 001`：提供 stage dispatch 与 conclusion 消费主循环
- `role-runtime 001`：提供 `conclusion.md` / `outbox.md` / turn 协议

### 8.2 主要风险

1. **Codex 未按约定写文件**：必须通过 suggested ref + turn output 双兜底降低失败率。
2. **现有无 artifact 感知的 deterministic 测试被打破**：必须保留 turn output fallback，保证前置模块回归稳定。
3. **artifact 输入解析与 stage 分发耦合过深**：要把版本化逻辑收敛在 `internal/artifact`，orchestrator 只调用服务，不手写目录协议。
4. **目录快照实现过重**：首版只做快照与摘要，不做复杂逐文件 diff。

## 9. 与模块总纲的关系

`docs/artifact/prd.md` 定义的是“artifact 模块为什么存在”；`artifact 001` 定义的是“session 内版本化 artifact store 如何真正工作”。

相对模块总纲，本版新增了以下首版基线：

1. session 内 `_staging -> versioned store` 的最小发布链路；
2. `index.md / manifest.md / diff.md / lineage.md` 四类协议文件；
3. orchestrator 成功发布与下游输入解析的明确接入点；
4. 真实 Codex 简单任务闭环的可验证路径。
