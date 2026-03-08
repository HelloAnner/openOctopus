# Config 001 拆分实现计划

## 概述

本文档将 `config 001` 的实现拆成 6 个可串行执行的小计划，目标是先把 OpenOctopus 的统一输入协议做稳，再逐步接入 CLI 和 E2E。

与旧的“大一统阶段计划”不同，这里只聚焦 `config` 模块，不跨到 orchestrator、session、role-runtime 的实现细节。

## 计划列表

| 编号 | 文件 | 目标 | 依赖 |
| --- | --- | --- | --- |
| 01 | `01-config-models.md` | 落地强类型配置模型与基础常量 | - |
| 02 | `02-loader-and-override.md` | 落地 YAML / env / flags 合并链路 | 01 |
| 03 | `03-defaults-and-effective-config.md` | 落地默认值注入与默认值记录 | 01, 02 |
| 04 | `04-validator-and-error-model.md` | 落地分层校验、错误模型和 `rule_id` 映射 | 01, 02, 03 |
| 05 | `05-cli-validate-and-run-gate.md` | 将 `config` 接入 `validate` / `run` 入口 | 01-04 |
| 06 | `06-e2e-fixtures-and-tests.md` | 建立 config 首版黑盒 E2E | 01-05 |

## 推荐执行顺序

```text
01 强类型模型
  ↓
02 加载与覆盖
  ↓
03 默认值与有效配置
  ↓
04 校验器与错误协议
  ↓
05 CLI 接入与启动阻断
  ↓
06 E2E 夹具与黑盒验证
```

## 统一约束

- 代码实现以 Go 为准，优先采用 `Koanf`、`go-playground/validator`、`Cobra`。
- 必须同步维护 `docs/config/001/yaml-rules.md`，不能让文档和实现分叉。
- 不允许 `git add`、`git commit`。
- 未来若新增 `config 002`，不得覆盖本目录文件，应新建 `docs/config/002/`。
