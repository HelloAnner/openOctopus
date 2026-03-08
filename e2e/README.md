# OpenOctopus E2E

## 运行方式

本仓库的 `config` 模块真实 E2E 默认直接在宿主机运行，不使用 Docker。

测试会：

1. 复用当前用户真实 `~/.codex/` 环境。
2. 在仓库根目录下创建 `e2e-test/` 作为真实测试工作目录。
3. 使用 `go build` 构建当前仓库的 `openoctopus` 二进制。
4. 用 `pytest` 黑盒调用 `validate` / `run` 命令。

## 前置条件

- 本机已安装 `go`
- 本机已安装 `codex`
- 当前用户 `~/.codex/auth.json` 已存在

## 执行命令

```bash
python3 -m pytest e2e/config -v
```

或：

```bash
make check
```
