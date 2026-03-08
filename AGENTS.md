## 1. 项目概况

- OpenOctopus 当前是一个基于 Go 实现的本地 CLI 原型，核心目标是把多角色协作、配置校验与会话落盘能力先跑通。
- 现阶段已落地的主能力集中在两部分：一是 `validate` / `run` 命令；二是 `octopus.yaml` 的加载、默认值注入、校验与 session 骨架目录创建。
- 仓库当前以后端 CLI、配置模型、文档设计和 E2E 验证为主，UI 设计稿目录尚未建立，不应把 `ui/` 视为当前已存在目录。
- 平台整体 PRD 采用版本化管理；当前生效的整体 PRD，应始终以 `docs/prd/` 下序号最大的 `prd-{NNN}.md` 为准。按当前仓库现状，这个文件是 `docs/prd/prd-001.md`。

## 2. 目录说明

- `cmd/openoctopus/`：CLI 入口与子命令实现，当前包含 `validate`、`run` 等命令与对应测试。
- `internal/config/`：配置域核心实现，按 `model`、`loader`、`defaults`、`validator`、`service`、`errors` 分层组织。
- `internal/session/`：session 工作目录骨架创建逻辑，当前负责创建会话目录和基础 `metadata.md`。
- `docs/`：产品与模块设计文档目录；包含平台整体 PRD、各模块 PRD、版本化设计文档与 `timeline.md`。
- `docs/prd/`：平台整体 PRD 历史版本目录；**当前版本永远取序号最大的 `prd-{NNN}.md`**，不要再把 `docs/prd.md` 当成最新入口。
- `e2e/`：真实链路 E2E 测试目录，包含 Python + Playwright + requests 测试、测试配置和 fixtures。
- `e2e-test/`：本地 E2E 运行产生的测试工作目录与临时产物目录，不属于正式源码实现目录。
- `Makefile`：仓库统一检查入口；任务结束前必须执行 `make check`。

### 3.6 UI 文档规范

- 当前仓库**尚不存在** `ui/` 目录；只有在后续正式引入 UI 设计稿时，才按以下规范新增并维护。
- `ui/ui.md` 是平台 UI 全局总纲，负责统一空间划分、设计语言、配色 Token、页面模板和交互基线。
- `ui/main.pen` 是平台级总入口与整体设计稿，用于承载全局入口、跨模块关系、空间切换和整体信息架构。
- `ui/{module}/main.pen` 是模块级总稿，用于汇总该模块全部子页面、主流程、列表/详情/弹窗关系和模块内导航骨架。
- `ui/` 下每一个**最终子页面**必须使用版本化目录：`ui/{module}/{page}/{version}/ui.md`。
- 首个版本统一从 `001/` 开始，后续迭代在同级新增 `002/`、`003/`，禁止直接覆盖旧版本。
- 模块目录只用于归类；真正可评审、可演进的页面稿以最终 `.../{version}/ui.md` 为准。
- 设计交付采用双层 `main.pen`：一层是 `ui/main.pen`，一层是 `ui/{module}/main.pen`；二者与对应页面稿必须保持一致。
- 新增模块页面或调整模块内入口时，优先更新 `ui/{module}/main.pen`；如果影响平台级入口、空间结构或跨模块导航，必须同步更新 `ui/main.pen`。
- 新增或调整页面交互时，必须同时检查是否与 `ui/ui.md` 总纲保持一致。

### 3.7 Docs 版本目录规范

- `docs/timeline.md` 是 `docs/` 目录的版本演进时间线，负责以树状结构维护每个模块当前已有的文档阶段。
- 平台整体 PRD 的**当前版本认定规则**：始终以 `docs/prd/` 下序号最大的 `prd-{NNN}.md` 为准；当前仓库对应文件为 `docs/prd/prd-001.md`。
- 每当新增模块文档目录，或新增 / 调整 `docs/{module}/{00x}/` 版本目录时，必须同步更新 `docs/timeline.md`。
- `docs/timeline.md` 中每个版本节点必须固定写两句话：
  - 第一句：这一阶段“做什么”。
  - 第二句：相对上一阶段“核心改进点 / 修改差异是什么”。
