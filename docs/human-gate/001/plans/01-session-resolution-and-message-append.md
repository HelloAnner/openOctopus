# Human Gate 001 / Plan 01：Session 解析与人工消息追加

## 目标

先补齐 `human-gate` 最基础的两个能力：稳定定位 `--session` 对应目录，以及按正式协议向 `planner/human_messages.md` 追加消息块。没有这层基础，就无法可靠实现 `inject` / `resume`。

## 范围

- 新增 session 目录解析助手
- 新增 `nextMessageID` 规则
- 新增 `inject` 所需的消息块渲染/追加能力
- 覆盖 `--message` / `--input` 两种输入路径

## 涉及文件

- `internal/humangate/session.go`
- `internal/humangate/messages.go`
- `internal/humangate/service_test.go`

## 验收点

1. `--session` 传绝对目录时可直接使用。
2. `--session` 传 session id 时，可在当前工作目录下解析到 `.octopus/sessions/<id>`。
3. `planner/human_messages.md` 能生成连续的 `msg-000001`、`msg-000002`。
4. `inject` 追加的块包含 `source`、`kind`、`target_role_id`、`created_at`、`content`。
