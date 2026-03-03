# Plan 01: 项目初始化与环境搭建

## 目标
建立 UV 项目基础结构，完成依赖管理和开发环境配置。

## 任务清单

### 1.1 创建 UV 项目
- [ ] 执行 `uv init --python 3.10 openoctopus`
- [ ] 配置 `pyproject.toml`
  - [ ] 项目名称和版本
  - [ ] Python 版本要求 >=3.10
  - [ ] 包发现配置
  - [ ] 入口点脚本 `openoctopus = openoctopus.cli.main:app`

### 1.2 添加核心依赖
```toml
dependencies = [
    "pydantic>=2.0.0",
    "typer>=0.9.0",
    "rich>=13.0.0",
    "pyyaml>=6.0",
]
```

### 1.3 添加开发依赖
```toml
dev-dependencies = [
    "pytest>=7.0.0",
    "pytest-cov>=4.0.0",
    "pytest-asyncio>=0.21.0",
    "ruff>=0.1.0",
]
```

### 1.4 配置开发工具
- [ ] 配置 Ruff（.ruff.toml）
  - [ ] 行长度 100
  - [ ] 启用 isort 规则
  - [ ] 忽略特定规则
- [ ] 创建 `.gitignore`
- [ ] 创建 `README.md`（基础版本）

### 1.5 创建目录结构
```
openoctopus/
├── __init__.py
├── __main__.py
├── cli/
│   ├── __init__.py
│   ├── main.py
│   ├── commands/
│   │   ├── __init__.py
│   │   ├── init.py
│   │   ├── validate.py
│   │   ├── run.py
│   │   └── version.py
│   └── utils/
│       ├── __init__.py
│       ├── console.py
│       └── formatters.py
├── config/
│   ├── __init__.py
│   ├── models.py
│   ├── loader.py
│   ├── validator.py
│   └── templates.py
└── core/
    ├── __init__.py
    ├── constants.py
    └── exceptions.py
tests/
├── __init__.py
├── conftest.py
├── unit/
└── e2e/
    ├── __init__.py
    ├── test_init.py
    ├── test_validate.py
    ├── test_run.py
    ├── test_version.py
    └── test_help.py
```

## 验收标准
- [ ] `uv sync` 成功执行
- [ ] `python -m openoctopus --version` 可运行
- [ ] `openoctopus` 命令可用
- [ ] Ruff 检查无错误

## 预计耗时
30 分钟
