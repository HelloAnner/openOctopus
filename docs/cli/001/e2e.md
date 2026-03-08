# CLI 模块 001 阶段 E2E 方案

## 1. 目标

`cli 001` 的 E2E 不验证内部实现细节，而是验证一个黑盒事实：**OpenOctopus 首版正式 CLI 是否已经具备稳定的脚本输出协议、session 状态观测能力和基础退出码约定。**

本阶段的关注点不是“业务模块是否存在”，因为 `config`、`session`、`event-bus`、`orchestrator`、`role-runtime`、`human-gate` 已经分别有自己的模块 E2E；`cli 001` 只验证这些模块通过正式 CLI 暴露出来以后，是否已经变成“适合用户和脚本消费”的入口层。

---

## 2. 测试原则

- 默认直接在宿主机执行，复用真实 `~/.codex/` 环境。
- 不 mock CLI 进程，统一通过 `pytest + subprocess` 黑盒调用正式二进制。
- 优先验证 stdout / stderr / 退出码 / 文件系统副作用，不读取 Go 私有结构。
- `cli 001` 主链路尽量走正式命令：`validate`、`run`、`status`、`interrupt-all`。
- 对状态推进的确定性要求高的场景，优先使用 deterministic executor。

---

## 3. 验证范围

### 3.1 本阶段必须覆盖

1. `validate --format json` 成功时输出稳定 JSON。
2. `validate --format json` 配置错误时输出稳定 JSON 错误，并返回配置阻断退出码。
3. `run --format json` 成功时输出 `session_id` / `session_dir`。
4. `status --format json` 能读取 session 当前态与调度摘要。
5. `interrupt-all --format json` 后，`status` 能读到 `WAITING_HUMAN`。
6. session 不存在时，`status` 返回稳定退出码与可读错误。

### 3.2 本阶段暂不覆盖

- `status --watch`
- Bubble Tea 实时界面
- `report` / `debug` / `stop` / `init`
- Web UI / HTTP API

---

## 4. 夹具设计

建议目录：

```text
e2e/
├── cli/
│   ├── fixtures/
│   │   ├── valid-minimal/
│   │   │   └── octopus.yaml
│   │   ├── valid-deterministic-success/
│   │   │   └── octopus.yaml
│   │   └── valid-interrupt-all/
│   │       └── octopus.yaml
│   ├── test_validate_output.py
│   └── test_run_status.py
```

### 4.1 `valid-minimal`

目标：验证 `validate --format json` 与 `run --format json` 的最小成功路径。

要求：

- 配置可直接通过校验。
- 使用真实 `codex` profile，保证 CLI 对宿主机 `~/.codex/` 依赖链路仍然可用。

### 4.2 `valid-deterministic-success`

目标：验证 `status --format json` 在完成态下可稳定返回核心字段。

要求：

- 单角色、单阶段。
- 使用 deterministic executor，避免外部不确定性。
- 通过环境变量让执行结果稳定为 `SUCCESS`。

### 4.3 `valid-interrupt-all`

目标：验证 `interrupt-all` 与 `status` 的联动。

要求：

- 至少两个未完成角色，便于验证批量中断。
- `run` 时禁用自动 runtime loop，保留待执行角色。
- `interrupt-all` 后 session 应立即进入 `WAITING_HUMAN`。

---

## 5. 用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| CLI-E2E-001 | `validate` JSON 成功 | `valid-minimal` | 退出码 `0`，stdout 为合法 JSON，`ok=true` |
| CLI-E2E-002 | `validate` JSON 失败 | 非法 YAML | 退出码 `2`，stderr 为合法 JSON，`code=config_validation_failed` |
| CLI-E2E-003 | `run` JSON 成功 | `valid-minimal` | 退出码 `0`，stdout 包含 `session_id` 和 `session_dir` |
| CLI-E2E-004 | `status` 读取完成态 | `valid-deterministic-success` | 退出码 `0`，stdout JSON 中 `workflow_status=COMPLETED` |
| CLI-E2E-005 | `interrupt-all` 后 `status` 读取等待人工 | `valid-interrupt-all` | `workflow_status=WAITING_HUMAN`，`blocker_summary` 包含原因 |
| CLI-E2E-006 | session 不存在 | `status --session missing` | 退出码 `3`，stderr 含稳定错误 |

---

## 6. 关键断言方式

### 6.1 JSON 输出断言

对 stdout / stderr：

- 先 `json.loads(...)`
- 再断言顶层字段：`ok`、`command`
- 再断言 `data` 或 `error`

不允许只用字符串包含关系判断 JSON 主体是否正确。

### 6.2 状态读取断言

`status` 至少要断言：

- `session_id`
- `workflow_status`
- `current_stage_id`
- `current_role_id`
- `schedule_version`
- `active_dispatch_count`
- `blocker_summary`

### 6.3 退出码断言

本阶段重点断言两类退出码：

- `2`：配置校验失败
- `3`：session 不存在

不要求在 `001` 覆盖所有业务错误码。

---

## 7. 测试文件规划

建议新增：

- `e2e/cli/fixtures/valid-minimal/octopus.yaml`
- `e2e/cli/fixtures/valid-deterministic-success/octopus.yaml`
- `e2e/cli/fixtures/valid-interrupt-all/octopus.yaml`
- `e2e/cli/test_validate_output.py`
- `e2e/cli/test_run_status.py`

并同步更新：

- `e2e/README.md`
- `Makefile`

---

## 8. 通过标准

以下命令通过，视为 `cli 001` E2E 合格：

```bash
python3 -m pytest e2e/cli -v
make check
```

