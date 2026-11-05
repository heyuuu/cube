# AGENTS.md

面向未来 ZCode agent 的项目工作规则。先读此文件，再动手改 cube。

> 项目采用 SDD（Spec-Driven Development）管理演进，**当前现状见 [`docs/spec/现状.md`](./docs/spec/现状.md)**（定位/架构/命令/API/数据/配置）。改动功能或架构前，先读 现状.md 对应段落。本文件与 现状.md 冲突时，**现状.md 是事实基准**（它描述代码「是什么」），本文件侧重「怎么改」。未来需求提案在 [`docs/proposals/`](./docs/proposals/)。

## 项目简介

**cube** —— 面向个人开发者的本地多项目管理工具（Web UI 优先，CLI 简单可行兜底）。Go 1.27 编写，module path `cube`（go.mod 第一行）。

- 历史有三代：v1 (php)、v2 (go)、**v3 (当前，按领域重构)**。
- 出口：本地 Web HTTP server（`cube server`，huma + 标准 ServeMux，主要交互形态）、CLI（人用 / alfred，web 不可用时兜底可行）。
- 定位原则：不做云服务、不绑 AI（cube 可被 AI 编排，但自身不集成 AI）。
- Go 源码与 go.mod 都在仓库根（`cmd/` / `internal/` / `pkg/` 分目录）；构建 / 安装 / 校验工具链收敛在仓库根 `justfile`（`just --list` 看全量；Makefile 只保留 `make tag` / `make last-proposal`）。

## 分层架构（改代码必须遵守的依赖纪律）

```
基础设施  internal/config / internal/logger / internal/version                 所有层共享
独立工具  pkg(git / gitapi / fuzzy / easycache / pathkit / slicekit / oskit / tui)  通用动作, 不含业务实体, 只依赖标准库/第三方（文件存储原语在外部包 github.com/heyuuu/x/store）
能力      internal/server                                        服务端框架, 不含业务实体
领域      internal/domain: project (含 projcache / scan / clone / cubefile / workspace) / usage / workbench / forge / opener / template / icons   业务 domain, 含实体和规则
出口      internal/cmd / internal/handlers / web                 把领域包成 CLI/Web（handlers=业务 HTTP handler 层；web 包只剩前端构建产物 go:embed 容器 embed.go）
装配      internal/app / cmd/cube/main                           接线（cmd/devtool 是脚手架工具的独立入口）
```

- **pkg 是独立工具层**：只依赖标准库/第三方，不含业务实体；基础设施层与能力层都可以依赖 pkg（`internal/logger → pkg/tui` 合规）。基础设施不依赖上层；能力层不依赖领域/出口；领域层依赖能力+基础设施+pkg；`handlers` 依赖领域层与 `internal/server` 框架；**cmd 与 web 不互调**（cmd 经 `internal/server/client` 走 localhost HTTP 探活/关停 server，是进程间通信，不算互调）；`internal/app` 是唯一接线点。
- **App 装配是显式构造，CLI 侧惰性触发**：`app.New()` 只建空壳（`internal/app/app.go`），root 的 `PersistentPreRunE` 调 `a.Init(cfgFile, debug, local)` 完成组装（config.Load → logger.Init → `di.RegisterAll` 建 DI 容器）；业务单例注册清单在 `internal/app/boot.go`（serviceFactories / handlerFactories，带 generate 注入点标记）。`cmd.Execute()`（`internal/cmd/root.go`）在 `cmd/cube/main.go` 里调用；命令工厂收 `*app.App`。**无全局单例、无 `app.Default()`、无包级 `init()` 反向依赖**（「惰性」仅指 cmd 层触发时机，装配本身仍是显式一次性构造）。
- 加一个新 domain = ①`internal/domain/<x>` 领域包 ②`internal/cmd/<x>.go` 子命令（+ root.go 注册） ③`internal/handlers/<x>_handler.go`（实现 `Handler.Register`，注册走 `Routes.Get` / `Routes.Post` + `server.JSONHandler`） ④config 加节（如需） ⑤`internal/app/boot.go` 注册清单加构造。**五处都是加法，不碰现有 domain**；脚手架已工具化——`just gen domain <name>` 自动完成（生成器在 `cmd/devtool/internal/generate`，以 boot.go 注入点标记为锚点追加）。

