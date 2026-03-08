# Plan 03 - 输入解析与 context 投影

## 目标

让 orchestrator 在分发前能把 `input.ref` 解析成真实 artifact 路径，并把输出建议 ref 一起写入 `context.md`。

## 产出

1. `ResolveLatest(...)` 能力
2. stage artifact 输入绑定解析
3. `context.md` 中的 `input_artifacts` / `output_artifacts` 投影
4. 相应 orchestrator 单测

## 关键点

- unresolved input 必须显式失败
- output suggested ref 要稳定、可预测
- 保持现有 `context.md` leading values 协议不破坏
