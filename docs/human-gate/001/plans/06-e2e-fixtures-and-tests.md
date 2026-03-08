# Human Gate 001 / Plan 06：E2E 夹具与黑盒测试

## 目标

建立 `human-gate 001` 的宿主机黑盒 E2E，验证中断、注入、恢复与批量等待人工这四条主链路。

## 范围

- 新增 `e2e/human-gate/` fixtures 和 pytest
- 复用现有 role-runtime harness
- 更新 `e2e/README.md` 与 `Makefile`

## 涉及文件

- `e2e/human-gate/fixtures/*`
- `e2e/human-gate/test_human_gate_interrupt.py`
- `e2e/human-gate/test_human_gate_resume.py`
- `e2e/README.md`
- `Makefile`

## 验收点

1. `python3 -m pytest e2e/human-gate -v` 通过。
2. `make check` 包含 human-gate E2E 且通过。
