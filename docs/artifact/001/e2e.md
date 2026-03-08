# Artifact 模块 001 阶段 E2E 方案

## 1. 目标

`artifact 001` 的 E2E 只验证一个黑盒事实：**角色成功后，OpenOctopus 是否能把原始输出接管成正式 artifact version，并让后续 stage 继续消费。**

本阶段必须同时覆盖两类链路：

1. **稳定回归链路**：deterministic executor，保证版本递增、索引、输入解析和差异摘要可重复验证。
2. **真实执行链路**：宿主机真实 `codex` CLI，完成一个简单的 Markdown 产物自动化流程。

## 2. 环境原则

1. 默认使用宿主机真实 `~/.codex/` 与真实 `codex` CLI。
2. 默认在仓库根目录 `e2e-test/` 下创建测试工作目录。
3. 每次执行前清理旧的 `e2e-test/`。
4. 不 mock artifact 模块，不直接从 Python 调 Go 私有函数。
5. 黑盒断言只看 CLI 退出码、session 文件、副产物文件和正式 artifact 协议文件。

## 3. 验证范围

### 3.1 本阶段必须覆盖

1. `artifacts/index.md` 与 `audit/lineage.md` bootstrap。
2. stage 成功后 artifact version 正式发布。
3. 同名 artifact 的 version 递增。
4. `manifest.md` / `diff.md` / `content_ref` 落盘正确。
5. 下游 stage `context.md` 能拿到上游 artifact 的解析结果。
6. 真实 Codex 能在 session 内生成简单 Markdown 产物并被 artifact 模块接管。

### 3.2 本阶段暂不覆盖

1. 二进制大文件 artifact。
2. UI 下载与浏览。
3. 跨 session artifact 共享。

## 4. 测试夹具设计

### 4.1 `valid-deterministic-publish`

目标：证明最小成功 stage 可以把 turn 输出或显式 source ref 发布为正式 artifact。

特点：

- 单 stage、单 role
- deterministic executor 返回 `SUCCESS`
- 断言 `artifacts/index.md`、`artifacts/artifact_a/0001/manifest.md`、`diff.md`、`content.*` 存在

### 4.2 `valid-two-stage-resolution`

目标：证明 stage_a 发布的 artifact，可以在 stage_b 分发时被解析到 `context.md`。

特点：

- 两个 stage 线性流转
- stage_b 的 `input.ref = artifact_a`
- 断言 stage_b 的 `context.md` 含 `content_ref` / `manifest_ref`

### 4.3 `valid-version-bump`

目标：证明同名 artifact 在两次成功发布后会递增成 `0001`、`0002`，并生成第二版 `diff.md`。

特点：

- 两个成功 stage 都输出同名 `artifact_a`
- 断言 `artifacts/artifact_a/0001/` 与 `0002/` 同时存在
- `index.md` 的 `version_count` 为 `2`

### 4.4 `valid-codex-simple-flow`

目标：证明真实宿主机 `codex` CLI 可以在 session 内完成一个简单 artifact 自动化流程。

建议流程：

1. stage_a 用真实 Codex 读取 `context.md`，在 `suggested_ref` 写一个 3~5 行 Markdown 说明，并输出 `SUCCESS`。
2. stage_b 再读取 stage_a 已发布 artifact，在新的 `suggested_ref` 写一份极简 review / summary Markdown，并输出 `SUCCESS`。

黑盒重点只验证文件链路，不依赖具体自然语言内容。

## 5. 核心用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| ART-E2E-001 | bootstrap artifact 协议 | `valid-deterministic-publish/octopus.yaml` | `artifacts/index.md` 与 `audit/lineage.md` 不再是 placeholder |
| ART-E2E-002 | 首个 artifact 发布 | `valid-deterministic-publish` | `artifact_a/0001/manifest.md`、`diff.md`、`content.*` 存在 |
| ART-E2E-003 | 下游输入解析 | `valid-two-stage-resolution` | stage_b `context.md` 含 `ref/content_ref/manifest_ref` |
| ART-E2E-004 | 同名 artifact 版本递增 | `valid-version-bump` | 同时存在 `0001` 与 `0002`，且 `index.md` 版本计数正确 |
| ART-E2E-005 | lineage 记录 | 任一成功用例 | `audit/lineage.md` 含 stage / role / task / manifest / content 信息 |
| ART-E2E-006 | 真实 Codex 简单闭环 | `valid-codex-simple-flow` | workflow `COMPLETED`，两个 artifact 都完成正式发布 |

## 6. 关键断言方式

### 6.1 bootstrap 断言

- `artifacts/index.md` 不包含 `Initialized by session 001.`
- `audit/lineage.md` 不包含 `Initialized by session 001.`
- 两个文件都有 `updated_at`

### 6.2 发布断言

至少验证：

- `artifacts/index.md` 出现 `artifact_name: artifact_a`
- `artifacts/artifact_a/0001/manifest.md` 存在
- `artifacts/artifact_a/0001/diff.md` 存在
- `content_ref` 指向的文件或目录真实存在

### 6.3 版本递增断言

- `0001/` 与 `0002/` 同时存在
- `index.md` 中同名 artifact 的 version 记录连续
- 第二版 `diff.md` 含 `previous_hash` 与 `current_hash`

### 6.4 输入解析断言

- `roles/{role_id}/context.md` 中能看到 `input_artifacts`
- 至少一条输入包含 `ref`、`content_ref`、`manifest_ref`
- `content_ref` 对应路径真实存在

### 6.5 真实 Codex 断言

- 使用真实 `CODEX_HOME`
- `turns/0001-output.md` 中 `executor_provider = codex`
- `artifacts/<name>/0001/content.md` 真实存在
- `session.state.md.status = COMPLETED`

## 7. 推荐测试实现

### 7.1 继续复用 `pytest` 基础设施

建议继续复用：

- `project_root`
- `binary_path`
- `workspace_dir`
- `run_cli(...)`
- `prepare_module_case(...)`

artifact 模块只需新增：

- `e2e/artifact/fixtures/...`
- `e2e/artifact/test_artifact_pipeline.py`

### 7.2 环境变量策略

为了避免影响其他模块 E2E：

1. `run_cli` 默认关闭 role-runtime loop。
2. 只有 `role-runtime` 与 `artifact` 模块用例显式把 `OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP=0` 打开。

这样可以保证新增真实 Codex 执行链路后，不会把原本只验证 session / bus / orchestrator bootstrap 的测试误拖入真实执行。

## 8. 通过标准

以下条件同时满足，artifact 001 E2E 才算达标：

1. deterministic 用例能稳定验证 artifact 发布、输入解析、version bump 与 lineage。
2. 真实 Codex 用例能在宿主机直接跑通一个简单双阶段 artifact 闭环。
3. `make check` 全量通过。