- `001` 作为首版时，第二句必须明确“建立了什么首版基线、替换了什么旧做法或原型做法”。
- 如果某模块当前只有 `docs/{module}/prd.md`、还没有 `001/`，也必须在 `docs/timeline.md` 中标记“尚未进入版本化阶段”。
- `docs/timeline.md` 的目录树必须与仓库真实 `docs/` 目录保持一致，不能只补描述不核对实际结构。
- 平台整体 PRD 不再使用 `docs/prd.md` 作为持续更新入口；统一在 `docs/prd/` 下按 `prd-{NNN}.md` 维护历史与当前版本，`NNN` 从 `001` 开始连续递增。
- 当用户提出新的整体需求，并明确要求保存为新的 `docs/prd/prd-{NNN}.md` 时，必须先阅读 `docs/prd/` 下已有历史版本，基于上一版和历史变更链路做增量设计，禁止脱离历史上下文直接重写一份全新 PRD。
- 编写新的平台整体 PRD 时，必须明确上一版保留了什么、废弃了什么、新增了什么，以及这些调整分别影响哪些模块边界、职责或交付顺序，保证版本之间可追溯。
- 新的 `docs/prd/prd-{NNN}.md` 完成后，必须再阅读一次 `docs/timeline.md`，结合新的整体设计判断哪些模块需要新增一个版本目录或补充下一阶段实现，并在最新 PRD 文末单独写清楚。
- 最新平台整体 PRD 文末必须固定增加“模块落地影响与执行顺序”一节，至少包含两部分：一是受本次设计影响、需要新增版本实现的模块清单；二是按依赖关系和执行优化后的推荐落地顺序，并说明这样排序的原因。

---

## 4. E2E 测试偏好

### 4.1 核心原则

- **真实环境优先**：E2E 必须基于真实本地 Docker 启动的完整服务，禁止 Mock 服务端。
- **黑盒验证**：通过 HTTP API + 浏览器前端交互验证，不依赖内部实现细节。
- **干净环境**：每次运行前清理旧的测试工作目录；如使用宿主机直跑方案，默认清理仓库根目录下的 `e2e-test/`。
- **自动化**：使用 **Python 脚本 + Playwright**（agent-browser）实现，CI 可直接执行。
- **Codex 环境复用**：凡是 E2E 涉及 `codex` CLI、role runtime 中的 Codex 执行链路、或任何读取 Codex 本地配置的场景，必须直接复用宿主机真实 `~/.codex/` 环境，禁止伪造 token、禁止使用假的 `CODEX_HOME` 替代真实目录，除非用户明确要求做隔离副本。
- **宿主机优先原则**：如果被测链路依赖宿主机已登录的 Codex CLI 状态，优先在宿主机直接执行测试，并使用仓库根目录下的 `e2e-test/` 作为真实测试工作目录；无特殊要求时不要再引入 Docker。

### 4.2 工具选型

| 工具                   | 用途                                      |
| ---------------------- | ----------------------------------------- |
| `pytest`               | 测试框架，用例组织与断言                  |
| `requests`             | HTTP API 测试（登录、接口调用、状态轮询） |
| `playwright`（Python） | 浏览器自动化，测试前端 UI 交互            |
| `docker compose`       | 仅在用户明确要求容器化 E2E 时使用         |
| `codex` CLI            | 真实验证本机 Codex 运行环境与配置装载     |

### 4.3 E2E 文件结构规范

```
e2e/
├── requirements.txt              # Python 依赖（pytest/playwright/requests）
├── conftest.py                   # session 级 fixtures：binary_path / codex_home / workspace_dir
├── docker-compose.test.yml       # 仅在用户明确要求容器化 E2E 时保留
├── config/
│   ├── config.test.yaml          # 测试配置（如涉及 Codex，需显式声明 `CODEX_HOME` / `codex` 路径策略）
│   └── .env.test                 # 测试环境变量（可指向宿主机 `~/.codex/`）
├── {module}/                     # 按模块分目录（infra / iam / task / ...）
│   ├── test_xxx.py               # 测试文件
│   └── ...
└── README.md                     # 快速上手（如何安装依赖、如何运行）
```

### 4.4 Codex 相关 E2E 补充约束

- 只要测试链路里出现 `provider: codex`、`cli_path: codex`、或任何依赖 `~/.codex/` 的读取行为，就必须把这条链路视为“真实 Codex E2E”，不能退化成纯 mock。
- 如果测试在宿主机直接运行，优先读取当前用户真实 `~/.codex/`，并将真实测试产物、临时配置、session 目录统一写入仓库根目录 `e2e-test/`。
- 如果测试在 Docker 内运行，必须至少保证以下挂载：
  - `~/.codex/auth.json`
  - `~/.codex/config.toml`
  - `~/.codex/prompts/`
  - `~/.codex/skills/`
- 涉及真实 Codex 环境的测试必须避免写坏宿主机配置；如测试需要写入，优先写测试工作目录，禁止覆盖 `~/.codex/` 内已有配置文件。

## 6. 任务收尾要求

- 每一次任务全部结束时，必须执行 `make check` 保证基础的规则和检查没问题 和 make install 。
- 每一次对话结束前，凡是本次任务涉及目录说明、目录索引、README 树、模块结构树、ASCII 目录树等内容，都必须再次核对，保证文档中的目录结构树与仓库真实目录一致；如果本次修改了目录结构，必须同步更新对应说明文档。
