# Plan 02 - 版本发布与 manifest/diff

## 目标

落地把 source ref 快照成正式 artifact version 的核心能力，包括 version 递增、manifest 生成、diff 摘要和 content hash。

## 产出

1. `Publish(...)` 主流程
2. 文件与目录 source 的快照能力
3. `manifest.md` / `diff.md` 渲染
4. version 递增与 latest 解析单测

## 关键点

- 文件走 `content.*`
- 目录走 `content/`
- 第二版开始必须能引用上一版信息生成差异摘要