## 关键机制（改动前先理解）

- **项目前提：所有项目都是 git 项目**。`.git` 存在是扫描判定项目的必要条件（详见 `internal/domain/project/scan.go`）。因此 `tags` 不打冗余的 `git` 标签，只标额外特征（`worktree` / `godot`）。改扫描/tag 逻辑时遵守此假设。
- **持久配置不按项目路径记录内容**：project 一切由 scan-rule 推导。**禁止的是在 settings.json / config 等「持久配置」里按项目路径记录具体 project 的内容**——项目目录会重命名/移动，按路径 keyed 的持久配置会失联留脏数据；**短生命周期、自动刷新、可丢弃的 cache（如 git.json）不在此列**。需要「project 自身的配置」（如 monorepo workspace 声明，方向定为项目根 `.cube/cube.json`）时，放项目内跟仓库走，不放全局配置。
- **projcache 异步采集**（原 projcache，1030 起含 workspace 更名）：`project list --status` 等读命令从 `~/.config/cube/cache/git.json` 读项目状态快照（git 信息 + worktree 枚举 + workspace 成员，几乎零开销）；**单写者模型**——常驻 server 是 git.json 的唯一写方（后台异步采集回写），CLI 只读不写，落盘靠原子写（tmp + rename），无跨进程锁。**读路径不得阻塞采集——只能读快照**（workspace 也一样：解析/探测只发生在采集侧，读路径不读 `.cube/cube.json`）。详见 [`docs/spec/现状.md`](./docs/spec/现状.md)「三、关键机制」。
- **monorepo workspace（1030）**：项目根 `.cube/cube.json` 声明打开子目录（`workspaces` 显式声明优先，空数组不回落探测；字段缺失按 `workspaceScanRule`/默认 `pnpm|npm` 探测 pnpm/npm 标准声明文件为正选——规则格式 `|` 分规则组按序回落，组内 `+` 合并去重）。文件格式层在 `internal/domain/project/cubefile`（多节容器：读写 + 节存在性语义 + 未知节保留），workspace 解析与探测在 `internal/domain/project/workspace` 包（纯函数），组合进 projcache 采集。此为「project 自身的配置放项目内跟仓库走」的落地形态，不放全局配置。
- **OpenTargets 是 CLI 专用出口**（`cube open` / alfred / `project/open` 归属校验）：Web 前端不消费它（无 HTTP 端点），前端下拉从 list DTO 快照字段自拼同一套排序——改打开目标的排序/展示规则时，后端 `targetEntries` 与前端 `projectTargets` 两边同步。
- **opener：接口 + 唯一 exec 实现 + settings.json**：`Opener` 是接口（`internal/domain/opener/opener.go`），唯一实现 `actionOpener`（`exec.go`）：按 role 各配一条动作串 `actions: {role: "<kind>:<模板>"}`，kind ∈ {`exec:`（命令，sh 风格分词 + `$0/$1` 占位）、`url:`（`/` 开头站内路由拼 baseURL + query encode；`http(s)://` 外部链接）}，两种动作统一经 `Executor` 执行（`executor.go`，测试注入 fake）；role 是纯参数约束（槽数，见 `role.go`），声明 roles = actions 键集合；`Open(role, slotArgs...)` 的 role 校验收敛在实现内。baseURL（站内路由用，来自 config `server.port`）由装配注入，opener 包不 import config。**opener 数据存 settings.json 的 openers 节**（节级 API 在外部包 `github.com/heyuuu/x/store`：`store.LoadSection[T]` / `store.SaveSection`；Service 直读不缓存、写侧领域校验；详见 现状.md 3.3）——改 opener 时同步看 `internal/domain/opener/` 下 `opener.go`、`exec.go`、`role.go`、`executor.go`、`intent.go`。
- **CLI 惰性装配**：全局 flag（`--config` / `--debug` / `--local`）走 cobra PersistentFlags 真解析（`StringVar` 闭包注入，`internal/cmd/root.go`）；装配链（config→logger→app DI 容器）由 root 的 `PersistentPreRunE` 直调 `a.Init(...)` 完成——不进 RunE 的路径（`--help`/未知命令）全程零装配。命令工厂统一收 `*app.App`，RunE 内经 `a.XxxService()` / `a.Container()` 取依赖。钩子顶部 `cmd.SilenceUsage = true` 画出 usage 边界：输入类错误（flag/参数）附 usage，运行期错误（Init/RunE）不附。错误输出双通道：cobra `Error:` 行给终端、`slog` 给日志。注意 cobra 只执行最近一层 PersistentPreRunE——子命令自定义钩子会顶掉 root 的装配钩子。
- **Web 出口分两层**：服务端框架在 `internal/server` 包（Server 装配 / 路由 / envelope，`server.NewServer(cfg.Server, handlers []Handler)` 显式收 handler 列表——system 与 static handler 也在装配清单里显式注册、不自动追加；`internal/server/client` 提供 server 进程探活/关停）；业务 handler 在 `internal/handlers` 包（`<domain>_handler.go` 同包分文件，不按 domain 分子包），实现 `Handler.Register(r *server.Routes)`——注册统一走 `r.Get` / `r.Post` + `server.JSONHandler`（自动包 envelope），WebSocket 等原生路由走 `r.Raw`（不经 huma）。统一 `Envelope{ok,message,data}` envelope（泛型 `Envelope[T]`，见 `internal/server/routes.go`）；路径强制 `/api/` 前缀，由 `parseInfoFromPath` 解析 group tag + operationId。响应 JSON 经 `encoding/json/v2`（`humaJSONFormat`，`internal/server/routes.go`）把 nil 切片序列化为 `[]`——新增 handler 自动复用，不要在 handler 里手写 `make([]T, 0)` 兜底。`web` 包只剩前端构建产物的 go:embed 容器（`web/embed.go` 的 `FS()`，直接嵌入 `web/dist`，无拷贝步骤）。
- **配置与双环境**：默认目录按环境分流——dev（源码直跑 / air / `just run`）→ `~/.config/cube-dev/`，prod（`just build` / `just install`，ldflags 注入了正式 version）→ `~/.config/cube/`；身份判定见 `version.IsDev()`，详见 [`docs/proposals/archived/1026-环境分离/`](./docs/proposals/archived/1026-环境分离/)。`config.json` 按 domain 分节，`server.port` 是端口唯一事实源（无 `-p` flag、不支持多实例）。`--config` 覆盖配置文件路径，`--debug` 开 debug（只影响 logger 初始化）。配置文件缺失降级为默认值，JSON 解析失败报错退出。settings.json 各节读失败降级为空、坏数据不阻断（降级优先）。**无热 reload**（配置改动重启生效；settings.json 用户数据走各域 save API 即时生效）。配置目录下的运行期状态（`state/usage.jsonl`、`cache/git.json`、`log/app.log`）由 `config.Config` 的路径方法（`internal/config/paths.go`）统一计算，不要在调用方硬拼路径。应用标识（`AppName`）收敛在 `internal/version`：status 身份 / 进程探测 / shutdown token / 默认配置目录路径均由 `version.AppName` 派生；日志文件名是通用的 `app.log`，不含应用名。

