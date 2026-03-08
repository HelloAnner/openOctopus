# Recovery 模块 001 阶段 E2E 设计

## 1. 目标

`recovery 001` 的 E2E 目标是证明：**一个已经创建好的 session，在主进程退出、状态文件缺失、或 session 需要重新进入推进循环时，系统可以通过正式 `recover` 命令做校验、修复与续跑。**

E2E 只验证黑盒行为：CLI 输出、session 文件结果、终态是否正确，不直接调用 `internal/recovery` 私有函数。

---

## 2. 本阶段要覆盖的能力

首版 E2E 聚焦三类恢复链路：

1. **续跑恢复**：`run` 创建 session 并完成首轮 dispatch 后，`recover` 能继续推进到 `COMPLETED`。
2. **文件修复恢复**：人为删除 `session.state.md` 后，`recover` 能重建当前态并完成续跑。
3. **严格失败保护**：如果 `events.md` 被篡改破坏哈希链，`recover` 必须直接失败。

补充一个边界场景：

4. **等待人工保护**：若 session 当前为 `WAITING_HUMAN`，`recover` 只能报告需要人工，不得替代 `resume`。

---

## 3. 明确不测的内容

`001` E2E 暂不覆盖：

- 跨主机 session 恢复
- Docker / daemon 模式恢复
- 复杂断电后多角色并发写冲突回放
- `FAILED` 终态自动翻回可执行态
- `tmux` / Web 页面上的恢复入口

---

## 4. 黑盒原则

所有断言只依赖以下可观察对象：

1. `openoctopus run` / `openoctopus recover` / `openoctopus status` 的 stdout / stderr / exit code
2. session 目录下的 Markdown / YAML 文件
3. deterministic executor 产出的角色 turn / conclusion / artifact 文件

不读取内部 Go 结构体，不直接调用 `internal/recovery` 私有接口。

---

## 5. 测试夹具规划

建议新增：

```text
e2e/recovery/
├── fixtures/
│   ├── valid-dispatched-session/
│   │   └── octopus.yaml
│   ├── valid-missing-session-state/
│   │   └── octopus.yaml
│   ├── valid-waiting-human/
│   │   └── octopus.yaml
│   └── valid-broken-events/
│       └── octopus.yaml
└── test_recovery_flow.py
```

所有 fixture 都保持单 stage、单 role 为主，避免把失败原因混到多阶段联动上。

---

## 6. 用例矩阵

| 用例 ID | 场景 | 输入 | 预期 |
| --- | --- | --- | --- |
| REC-E2E-001 | 已分发 session 续跑恢复 | `valid-dispatched-session` | `recover` 后 session 进入 `COMPLETED` |
| REC-E2E-002 | 缺失 `session.state.md` 修复恢复 | `valid-missing-session-state` | `recover` 会重建 `session.state.md`，并最终完成 |
| REC-E2E-003 | `events.md` 哈希链损坏 | `valid-broken-events` | `recover` 返回失败，且不继续推进 |
| REC-E2E-004 | `WAITING_HUMAN` 保护 | `valid-waiting-human` | `recover` 成功返回但 `continued=false`，提示走 `resume` |

---

## 7. 关键执行步骤

### 7.1 REC-E2E-001：已分发 session 续跑恢复

步骤：

1. 使用 `run --config ...` 创建 session，并设置 `OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP=1`。
2. 断言 `planner/master_schedule.md` 中已有 `status: DISPATCHED`。
3. 执行 `recover --session <session_dir>`，并让 deterministic executor 返回 `SUCCESS`。
4. 断言：
   - `session.state.md` 最终为 `COMPLETED`
   - `roles/{role_id}/turns/0001-output.md` 存在
   - `audit/replay.md` 已写入本次恢复记录
   - `state/checkpoints/` 中存在 `recover-start` 或阶段边界快照

### 7.2 REC-E2E-002：缺失 `session.state.md` 修复恢复

步骤：

1. 按 REC-E2E-001 的方式先创建已分发 session。
2. 手工删除 `session.state.md`。
3. 执行 `recover --session <session_dir>`。
4. 断言：
   - `session.state.md` 被重新创建
   - 重新创建后的文件包含 `status:`、`current_stage_id:`、`updated_at:`
   - session 最终可推进到 `COMPLETED`

### 7.3 REC-E2E-003：`events.md` 哈希链损坏

步骤：

1. 创建已分发 session。
2. 手工篡改 `bus/events.md` 某条事件的 `event_hash` 或摘要内容。
3. 执行 `recover --session <session_dir> --format json`。
4. 断言：
   - 命令非 0 退出
   - stderr JSON 中 `ok=false`
   - 错误码稳定可识别
   - 不生成新的 turn 输出，不推进 workflow

### 7.4 REC-E2E-004：`WAITING_HUMAN` 保护

步骤：

1. 运行一个 deterministic `BLOCKED` 的 session，使其进入 `WAITING_HUMAN`。
2. 执行 `recover --session <session_dir> --format json`。
3. 断言：
   - 命令返回 0
   - `continued=false`
   - `recovered_status=WAITING_HUMAN`
   - `session.state.md` 仍保持 `WAITING_HUMAN`
   - 不新增角色 turn

---

## 8. 断言重点

`recovery 001` 的黑盒断言重点如下：

1. `recover` 是否是正式 CLI 命令。
2. `recover` 是否写了 `audit/replay.md`。
3. `recover` 是否真正使用现有 session 继续推进，而不是偷偷重建一个新 session。
4. `recover` 是否会修复最小必要文件，而不是留下半恢复状态。
5. `recover` 在 `WAITING_HUMAN` 或 `events.md` 损坏时是否保持保守行为。

---

## 9. 分阶段执行策略

### 9.1 第一批：续跑恢复主链路

优先建立：

- `REC-E2E-001` 已分发 session 续跑恢复
- `REC-E2E-002` 缺失 `session.state.md` 修复恢复

这两条链路先证明 recovery 不是空壳，而是真能“接着干活”。

### 9.2 第二批：严格失败与边界保护

随后补上：

- `REC-E2E-003` 事件哈希链损坏时直接失败
- `REC-E2E-004` `WAITING_HUMAN` 保护

这两条链路保证 recovery 首版是保守的，不会把 session 越修越乱。

---

## 10. 通过标准

以下条件同时满足，才视为 `recovery 001` 的 E2E 达标：

1. 已分发 session 可以通过 `recover` 继续执行并进入正确终态。
2. `session.state.md` 缺失时，系统能自动修复并继续推进。
3. `events.md` 被篡改时，系统会严格失败而不是继续运行。
4. `WAITING_HUMAN` 不会被 recovery 自动越权恢复。
5. 所有断言都只依赖 CLI 与文件系统结果。

