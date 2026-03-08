# Config 模块 001 阶段 E2E 方案

## 1. 目标

`config 001` 的 E2E 不验证内部实现细节，只验证一个黑盒事实：**给定不同来源的配置输入，`openoctopus validate` 和 `openoctopus run` 是否能在真实环境里输出正确结果，并在非法场景下稳定阻断。**

本方案遵循仓库既定 E2E 原则：

- 真实本地 Docker 环境优先。
- 每次执行前先 `docker compose down -v` 清理环境。
- 不 mock 配置校验链路。
- 通过 CLI 进程退出码、标准输出、标准错误和文件系统副作用做黑盒断言。

由于 `config 001` 还没有浏览器前端，因此本阶段 E2E 以 `pytest + subprocess + docker compose` 为主；`playwright` 保留在统一 E2E 依赖中，但本阶段不是主断言手段。

## 2. 验证范围

### 2.1 本阶段必须覆盖

1. 合法 YAML 能通过 `validate`。
2. 非法 YAML 会在 `validate` 阶段被阻断。
3. 非法配置执行 `run` 时，会在 session 创建前失败。
4. YAML / env / flags 的优先级符合 `defaults < yaml < env < flags`。
5. 默认值注入后，最小可运行配置仍可通过校验。
6. 结构、引用、安全、只读产物、策略阈值错误可以被黑盒识别。
7. 错误输出包含字段路径、错误分类和可读修复建议。

### 2.2 本阶段暂不覆盖

- orchestrator 真正调度执行。
- tmux pane 编排。
- role-runtime 执行器生命周期。
- recovery 重放。
- 浏览器 UI 操作。

这些能力会在后续模块 `001` 落地后，基于 config 的真实入口继续扩展集成 E2E。

## 3. 环境与目录约定

建议测试文件按以下结构落位：

```text
e2e/
├── requirements.txt
├── conftest.py
├── docker-compose.test.yml
├── config/
│   ├── fixtures/
│   │   ├── valid-minimal/
│   │   │   └── octopus.yaml
│   │   ├── valid-env-override/
│   │   │   └── octopus.yaml
│   │   ├── invalid-syntax/
│   │   │   └── octopus.yaml
│   │   ├── invalid-missing-role/
│   │   │   └── octopus.yaml
│   │   ├── invalid-shell-security/
│   │   │   └── octopus.yaml
│   │   ├── invalid-immutable-conflict/
│   │   │   └── octopus.yaml
│   │   └── invalid-threshold/
│   │       └── octopus.yaml
│   ├── test_validate.py
│   └── test_run_gate.py
└── README.md
```

### 3.1 Docker 约定

`docker-compose.test.yml` 至少包含一个测试容器：

- 构建当前仓库的 `openoctopus` 二进制或镜像。
- 挂载 `e2e/config/fixtures` 到容器内固定路径。
- 在容器内执行 `validate` / `run` 命令。

执行前固定清理：

```bash
docker compose -f e2e/docker-compose.test.yml down -v
```

执行命令推荐：

```bash
pytest e2e/config -v
```

## 4. 测试夹具设计

### 4.1 `valid-minimal`

目标：证明最小 YAML 可以依赖默认值通过校验。

特点：

- 只保留首版必填字段。
- 不显式写入 `retry`、`timeout`、`checkpoint` 等保守参数。
- 使用最小角色、最小阶段、最小流转。

### 4.2 `valid-env-override`

目标：证明环境变量可以覆盖 YAML。

设计方式：

- YAML 中故意写入一个会导致校验失败的可覆盖标量值。
- 通过环境变量将其修正为合法值。
- 如果优先级正确，`validate` 成功；如果优先级错误，`validate` 失败。

### 4.3 `invalid-syntax`

目标：证明 YAML 语法错误会在第一层直接失败。

### 4.4 `invalid-missing-role`

目标：证明 `stage.role` 引用不存在角色时会在引用校验失败。

### 4.5 `invalid-shell-security`

目标：证明角色启用 `shell_exec` 但缺少 `security.shell` 时会被阻断。

### 4.6 `invalid-immutable-conflict`

目标：证明 `immutable_artifacts.paths` 与未授权可写路径冲突时会被阻断。

### 4.7 `invalid-threshold`

目标：证明 `loop_guard` / `deadlock_guard` / `master_watch` 阈值非法时会被阻断。

## 5. 核心用例矩阵