## 常用命令

构建 / 安装 / 校验统一走 justfile（仓库根 `justfile`，`just --list` 看全量；Makefile 只保留 `make tag` / `make last-proposal`）：

```bash
just build        # 先 build-ui（pnpm --dir web build，产物 web/dist 由 web/embed.go 直接 go:embed），再 go build + ldflags 注入 version（产物 tmp/main，可用 OUTPUT 覆盖）并跑 version 自验
just install      # build-ui + go install + 安装校验 + 停旧服 + 写 zsh completion（含 scripts/zsh-append.sh 的 p/pz 函数）
just fmt          # 格式化（golangci-lint fmt）
just lint         # 校验（golangci-lint run）
just test         # go test -race ./...
just run [args]   # go run ./cmd/cube …（手动跑单次命令）
```

Web 开发热重载（`.air.toml` 在仓库根，pre_cmd 已内置 `just lint`）：

```bash
just dev           # 需安装 air；concurrently 同时起 air（--debug server start，端口读 dev config 的 6001）与 vite
```

脚手架（生成器在 `cmd/devtool`，模板在其 `internal/generate/tpl/`）：

```bash
just gen domain <name>                        # 生成业务 domain 脚手架（五处加法自动注入）
just gen api <GET|POST> <domain> <base-name>  # 在已有 domain 的 handler 追加 API 端点
just gen-api                                   # 前端类型再生成（cube openapi → openapi-typescript → src/api/schema.d.ts）
```

