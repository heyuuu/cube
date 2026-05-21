# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 构建与开发

```bash
# 构建
go build -o go-cube .

# 生成 Wire 依赖注入（修改 internal/app/wire.go 后需要执行）
go generate ./...

# 运行单个测试
go test ./internal/logger/ -run TestFormatFunc

# 格式化代码
goimports -w .
```

## 架构

Go-Cube 是面向开发者的本地多项目管理 CLI 工具，支持多工作区管理、HTTP API 服务以及 Alfred Workflow 集成。

**依赖注入**：使用 Google Wire（`internal/app/wire.go` → `wire_gen.go`）。修改 `wire.go` 后需执行 `go generate ./...` 重新生成。

**请求流程**：`main.go` → `cmd/`（Cobra 命令）→ `services/`（业务逻辑）→ `entities/`（领域模型）。`internal/app/app.go` 中的 `App` 结构体是 DI 容器，持有所有 Service。

**核心分层**：
- `cmd/` — CLI 命令，按领域分组：`project`、`application`、`remote`、`workspace`、`alfred`，以及 `server` 和 `config`
- `internal/services/` — 业务逻辑层。`ProjectService` 带有线程安全缓存；`HistoryService` 通过 GORM 使用 SQLite
- `internal/entities/` — 领域实体：`Project`、`Workspace`、`Application`、`Remote`、`ProjectScanner`
- `internal/handlers/` — HTTP API 处理器（挂载在 `/api/` 下）
- `internal/server/` — HTTP 服务启动与路由注册
- `internal/model/` — GORM 数据库模型（如 `ProjectHistory`）
- `internal/config/` — 配置加载，默认路径 `~/.go-cube/`（可通过 `--config` 覆盖）
- `internal/util/easycobra/` — 对 cobra.Command 的轻量封装
- `internal/util/matcher/` — 带加分机制的模糊搜索
- `internal/util/git/` — Git 工具（分支信息、worktree 支持）
- `internal/dto/` — API/Handler 层数据传输对象
- `internal/converter/` — Entity 与 DTO 之间的转换器

**启动顺序**：`rootPreExecute()` 解析全局 flags（`--config`、`--debug`），初始化配置和日志，之后才执行具体命令。命令通过 `app.InitApp()` 获取 Service 实例。