| 用例 ID | 场景 | 输入 | 期望结果 |
| --- | --- | --- | --- |
| CFG-E2E-001 | 最小合法配置校验 | `valid-minimal/octopus.yaml` | `validate` 退出码为 0 |
| CFG-E2E-002 | YAML 语法错误阻断 | `invalid-syntax/octopus.yaml` | `validate` 非 0，输出 `syntax` 类错误 |
| CFG-E2E-003 | 缺失角色引用阻断 | `invalid-missing-role/octopus.yaml` | `validate` 非 0，错误路径指向 `stages[*].role` |
| CFG-E2E-004 | shell 安全策略缺失阻断 | `invalid-shell-security/octopus.yaml` | `validate` 非 0，输出 `security` 类错误 |
| CFG-E2E-005 | 只读产物冲突阻断 | `invalid-immutable-conflict/octopus.yaml` | `validate` 非 0，输出只读冲突错误 |
| CFG-E2E-006 | 监控阈值非法阻断 | `invalid-threshold/octopus.yaml` | `validate` 非 0，输出 `policy` 类错误 |
| CFG-E2E-007 | 环境变量覆盖 YAML | `valid-env-override/octopus.yaml` + env | `validate` 成功 |
| CFG-E2E-008 | `run` 前置阻断 | 任一非法 YAML | `run` 非 0，且不创建 session 目录 |

## 6. 关键断言方式

### 6.1 `validate` 成功断言

- 进程退出码为 `0`。
- 输出中不包含错误级别摘要。
- 如支持默认值摘要输出，则断言至少出现一条已应用默认值记录。

### 6.2 `validate` 失败断言

- 进程退出码非 `0`。
- 输出中至少包含：
  - 错误类别，例如 `syntax` / `reference` / `security` / `policy`
  - 字段路径，例如 `stages[0].role`
  - 面向用户的修复建议

### 6.3 `run` 阻断断言

- 进程退出码非 `0`。
- `.octopus/sessions` 不存在，或保持为空。
- 不产生伪成功的 session 元数据、bus、roles 目录。

## 7. 推荐测试实现

### 7.1 `pytest` fixture

`conftest.py` 建议提供：

- `project_root`
- `compose_file`
- `run_cli(command: list[str], env: dict[str, str] | None = None)`
- `clean_environment()`
- `assert_no_session_created()`

### 7.2 CLI 调用方式

推荐统一用 Python `subprocess.run` 封装 CLI 黑盒调用，例如：

```python
result = run_cli([
    "openoctopus",
    "validate",
    "--config",
    "/work/e2e/config/fixtures/valid-minimal/octopus.yaml",
])
```

断言只关注：

- `returncode`
- `stdout`
- `stderr`
- 文件系统副作用

不读取内部 Go 结构体，不调用模块内私有函数。

## 8. 分阶段执行策略

### 8.1 第一批：config 单模块黑盒

这批测试在 `config 001` 完成后即可执行，聚焦：

- `validate` 成功/失败
- `run` 启动前阻断
- 优先级覆盖
- 默认值最小可运行

### 8.2 第二批：与 CLI / session 联动增强

待 `cli 001` 和 `session 001` 落地后，补充以下场景：

1. `validate --format json` 或等价输出中展示 `applied_defaults`。
2. `run` 成功时将有效配置快照写入 session 元数据。
3. 校验错误中输出 `rule_id`，并回链 `docs/config/001/yaml-rules.md`。
4. `reset-session`、`interrupt-all` 等入口对配置的复用一致性。

## 9. 通过标准

以下条件同时满足，才视为 `config 001` 的 E2E 方案达标：

1. 核心 8 个黑盒用例都可以在本地 Docker 环境稳定复现。
2. 同一非法 YAML 连续执行两次，失败结果类别与退出码一致。
3. `run` 的失败不会留下半初始化 session 垃圾目录。
4. 覆盖优先级、默认值、结构校验、引用校验、安全校验、策略校验六类核心能力。

## 10. 与后续模块的衔接

`config 001` 的 E2E 是整个平台 E2E 的起点，不是终点。它先证明“统一输入协议”可靠，后续模块再围绕这个入口扩展更长链路的黑盒验证：

- `session 001`：验证有效配置如何落到 `.octopus/` 目录。
- `event-bus 001`：验证配置驱动下的事件类型与落盘。
- `orchestrator 001`：验证配置能否驱动主循环编排。
- `role-runtime 001`：验证角色配置、工具权限和只读约束的执行效果。

也就是说，`config` 的 E2E 不追求一次把全链路测完，而是先把“入口是否可靠”做实。