测试（标准 go test，在仓库根执行；无额外 harness）：

```bash
go test ./...
go test ./internal/domain/opener/...     # 聚焦某个包
```

### 测试辅助包 `internal/testfixture`

cube 的 IO 测试（git 操作、文件扫描、缓存读写）通过 `internal/testfixture` 包提供统一 fixture builder：

- **临时目录用 `os.MkdirTemp` 落系统临时目录**（前缀 `cube-test-<测试名>-`），测试成功自动清理、失败保留现场排查（旧方案写项目内 `runtime/test/` 且永不清理，会持续膨胀并被 goimports/IDE 扫描拖慢）。
- **`Workspace`**：测试工作区。`ws := testfixture.NewWorkspace(t)` → `ws.Dir` 是该测试专属目录；`ws.Mkdir/WriteFile/Join` 在其下操作。
- **`BuildGitRepo(t, dir, GitRepoSpec)` / `ws.MakeGitRepo(name)`**：建真实 git 仓库（用系统 git + 注入 user 配置，不依赖全局 git config）。`GitRepoSpec` 声明预期状态（分支/commit 数/tag/remote/ahead/dirty）。
- **`ws.MakeProjectDir(relPath, opts...)`**：建「会被 cube 扫描识别为 project」的目录（默认含 git 仓库）。opts：`WithGodot()`/`WithWorktree()`/`WithDirty()`/`WithoutGit()`。

```go
ws := testfixture.NewWorkspace(t)
repo := ws.MakeGitRepoWith("repo", testfixture.GitRepoSpec{
    Branch: "develop", RemoteUrl: "/tmp/remote.git", MakeDirty: true,
})
// 或工程目录
ws.MakeProjectDir("scanroot/g1/proj", testfixture.WithGodot())
```

**为什么 testfixture 不 import `project` 包**：底层包（`git`/`projcache`）的测试要用 testfixture，而 `project → projcache → git` 是依赖链。若 testfixture 反向依赖 project 会形成循环。所以「构造 project.Service」这种依赖 `project` 包的逻辑写在调用方测试里（见 `project/scan_test.go` 的 `newServiceAt`），不沉淀进 testfixture。

### 测试策略（什么测、什么不测）

- **纯函数**（解析、计算、字符串处理）：普通表驱动测试。`fuzzy`/`pathkit`/`git/url`/`git 读输出解析`/`slicekit`/`easycache`/`opener 解析`。
- **依赖外部进程/库的 IO**（git 二进制读/写仓库）：**用 testfixture 建真实临时仓库测**，不 mock。`git` 的 `Refs/Remotes/IsDirty/LoadRepoStatus`、`git.FindGitRoot`、`projcache.Load/Save/Refresh/collectEntry`。
- **依赖真实目录扫描**：用 testfixture 建工程目录树，直接构造 `[]ScanRule` 喂给 `project.NewService`（绕开 config/app 装配）。`internal/domain/project/scan_test.go`。
- **opener 执行类**：通过 `Executor` 接口注入 fake，不真的启动编辑器。见 `internal/domain/opener/opener_test.go`。
- **web 层**：httptest 拉起真实 `Server.Handler()` 打真实 HTTP 请求，断言路由 / DTO / envelope / nil 序列化 / 静态资源契约。框架行为（静态资源 SPA fallback / system 端点 / shutdown token）与业务契约的用例都在 `internal/handlers` 包（`env_test.go` 的 newTestEnv 基建）。新增 handler 时在此模式上补用例。
- **不写单测的（靠手动/集成验证）**：
  - `git.Run`/`git.Clone`/`git.Push`（透传 stdio 到 `os.Stdout`，无法捕获输出；且本质是组装 git 参数）
  - `opener.Open` 的真实进程启动（已用 Executor 隔离，但默认实现的真启动仍靠手动验证）
  - `cmd/*`（cobra 命令编排）

## 必须遵守的编码规则

