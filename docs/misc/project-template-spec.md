# Go 后端 + Web 前端 · 项目设计规范（通用模板）

> 本文档是「Golang 后端 + 本地 Web 前端」这类**个人开发工具项目**的通用设计规范。它从 [cube](../) 项目提炼，把 cube 验证过有效的架构范式抽象成可复用模板。
>
> **定位**：新项目立项时，照此模板抄目录结构、技术选型、分层纪律、前后端契约；项目自身的行为规约（现状快照 + 需求提案）另写文档（见第十节）。本文是**项目骨架**的规范，不是产品功能规范。
>
> **维护节奏**：这份模板随 cube 演进迭代。

---

## 一、适用场景

**适合**：

- 个人/小团队的**本地工具**（CLI + 本地 Web server，非云服务）
- 后端用 Go，前端用现代构建链（Vite + React + TS）
- 前后端通过 **OpenAPI 契约**对接（后端生成 → 前端消费）
- 单一可执行二进制（前端 `go:embed` 进二进制）

**不适合**：

- 纯云服务 / 多租户 / 需要认证授权的 Web 产品
- 移动端、SSR、高并发公网服务
- 后端非 Go 的项目（模板的分层、依赖纪律是 Go 语境的）

**典型例子**：本地多项目管理、本地 db 查看、端口/进程查看、本地文件批处理、开发环境编排——即「反复手动操作 → 收进工具」这一类个人辅助工具。

---

## 二、推荐技术栈

### 2.1 后端

