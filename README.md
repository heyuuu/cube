# cube

## 介绍

面向个人开发者的本地多项目管理工具（Web UI 优先，CLI 简单可行兜底）。不做云服务、不绑 AI。

- 1.* 版本 — php 开发的纯命令行版本
- 2.* 版本 — go 语言重写的升级版本
- 3.* 版本 — 当前：按领域重构 + Web 工作台

## v3 现状

- **架构**：Go 1.27，module path `cube`（go.mod 在仓库根）；`cmd/`（CLI 出口与 devtool 脚手架）、`internal/`（domain / handlers / server / app / config 等内部实现）、`pkg/`（git、pathkit、tui 等独立工具包）分目录，依赖单向（详见 `docs/spec/现状.md`）
- **出口**：`cube server` 本地 Web HTTP server（huma + 标准 ServeMux，Scalar 文档 + OpenAPI 3.1 自动生成）；CLI 人用 / alfred 集成 / `p`·`pz` shell 函数
- **前端**：仓库根 `web/`（Vite + React 19 + React Compiler + Tailwind 4 + Base UI 风味 shadcn），构建产物 `web/dist` 经 go:embed 嵌入单二进制
- **装配**：`internal/app` 显式接线（`app.New()` 空壳 + `Init` 组装 DI 容器，`github.com/heyuuu/x/di`）；文件存储原语用 `github.com/heyuuu/x/store`
- **配置**：`~/.config/cube/`（开发环境为 `cube-dev`），config.json（程序配置）与 settings.json（用户可管理数据）分离，运行期状态分 `state/` / `cache/` / `log/`
- **工具链**：仓库根 justfile（build / install / fmt / lint / test / dev / gen）；脚手架 devtool 的 `just gen domain` 自动完成新领域的「五处加法」

## 常用命令

```bash
just dev      # 开发：air 热重载 + vite（concurrently 同起）
just build    # 构建前端 + go build（产物 tmp/main）
just install  # go install + 安装校验 + 停旧服 + zsh 补全
just lint     # golangci-lint
just test     # go test -race ./...
```

详见 `AGENTS.md` 与 `docs/spec/现状.md`。