1. **Go 代码修改后必须先格式化与校验再提交/收工**——这是强制规则：
   ```bash
   just fmt    # 格式化（golangci-lint fmt）
   just lint   # 校验（golangci-lint run）
   ```
   每次改完 `.go` 文件都要跑，不要跳过。`air` 的 pre_cmd 已内置 `just lint`，保持一致。**命令在仓库根执行**（go.mod 在仓库根）。
1b. **前端代码修改后必须先 `pnpm fix` 再提交/收工**——与 Go 侧规则 1 同级的强制规则：
   ```bash
   pnpm fix    # oxfmt 格式化 + oxlint 自动修复并报出剩余问题（error 不会自动消失，须手工修）+ tsgo 类型检查
   ```
   每次改完 `web/src` 下的文件都要跑，**命令在 `web/` 目录下执行**。fix 报出的 error（如 `react/react-compiler`）不会被自动修复，必须当场手工修掉再收工——曾经因为跳过此步，effect 回灌 state 类违规积压数个提交才集中暴露。
2. 遵循 v3 分层依赖纪律（见上），不要让 `cmd` 直接调 `web`、不要让基础设施包 import 领域包。
3. **加新 domain 走"五处加法"流程**，不修改既有 domain 的接线。
4. 日志统一用 `log/slog`（`slog.Debug` / `slog.Info` / ...），不要用 `fmt.Println` 做日志（`fmt` 仅用于面向用户的 CLI 输出）。debug 日志受 `--debug` 控制。
5. 错误处理遵循现有风格：可恢复的降级用 `slog` 记录后继续；致命错误用 `fmt.Errorf("...: %w", err)` 包装并返回。错误输出双通道：cobra 的 `Error:` 行给终端（`cmd.Execute()` 的返回值由 main 处理），`slog` 给日志。
6. **所有 Error 消息一律用中文**（`errors.New` / `fmt.Errorf` 的字符串）。变量名、标识符、以及约定俗成的英文专业名词/技术术语保留英文原词——例如 `repoUrl`、`OpenAPI`、`worktree`、`opener`、`config`、`slot`、`tty` 等。参考既有代码，如 `fmt.Errorf("repoUrl 不是合法地址: url=%s", rawRepoUrl)`、`errors.New("未找到指定app: " + appName)`。
7. **构造函数命名约定**（按返回值形态选前缀）：
   - `NewXxx()` → 返回 `*Xxx`（指针，单返回值）。例：`NewService` / `NewOpenerHandler` / `NewItem`。
   - `MakeXxx()` → 返回 `Xxx`（值类型，单返回值）。
   - `InitXxx()` → 用于构造时需要返回 `error` 等额外值的情况（即 `(*Xxx, error)` 或 `(Xxx, error)`）。
   - 解析/加载类函数不在本约定范围内，保留 `ParseXxx` / `LoadXxx` 等既有命名（如 `ParseRepoUrl`、`projcache.Load`）。
8. 写表用 `tui.PrintTable`，交互选择用 `tui.SelectItem`，保持 CLI 输出风格一致。
9. **优先复用 `pkg/` 下的辅助函数**，能力收敛在各工具子包内，不在调用方就地重造：
    - 动手前先 grep 对应 pkg 子包；缺什么就**在该子包里加新函数**，而不是在 `cmd/` / domain 里实现。例：git 读写统一走 `pkg/git`（读按主题分文件：`refs.go` 分支/tag/ahead-behind、`remote.go` 远端配置、`status.go` 工作区状态、`tree.go` / `diff.go` / `log.go` / `worktree.go` 各主题查询、共享执行核 `run.go`；写：clone / push / commit / fetch 等，见 `git.go`）；路径处理走 `pkg/pathkit`；切片运算走 `pkg/slicekit`。
    - pkg 子包内的函数必须**足够内聚且无副作用**：只依赖入参做纯运算，不读进程状态、不读环境。与环境强相关的副作用（`os.Getwd()` / `os.Getenv()` / 读 `~` / 当前时间等）只允许出现在**职责就是处理环境的包**（如 `pkg/pathkit` 展开 `~`、`internal/config` 读配置目录；`pkg/git` 的 `runOut` 为子进程显式构造环境属于其执行职责，不算读环境）；其它子包（`pkg/git` 的纯解析函数 / `pkg/slicekit` 等）一律不得调用这类函数。参数处理、cwd 解析、交互编排属于 `cmd` 层职责，不沉淀进 pkg 子包。
    - 警惕功能重叠：例如「向上探测 `.git` 根」已有 `git.FindGitRoot(dir)`，调用方就不该再写一遍 `os.Stat(filepath.Join(..., ".git"))` 的循环。
