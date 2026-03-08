# OpenOctopus E2E

## 运行方式

本仓库当前的 `config`、`session`、`eventbus`、`orchestrator`、`role-runtime`、`artifact`、`human-gate`、`cli` 模块真实 E2E 默认直接在宿主机运行，不使用 Docker。

测试会：

1. 复用当前用户真实 `~/.codex/` 环境。
2. 在仓库根目录下创建 `e2e-test/` 作为真实测试工作目录。
3. 使用 `go build` 构建当前仓库的 `openoctopus` 二进制。
4. 对 `eventbus` 额外构建一个测试专用 `eventbus-harness` 二进制。
5. 对 `orchestrator` 额外构建一个测试专用 `orchestrator-harness` 二进制。
6. 对 `role-runtime` 额外构建一个测试专用 `role-runtime-harness` 二进制。
7. 用 `pytest` 黑盒调用 `validate` / `run` / `status` / `interrupt` / `interrupt-all` / `inject` / `resume` 命令和 harness 子命令。

## 前置条件

- 本机已安装 `go`
- 本机已安装 `codex`
- 当前用户 `~/.codex/auth.json` 已存在

## 执行命令

只跑 `config`：

```bash
python3 -m pytest e2e/config -v
```

只跑 `session`：

```bash
python3 -m pytest e2e/session -v
```

只跑 `eventbus`：

```bash
python3 -m pytest e2e/eventbus -v
```

只跑 `orchestrator`：

```bash
python3 -m pytest e2e/orchestrator -v
```

只跑 `role-runtime`：

```bash
python3 -m pytest e2e/role-runtime -v
```

只跑 `cli`：

```bash
python3 -m pytest e2e/cli -v
```

只跑 `human-gate`：

```bash
python3 -m pytest e2e/human-gate -v
```

全量 E2E：

```bash
python3 -m pytest e2e/config e2e/session e2e/eventbus e2e/orchestrator e2e/role-runtime e2e/artifact e2e/human-gate e2e/cli -v
```

或：

```bash
make check
```
