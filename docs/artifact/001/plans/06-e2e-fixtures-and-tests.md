# Plan 06 - E2E 夹具与真实 Codex 验证

## 目标

建立 artifact 001 的黑盒 E2E，覆盖 deterministic 回归链路和真实 Codex 简单闭环。

## 产出

1. `e2e/artifact/fixtures/`
2. `e2e/artifact/test_artifact_pipeline.py`
3. `Makefile` 接入 artifact E2E
4. `make check` 全量验证

## 关键点

- deterministic 用例覆盖发布、输入解析、version bump
- 真实 Codex 用例只验证最小简单任务，不依赖自然语言细节
- 所有断言都落到 session 文件证据