10. **注释只写「为什么」和「目的」，不要复述「执行过程」**。函数体内的步骤标号（`// 1. 先读 pid 文件 // 2. 再发信号`）、逐行翻译式注释（`// 遍历列表`、`// 返回结果`）属于过程复述——代码本身已经表达了执行过程，注释再写一遍只会制造**两个需要同步维护的事实源**，代码改了忘改注释就会两边对不上。应保留的是代码读不出来的信息：设计意图（如「端口冲突要报错，否则造孤儿」）、非显然的取舍（如「用 SIGKILL 兜底而不是无限等」）、外部约束（如「子进程 stdio 接 /dev/null，日志走 slog」）。判断标准：如果删掉这条注释，读者看代码能否理解「在做什么」——能，就删；读者看代码无法理解「为什么这么做」，就留。
11. **前端 shadcn 基于 Base UI，不可使用 Radix UI 写法**。前端源码在 `web/`（仓库根），shadcn style 为 `base-mira`，UI 原语统一从 `@base-ui/react/<模块>` 子路径导入并按命名空间使用部件（如 `@base-ui/react/checkbox` 的 `CheckboxPrimitive.Root` / `.Indicator`）。AI 训练语料中的 shadcn 示例绝大多数是 Radix 版本，写/改 `web/src` 时**勿照搬 Radix 写法**：不引入 `@radix-ui/*` 依赖；组件多态渲染用 `render` prop 而非 `asChild`；状态样式用 `data-open` / `data-checked` 等具体布尔属性而非 `data-state="..."`。不确定 API 时以 `web/src/components/ui/` 现有组件为准，参考 [Base UI 文档](https://base-ui.com)。

12. **Go struct 与其方法必须放在同一文件中**。同一 struct 的定义（`type Xxx struct { ... }`）和该 struct 的所有方法（`func (x *Xxx) Method()`）必须写在同一个 `.go` 文件里，不要分散到多个文件——方便审阅时一次看完一个类型的全部行为。
   - 允许拆分的是：与 struct 无关的纯函数/纯辅助工具（如解析函数、常量、独立类型定义），它们可以按职责分文件存放。
   - 示例：`workbench/service.go` 包含 `Service` struct 定义和全部 17 个方法；`workbench/diff.go` 仅保留 `DiffEntry` 类型、`DiffTreesResult` 类型和纯辅助函数。

13. **git 子进程调用只允许出现在 `pkg/git` 包内**。包外（domain / cmd / web / app）不得 `exec.Command("git", ...)`、不得拼 git 子命令参数，一律调用 `pkg/git` 的**类型化函数**；缺什么就在包内新增类型化封装（含输出解析），**不对外暴露「调用方传 args 的通用执行口」**（历史上曾有 `RunRead`，已移除）。
   - 原因：git 的参数拼装与输出解析天生平台/环境强相关（`core.quotePath` 对非 ASCII 路径的八进制转义、locale 随配置翻译 stderr、pager 挂起、`--no-optional-locks` 防抢 `index.lock`、`diff --no-index` 有差异时退出码为 1、`ls-files --directory` 对混合目录的折叠语义等），必须收敛在单点做兼容——换环境（如部署到 Linux 服务器）出问题时统一排查、统一修，而不是去改散落在各处的参数。这些兼容注入见 `runOut`（`pkg/git/run.go`）。
   - 返回值解析也尽量放包内（返回类型化的 struct / 切片 / 集合，如 `[]DiffFile` / `*Ignored`），调用方只做业务语义映射；纯展示层 DTO 转换（如状态字母 → 前端语义词）留在出口层。
   - 例外：测试代码（`internal/testfixture` 与各 `*_test.go` 建仓/对账）可直接 exec git——fixture 若反向调用被测包会循环依赖。

14. **Web API 只用 GET / POST 两种 method**。GET = 查询（参数走 query），POST = 动作（参数走 body，平铺挂在 huma input 的 `Body` 子结构上，惯例同 `opener/open`）。不引入 PUT / DELETE / PATCH——保存、删除等语义放进 **API 名**（path / operationId，如 `workbench/file/save`、`xxx/delete`），不用 method 区分。前后端都不提供其他 method 的 helper（后端注册口仅 `Routes.Get` / `Routes.Post`，前端 `client.ts` 仅 `apiGet` / `apiPost`；曾有的 `apiPut` / `apiDelete` / `apiRegisterOp` 已移除）。
15. **Service 包的代码布局**（workbench / project / opener / history 等 service 形态的包）：
    - **入口统一在 service.go**：所有依赖 Service 的对外能力都是 `Service` 的公开方法，全部集中在 service.go（与规则 12 呼应）。方法体保持薄——参数校验与分支编排之外的具体逻辑一律外调；10 行以内、不值得单开主题的简单逻辑可直接写在方法里。
    - **复杂逻辑内聚到主题文件**：需要独立输出类型、含复杂算法、或多个方法共享辅助函数的主题（如 diff.go / pty.go）单独成文件，以**包级函数**暴露功能（可含该主题私有的类型与常量），service.go 只留一行委托。判断标准：这组逻辑是否值得脱离 Service 单独阅读与测试。
    - **types.go 放跨主题复用的类型**：struct / 常量，及其构建、解析函数（newXxx / parseXxx 等）；仅单主题使用的类型跟随主题文件。
    - **helpers.go 放跨主题复用的内部函数**：与具体主题无关的通用小工具。


## 文档

改动敏感区域前先读：

- [`docs/spec/现状.md`](./docs/spec/现状.md) —— 项目现状（定位/架构/命令/API/数据/配置）。改架构边界或加 domain 前必读。**与代码冲突时以代码为准**。
- [`docs/proposals/`](./docs/proposals/) —— 待办需求提案（按 `1xxx-主题/` 目录组织，每个提案含 README.md，部分含 alternatives.md / design/）。**新建提案必须先 `make last-proposal` 取最大 ID，新 ID = 最大 ID + 1，不得重复或凭猜测取号**（详见 docs/README.md 的「ID 分配规则」）。
- [`docs/references/`](./docs/references/) —— 竞品分析（mani / gitbatch，做批量操作前必读 gitbatch 避坑点）。
- `README.md` —— 项目简介与 v3 现状总览。

## 已知 gotcha

- **go.mod 在仓库根**（go 源码与 justfile / .air.toml 同在仓库根，`cmd/` `internal/` `pkg/` 分目录）。`go build` / `go test` / `just` 各命令都在仓库根跑。
- **module path 是 `cube`**（不是 `github.com/heyuuu/cube`——历史 README 里写的旧值，以 go.mod 为准）。import 路径写 `cube/...`。
- **`web/dist` 不入库但 go:embed 需要它存在**（`web/embed.go` 直接嵌入 `web/dist`，无拷贝步骤）：clone 后首次 `go build` 前必须先 `just build-ui`（或手工 `pnpm --dir web build`），否则 embed 报错。
- `.gitignore` 忽略：仓库根的 `tmp/`、`runtime/`（测试产物）、`.zcode/plans` / `.claude/plans` / `.cursor/plans`，以及 `web/.gitignore` 里的 `web/dist`、`web/openapi.json`、`web/node_modules`。不要提交这些。
- 默认配置目录按环境分流：dev `~/.config/cube-dev/`、prod `~/.config/cube/`（非项目目录），运行期状态（`state/usage.jsonl`、`cache/git.json`、`log/app.log`）落在对应配置目录。两环境数据不互通（usage 各自积累、config 人工 diff 合并）。
- **开发中修改本地配置时只允许动 `~/.config/cube-dev/` 下的内容，禁止修改正式配置 `~/.config/cube/` 下的任何文件**——正式配置由已安装的 prod 二进制使用，格式随旧版本；开发期格式变更只迁移 dev 一份，正式配置待安装新版本时人工迁移。
- **前端源码在 `web/`（仓库根）**，构建产物 `web/dist` 由 `web/embed.go` 直接嵌入。改前端改 `web/`，不要手改 `web/dist/`（会被覆盖）。