| 能力 | 选型 | 理由 |
|---|---|---|
| HTTP 框架 | **[huma/v2](https://github.com/danielgtaylor/huma)** + `humago` adapter | 自动生成 OpenAPI 3.1，自动挂 `/openapi.json` `/docs`，强类型输入输出，零手写路由文档。是「后端生成 OpenAPI 给前端」工作流的源头。 |
| CLI 框架 | **[cobra](https://github.com/spf13/cobra)** | Go CLI 事实标准。子命令分组、flag、补全都顺手。 |
| 日志 | **标准库 `log/slog`** | 不引第三方。文件通道 JSON handler + debug 时 tint 彩色 stdio 双通道。 |
| 配置 / 持久数据 | JSON 文件（config.json 按 domain 分节 + settings.json 多节开放结构） | 个人工具不需要 yaml/toml 的复杂度，JSON 手改方便、读写简单。无热 reload——配置改动重启生效，用户可管理数据走各域 save API 即时生效。 |
| 持久化 | **文件存储**（`[heyuuu/x/store](https://github.com/heyuuu/x)`：原子写 / 缩进 JSON / JSONL 流式迭代），不引数据库 | 个人工具的数据规模（KB~MB 级）文件足够；sqlite/ORM 换来的是迁移、锁、驱动平台差异等一整类问题。重读场景配内存缓存。 |
| 依赖装配 | **`heyuuu/x/di`**（极简 DI 容器） | 显式注册清单 + 按类型解析，替代散落的手工接线；无全局单例。 |
| git 操作（如需） | **系统 `git` 单轨**（读输出机器可读格式并类型化解析；写透传 stdio） | 行为与用户日常 git 完全一致、自带 index 缓存；纯 Go 库（go-git）在大仓库上有忽略链/性能偏差。读写都收敛在单个 util 包内。 |
| TUI（如需） | **[huh/v2](https://github.com/charmbracelet/huh)** + **[lipgloss/v2](https://github.com/charmbracelet/lipgloss)** | 交互式表单/选择、表格样式。 |

**版本基线**（参考 cube，新项目取最新稳定版）：
```
go 1.27+
github.com/danielgtaylor/huma/v2  v2.39+
github.com/spf13/cobra            v1.10+
```

### 2.2 前端

| 能力 | 选型 | 理由 |
|---|---|---|
| 构建链 | **[Vite](https://vitejs.dev)** | 快、标准、生态顺。`dev` 热重载 + `build` 产物给 go:embed。 |
| 框架 | **[React](https://react.dev)** + **TypeScript** | 组件生态最大；Monaco/xterm/复杂 UI 都有现成绑定。 |
| UI 组件 | **[shadcn/ui](https://ui.shadcn.com)**（Base UI 底座，非 Radix）+ **Tailwind CSS v4** | 源码进项目、可改、无运行时依赖负担。 |
| 数据请求 | **[TanStack Query](https://github.com/TanStack/query)** | 与 RESTful + OpenAPI 生成的类型客户端契合，缓存/loading/error 省心。 |
| 路由 | **react-router**（library mode，路由表集中在 `App.tsx`） | history 路由配 server 的 SPA fallback。 |
| API 类型 | **后端 OpenAPI 生成 → 前端类型/客户端代码生成** | 见第五节，核心工作流。 |

**可选前端库**（按需）：
- 文件树：[react-arborist](https://github.com/brimdata/react-arborist)
- 代码编辑器：[Monaco Editor](https://github.com/microsoft/monaco-editor) + `@monaco-editor/react`
- 终端：[xterm.js](https://xtermjs.org)
- 图表/复杂表格等：按需引入

### 2.3 工具链

| 能力 | 选型 |
|---|---|
| Go 格式化 + lint | `golangci-lint fmt` / `golangci-lint run`（justfile 收敛为 `just fmt` / `just lint`） |
| 热重载（开发） | [air](https://github.com/cosmtrek/air)，pre_cmd 内置 `just lint` 预检 |
| 前端热重载 | Vite dev server，开发期走 proxy 到后端 |
| 任务编排 | **[just](https://just.systems)**（justfile：build / install / fmt / lint / test / dev / run / gen）；Makefile 只留 `tag` 等与 git 交互的动作 |
| 版本号注入 | `go build -ldflags`（justfile 注入 git tag/commit/buildTime 到 `internal/version` 包变量） |
| 脚手架 | 独立 devtool 二进制（`just gen domain <name>` 自动完成加新领域的全部加法，含装配清单注入点） |
| API 文档 | huma 内置 Scalar 渲染（`/docs`） |

---

## 三、分层架构与依赖纪律

这是模板的**核心纪律**。Go 后端必须保持单向依赖，禁止循环。

```
基础设施  internal/config / internal/logger / internal/version            所有层共享，不依赖上层
独立工具  pkg/*（git / pathkit / slicekit / fuzzy / tui ...）             通用动作，不含业务实体，只依赖标准库/第三方
能力      internal/server                                        服务端框架，不含业务实体
领域      internal/domain/<domain>/（含实体、规则、service）            业务核心，依赖能力+基础设施+pkg
出口      internal/cmd / internal/handlers / web                    把领域包成 CLI / Web
装配      internal/app / cmd/<app>/main                            唯一接线点
```

### 依赖规则

1. **基础设施层**（config/logger/version）：不 import 任何上层。
2. **独立工具层**（pkg/*）：只依赖标准库/第三方，不含业务实体。子包必须**内聚且无副作用**——只对入参做纯运算，不读进程状态/环境。与环境强相关的副作用（`os.Getwd`、`os.Getenv`、展开 `~`、当前时间）只允许出现在职责就是处理环境的包里（如 `pathkit` 展开 `~`、`config` 读配置目录）。
3. **领域层**（`internal/domain/*`）：依赖能力 + 基础设施 + pkg。domain 之间尽量不互相依赖；确需共享实体，下沉到 pkg 或基础设施层。
4. **出口层**：`internal/cmd`（CLI）与 `web`（前端 embed）不互调；cmd 经 server 框架的 client 子包走 localhost HTTP 探活/关停属进程间通信，不算互调。业务 HTTP handler 统一在 `internal/handlers`（同包按 domain 分文件），依赖领域层与 server 框架。
5. **装配层**（`internal/app`）：唯一接线点。`app.New()` 返回空壳，root 命令的 `PersistentPreRunE` 调 `a.Init(...)` 完成组装（config 加载 → logger → DI 容器注册）——显式一次性构造 + CLI 侧惰性触发。**无全局单例、无 `app.Default()`、不在包级 `init()` 里反向依赖未就绪的服务**。

### 加新 domain 的「五处加法」流程

加一个新领域，是**纯加法**，不动既有 domain：

1. 领域包：`internal/domain/<domain>/`（实体 + service + 规则）
2. CLI 子命令：`internal/cmd/<domain>.go` + root.go 注册
3. Web handler：`internal/handlers/<domain>_handler.go`，实现 `Handler.Register(r *server.Routes)`
4. config 加节（如需）：在 `config.Config` 加字段 + 默认值
5. 装配清单：`internal/app/boot.go` 的 serviceFactories / handlerFactories 加构造

> 五处都是加法是纪律的核心——**新增不污染既有 domain**。如果加新 domain 必须改既有 domain，说明领域边界划错了，先重构边界。做法已工具化：devtool 的 `just gen domain <name>` 以 boot.go 内的注入点标记为锚点自动完成五处。

### domain 内部结构（建议）

```
internal/domain/<domain>/
  service.go           // Service 聚合根：struct 定义 + 全部公开方法（与 struct 同文件）
  types.go             // 跨主题复用的类型 / 常量 / 构建解析函数
  <主题>.go            // 复杂逻辑内聚的主题文件（如 scan.go / clone.go / diff.go），包级函数暴露
  helpers.go           // 跨主题复用的内部小工具
  <domain>_test.go     // 表驱动测试
  <子包>/              // 重的子能力下沉为子包（如 projcache/）
```

---

## 四、目录结构规范

### 4.1 后端（Go）

```
<project>/                     # go.mod 在仓库根
├── cmd/
│   ├── <app>/main.go          # 入口，调用 cmd.Execute()
│   └── devtool/               # 脚手架工具（generate / tag 等开发命令）的独立入口
├── internal/
│   ├── app/                   # 装配层：app.go（App/Init）、boot.go（DI 注册清单 + 注入点）、hooks.go（生命周期钩子）
│   ├── cmd/                   # CLI 出口：root.go + 各命令文件（收 *app.App）
│   ├── config/                # 基础设施：config 加载/默认值 + settings 节级 API + Paths 路径集
│   ├── logger/                # 基础设施：slog handler（文件 + debug stdio 双通道）
│   ├── version/               # 基础设施：版本号（ldflags 注入）+ AppName 派生
│   ├── server/                # 能力层：Server / Routes / envelope / TypedHandler（client/ 子包管进程探活关停）
│   ├── domain/<domain>/       # 领域层：每个 domain 一个包
│   ├── handlers/              # 业务 HTTP handler 层：<domain>_handler.go 同包分文件 + system/static handler
│   └── testfixture/           # 测试辅助（建临时仓库/目录）
├── pkg/                       # 独立工具层：git / pathkit / slicekit / fuzzy / tui 等子包
├── web/                       # 前端工程 + embed 容器（embed.go，见 §8.2）
├── justfile                   # build / install / fmt / lint / test / dev / run / gen
├── Makefile                   # 仅 tag / 发号等动作
├── .air.toml                  # air 热重载配置
├── scripts/                   # shell 集成等附加脚本
└── docs/                      # 文档（现状快照 + 提案，见 §10）
```

### 4.2 前端（Vite + React）

```
web/
├── embed.go                 # //go:embed all:dist → FS()（独立 Go 包，不必在 main 包）
├── package.json / pnpm-lock.yaml
├── vite.config.ts           # build.outDir = 'dist'，产物给 go:embed；dev server proxy /api 到后端
├── index.html               # Vite 入口
├── src/
│   ├── main.tsx             # React 挂载
│   ├── App.tsx              # 路由表（单一数据源）
│   ├── pages/<name>/        # 每页一个目录（<Name>Page.tsx + index.ts）
│   ├── components/ui/       # shadcn 组件（vendored）
│   ├── api/                 # OpenAPI 生成的类型（schema.d.ts）+ 类型安全 client
│   ├── lib/ hooks/          # 工具函数 / 全局 QueryClient / hooks
│   └── index.css            # Tailwind v4 与主题变量（CSS-first，无 tailwind.config）
└── dist/                    # 构建产物（gitignore），go:embed 目标
```

**关键约定**：
- 前端源码与构建产物都在 `web/` 下，`web/dist/` 是 embed 目标，进 `.gitignore` 不入库。
- **clone 后首次 go build 前必须先构建前端**（`just build-ui`），否则 embed 找不到 dist 报错。

---

## 五、前后端契约：OpenAPI 工作流 ⭐

**这是模板的枢纽。** 前后端通过 OpenAPI 文件对接，不手写接口类型，不靠口头约定。

### 5.1 工作流总览

```
┌─────────────┐   huma 自动生成    ┌──────────────┐   代码生成      ┌──────────────┐
│ Go 后端      │ ───────────────▶ │ openapi.json │ ──────────────▶ │ 前端 TS 类型  │
│ (huma v2)   │                   │ (契约源)      │                 │ + API 客户端 │
└─────────────┘                   └──────────────┘                 └──────────────┘
       ▲                                                                  │
       │                                                                  ▼
       └──────────────── 实际 HTTP 请求（运行时） ─────────────────────────┘
```

### 5.2 后端：声明即契约

huma v2 的核心价值：**Go struct 的 tag 就是 OpenAPI schema 的来源**。写 handler 时声明输入输出 struct，OpenAPI 自动生成。

**统一响应信封**（强制，所有 API 走同一形态，`internal/server/routes.go`）：
```go
// JSONOutput api 标准输出结构
type JSONOutput[T any] struct {
    Body Envelope[T]
}

// Envelope api 标准输出的信封：ok 标记成败、message 携带失败原因、data 携带负载数据。
type Envelope[T any] struct {
    Ok      bool   `json:"ok"`
    Message string `json:"message"`
    Data    *T     `json:"data"`
}
```

**注册口**（强制 `/api/` 前缀，自动从 path 推导 tag + operationId；handler 只写 `func(I) (O, error)`，envelope 由 `JSONHandler` 包装）：
```go
// internal/handlers/<domain>_handler.go
func (h *ProjectHandler) Register(r *server.Routes) {
    r.Get("/api/project/list", "查询项目列表", server.JSONHandler(h.list))
    r.Post("/api/project/save", "保存项目", server.JSONHandler(h.save))
}
```

**序列化**：响应 JSON 走 `encoding/json/v2`——nil 切片序列化为 `[]`、nil map 为 `{}`，前端不会拿到 null 崩溃；handler 里不要手写 `make([]T, 0)` 兜底。

**huma 自动暴露**（无需手写路由）：
- `GET /openapi.json` / `/openapi.yaml` — OpenAPI 3.1 规范
- `GET /docs` — Scalar 渲染的交互式文档

**CLI 导出**（给前端代码生成用，独立一级命令）：
```bash
<app> openapi -o openapi.json   # 产物 gitignore，不入库
```

### 5.3 前端：类型与客户端生成

从 `openapi.json` 生成前端 TS 类型 + API 客户端。推荐组合（cube 实测）：**openapi-typescript**（纯类型 `.d.ts`）+ **openapi-fetch**（类型安全 fetch 封装）+ TanStack Query——类型安全、无重运行时、与 Query 契合。

**生成脚本**（`web/package.json`，cube 实际形态）：
```json
{
  "scripts": {
    "gen:api": "just -f ../justfile run openapi -o openapi.json && openapi-typescript openapi.json -o src/api/schema.d.ts"
  }
}
```

**生成产物 gitignore**（`src/api/schema.d.ts` 与 `openapi.json` 均不入库，由生成得到）。

### 5.4 契约纪律

1. **openapi.json 是契约唯一源**。后端改了 handler → 重新导出 openapi.json → 前端重新 gen:api。
2. **前端不手写接口类型**，一律由生成来。手写 = 漂移风险。
3. **响应统一走 Envelope 信封**，前端封装一层 unwrap。
4. **API 只用 GET / POST**：GET = 查询（query 传参）、POST = 动作（body 传参）；保存/删除语义放进 API 名（`xxx/save`、`xxx/delete`），不引入 PUT/DELETE/PATCH。
5. **错误码**：HTTP 状态码 + Envelope.message；业务错误用标准 HTTP 码（400/404/409），不要全 200 靠 body.ok 区分。

### 5.5 开发期联调

- **后端热重载**：`air`（改 Go 代码自动 rebuild + restart server，pre_cmd 跑 `just lint` 预检）。
- **前端热重载**：`vite dev`（dev server，proxy `/api` 到后端端口）。`just dev` 用 concurrently 把两者一起拉起。
- **契约同步**：改后端 handler 后，重跑 `<app> openapi -o openapi.json` + 前端 `pnpm gen:api`。

---

## 六、配置管理

### 6.1 配置目录

默认 `~/.config/<app>/`（开发环境 `~/.config/<app>-dev/`，由 ldflags 注入的 version 推导；`--config` flag 覆盖文件路径）。结构：
```
~/.config/<app>/
├── config.json          # 主配置（按 domain 分节，程序配置）
├── settings.json        # 用户可管理数据（openers / 规则等，多节开放结构，运行中可变更）
├── state/               # 运行期状态（日常使用产生，丢了可接受但不理想）
├── cache/               # 纯缓存（可整体删除且行为不变差）
└── log/app.log          # 日志
```

**规则**：config.json 与 settings.json 是**配置/数据**（换机器要带走）；state/ cache/ log/ 是**运行期状态**（程序管）。路径一律由 config 包 `Config` 的路径方法统一计算（自 `DataDir` 派生），调用方不硬拼。

### 6.2 config.json 分节

按 domain 分节，每个 domain 一个 struct：
```go
type Config struct {
    DataDir string       `json:"dataDir"` // 缺省取配置文件所在目录
    Server  ServerConfig `json:"server"`
    // 加新 domain：在此加字段
}
```

### 6.3 配置纪律

- **降级优先**：config.json 缺失降级为默认值（不阻断启动）、JSON 解析失败报错退出；settings.json 各节读失败降级为空，坏数据跳过不阻断。
- **原子写**：保存一律 tmp + rename（`heyuuu/x/store` 的 `WriteFileAtomic` / `SaveJson`），避免半写状态。
- **无热 reload**：配置改动重启生效（热 reload 的 fsnotify 监听 + Reload 链路已移除——隐藏行为多、状态同步边界模糊）；settings.json 用户数据走各域 save API 即时生效。
- **路径展开**：`~/xxx` 等相对形态的展开收敛在 CLI 出口层（如 `ExtendPath`），底层包只认绝对路径、不做基于 cwd 的隐式解析。

---

## 七、日志

**统一用标准库 `log/slog`**，禁止用 `fmt.Println` 做日志（`fmt` 仅用于面向用户的 CLI 输出）。

### 7.1 双通道 handler

- **文件通道**：始终启用，JSON handler 落 `<dataDir>/log/app.log`，完整结构化日志。
- **stdio 通道**：仅 debug 模式（`--debug` flag）启用，tint 彩色输出到 stderr（非 TTY 自动去色）。

### 7.2 日志格式

- debug 日志受 `--debug` 控制；非 debug 只输出 Info 及以上。
- 日志文件名是通用的 `app.log`，不含应用名——应用标识（`AppName`）收敛在 version 包常量，配置目录名 / status 身份 / 进程探测 / shutdown token 均由它派生，多项目复用骨架时只改这一处。

---

## 八、构建与发布

### 8.1 任务编排（justfile）

```just
version := `git describe --tags --abbrev=0`
ldflags  := "-X <module>/internal/version.version=" + version + " ..."

build: build-ui
    go build -ldflags "{{ldflags}}" -o tmp/<app> ./cmd/<app>

install: build-ui
    go install -ldflags "{{ldflags}}"
```

**版本注入**：`internal/version` 包暴露变量，justfile 用 ldflags 注入 git tag/commit/buildTime。`<app> version` 命令 + huma 的 OpenAPI `info.version` 都用它；未注入（源码直跑 / air）即视为 dev 环境。

### 8.2 go:embed 前端产物

独立 `web` 包承载 embed（`//go:embed` 不能引用上级目录，放在前端工程内的独立包最省事）：
```go
// web/embed.go
package web

import ("embed"; "io/fs")

//go:embed all:dist
var embedded embed.FS

// FS 返回前端产物根（dist/ 下内容提升到根，StaticHandler 直接按相对路径读）。
func FS() fs.FS { ... }
```

装配层把 `web.FS()` 注入 StaticHandler（handlers 包持有 `fs.FS`，不 import web）：
```go
// internal/app/boot.go
di.Value(handlers.NewStaticHandler(web.FS())),
```

**纯后端模式**：启动时读一次 `index.html`，读不到（前端未构建）则不挂页面路由、只提供 API，日志提示先构建前端。SPA fallback：路径在产物中存在则原样返回，否则回 index.html；`/api`、`/docs`、`/openapi.json` 不参与回退（404）——否则 API 打错路径会拿到 HTML 200，错误被吞。

### 8.3 热重载（开发）

**`.air.toml`**（仓库根）：监听 `.go` / 模板文件变化，rebuild + restart。**内置预检**：
```toml
[build]
  pre_cmd = ["just lint"]
  cmd = "go build -o ./tmp/<app> ./cmd/<app>"
  args_bin = ["--debug", "server", "start"]
```

**强制规则**：Go 代码改完必须先 `just fmt` + `just lint` 再提交/收工。air 已内置，`just run`（go run 单次）与 `just dev`（concurrently 同起 air + vite）保持一致。

### 8.4 发布产物（gitignore）

```
tmp/                # air 产物、构建中间
runtime/            # 历史运行期数据
web/dist/           # 前端构建产物（go:embed 目标，构建时生成）
web/openapi.json    # OpenAPI 生成产物
web/node_modules/
```

---

## 九、测试策略

### 9.1 测什么

| 类型 | 方法 |
|---|---|
| **纯函数**（解析、计算、字符串、切片运算） | 表驱动测试 |
| **依赖外部进程/库的 IO**（git 读写仓库、文件扫描、缓存读写） | **用 testfixture 建真实临时环境**测，不 mock |
| **文件存储类**（JSON/JSONL 读写、原子写） | testfixture 临时目录真文件 |
| **依赖真实目录扫描** | 用 testfixture 建工程目录树 |
| **HTTP 层**（handler 契约） | httptest 拉起真实 `Server.Handler()` 打真实 HTTP 请求 |

### 9.2 不测什么（靠手动/集成验证）

- 透传 stdio 的命令（`git.Run` 类，无法捕获输出）
- 实际有副作用的启动（开编辑器、开 IDE）
- CLI 命令编排（cobra wiring）

### 9.3 testfixture 模式

提供 `internal/testfixture` 包，统一 IO 测试的 fixture builder：
- 临时目录用 `os.MkdirTemp` 落**系统临时目录**（前缀 `<app>-test-<测试名>-`），测试**成功自动清理、失败保留现场**排查。
- 提供建真实 git 仓库（注入 user 配置、可声明分支/commit/tag/remote/dirty 预期状态）、建工程目录树、建 worktree 的 helper。

**为什么 testfixture 不 import 领域包**：底层包测试要用 testfixture，而 `领域 → 底层` 是依赖链。testfixture 反向依赖领域会循环。所以「构造领域 Service」这种依赖领域包的逻辑写在调用方测试里，不沉淀进 testfixture。

---

## 十、项目自身文档（SDD）

本模板规范的是**项目骨架**。每个项目还要维护自身的行为文档，采用「现状快照 + 提案」两件套（cube 实际实践）：

```
docs/
├── README.md              # 文档索引 + 提案 ID 分配规则
├── spec/现状.md            # 事实快照：描述当前代码「是什么」（定位/架构/命令/API/数据/配置/测试），
│                          #   每次合并有意义变更后同步更新；与代码冲突时以代码为准
├── proposals/             # 未来需求提案：每个提案一个目录（1xxx-主题/，ID 四位递增不回收）
│   ├── <活动提案>/README.md
│   ├── parked/            # 方向认可但触发条件未到（README 记录解挂条件）
│   └── archived/          # 已完成归档（记录最终落地形态与方案演变）
└── references/            # 竞品分析等参考文献
```

- **发号硬约束**：新建提案前先取当前最大 ID（cube 用 `make last-proposal`），新 ID = 最大 + 1，不重复不猜测。
- **维护节奏**：合并一个有意义变更后同步更新 现状.md；提案实施完毕把目录挪进 archived/（只挪目录不改号）。
- `docs/` 记「为什么这么做」，现状.md 记「当前是什么」。

---

## 十一、可选扩展点

模板的基线是「CLI + HTTP API + embed 前端」。以下能力按需挂载，都是**加法**：

| 扩展 | 形态 | 架构落点 |
|---|---|---|
| **长连接会话**（PTY 终端、文件 watch 推送） | WebSocket | handler 注册走 Routes 的 Raw 口（不经 huma、不进 OpenAPI）；子进程生命周期挂 app 的 OnServerStop 钩子回收。**引入长连接是架构复杂度的临界点，单独权衡。** |
| **TUI 交互**（多选、表单、表格） | huh + lipgloss | `pkg/tui` 独立工具包，CLI 出口的交互组件统一收口。 |
| **Alfred / Raycast / Script Filter 集成** | JSON 输出 | cmd 下独立子命令组；配 usage 记录做 frecency 排序。 |
| **server 进程管理** | HTTP 应用层（status/shutdown 端点 + HMAC 鉴权） | system handler + server/client 子包；探活验 `app` 字段防端口误占，stop 按 instance 标识判旧实例下线。不写 pid 文件、不 fork 后台。 |
| **多 remote 批量操作** | 批量 API | 领域层加批量 service；注意并发/失败策略（全失败回滚 vs best-effort）。 |
| **worktree 管理** | 领域子能力 | 物理位置约定 + 是否纳入扫描的约定需先决策。 |

> 挂载扩展点时复用「五处加法」流程，新增不污染既有 domain。

---

## 十二、起步流程（新项目如何用模板）

1. **立项**：建 `docs/`（README.md + spec/现状.md），明确做什么、不做什么。
2. **搭骨架**：照 §4 目录结构建空包；`go mod init <module>`；抄 justfile / Makefile / .air.toml / .gitignore / web 工程脚手架。
3. **后端起步**：
   - 写 `internal/config`（配置加载 + Paths）、`internal/logger`（双通道 handler）、`internal/version`（ldflags 变量 + AppName）。
   - 写 `internal/app`（App 空壳 + Init 组装 DI）与 `internal/cmd/root.go`（PersistentPreRunE 惰性 Init）。
   - 写第一个 domain（实体 + service）+ `internal/handlers/<domain>_handler.go`（用 Envelope 信封 + `r.Get` / `r.Post`）。
   - 跑 `<app> server start`，访问 `/docs` 确认 OpenAPI 生成正常。
4. **前端起步**：
   - `web/` 下用 Vite 建 React + TS 工程；装 shadcn（Base UI 底座）+ Tailwind + TanStack Query。
   - 配 `vite.config.ts`：`build.outDir = 'dist'`，dev server proxy `/api` 到后端。
   - 装 openapi-typescript + openapi-fetch，配 `gen:api` 脚本。
   - `<app> openapi -o openapi.json` → `pnpm gen:api` → 写第一个 page 消费 API。
5. **embed 打通**：`web/embed.go` 写 `//go:embed all:dist` → `just build` → 验证单二进制能 serve 前端。
6. **迭代**：按 现状.md 与提案推进，每个 domain 走五处加法（有 devtool 则 `just gen domain`）。改后端 handler 后重跑 openapi + gen:api 保持契约同步。

---

## 附录：决策记录（随迭代补充）

| 日期 | 决策 | 上下文 |
|---|---|---|
| 2026-08-04 | 模板首版，从 cube 提炼。技术栈：huma v2 + cobra + slog；前端 Vite + React + TS + shadcn。 | cube 规划多窗口工作台（workspace），决定引入前端构建链，顺势把项目范式抽象成通用模板。 |
| 2026-10-31 | 模板同步到 cube 重构后形态：justfile 工具链 / cmd+internal+pkg 目录布局 / DI 显式装配 / gorm·fsnotify·go-git 移除（文件存储 + 系统 git 单轨 + 无热 reload）/ web 直接 embed dist / 现状+提案 SDD。 | cube 按模板源重构代码文件结构，上游化 x/store、x/di，模板随之对齐。 |
