# 1047 内部类型出参隔离

## 状态

挂起（2026-11-08）。全量接口盘点已完成、修复模式已验证，待触发条件再批量实施。

**解挂条件**（满足其一）：

- 需要动到对应 domain 类型结构时（字段增删/语义变更），顺手把该域出参 DTO 化，避免内部演化自动冲击 API；
- API 出现前端之外的第二消费者（脚本 / 第三方编排）时——内部类型直通的前提是「前端是唯一消费者、wire 格式即契约」，该前提失效前必须先完成隔离；
- `forge.Account`（token 字段上 API 面）或 `config.Config`（dataDir 暴露）所在域有任何改动时，优先带上。

## 背景

Web API 的出参应只含**基础类型（含 slice/map）+ handlers 包内定义的类型**。实际盘点（2026-11-08，42 个端点全量扫描）发现 17 个端点的响应类型整体是内部类型直通：domain 实体、pkg 类型、甚至基础设施层类型直接进 OpenAPI schema 和前端。

首例已修复验证：project list 的 `gitInfo` 原直通 `projcache.Entry`，已隔离为 handlers 内的 `ProjectInfoDTO` + 嵌套 `RemoteDTO`/`WorktreeDTO`（commit `4454781`）——json tag 与原 wire 格式逐字段一致，前端无感；projcache 此后可自由演化字段（如删 Branches/Detached）不再冲击 API。本提案把该模式铺到其余端点。

## 问题清单（盘点快照）

### A 级：响应类型整体直通内部类型（17 端点）

| 域 | 端点 | 暴露类型 | 备注 |
|---|---|---|---|
| forge | GET `/api/forge/list` | `ListResult[forge.Forge]` | 含 `*icons.Icon` 字段 |
| forge | GET `/api/forge/account/list` | `ListResult[forge.Account]` | **含 Token 字段**，靠 handler 手动打码（forge_handler.go accountList）——类型带 token 上 API 面，未来新端点忘打码即泄漏 |
| forge | GET `/api/forge/overview` | `*forge.Overview` | 三层嵌套：`RepoRow` → `gitapi.RemoteRepo` + `*LocalRepo`（后者又嵌 projcache 字段）、`AcctMeta` |
| workbench | info / refs / remotes / commits / commit / worktrees / tree / file / diff / worktree/add（10 端点） | `workbench.Info/Refs/RemoteEntry/CommitsPageResult/WorktreeStatus/TreeListResult/FileResult/DiffTreesResult/WorktreeCreated` | 且嵌 pkg 与兄弟 domain 类型：commits → `[]git.CommitEntry`、commit → `*git.CommitDetail`、worktrees → `[]workspace.Workspace` |
| project | GET `/api/project/scan-rules` | `ListResult[project.ScanRule]` | 含 `*icons.Icon`；**入参侧已有 Item 转换，出参侧漏了** |
| project | GET `/api/project/clone-rules` | `ListResult[project.CloneRule]` | 同上 |
| usage | GET `/api/usage/recent-paths` | `ListResult[usage.PathUsage]` | 形状极简 `{path,time}`，但同为 domain 直通 |
| template | GET `/api/template/source/list` | `[]template.TplSource` | 同 scan-rules：入参有 Item 转换、出参直通 |
| config | GET `/api/config` | `config.Config` | **基础设施层类型直通**，含 `dataDir` 本地路径 |

### B 级：handlers 内定义的类型，字段引用内部类型

`icons.Icon` 作为字段出现在 4 个 handlers 类型：`ScanRuleItem` / `ForgeItem` / `OpenerItem`（入参）+ `OpenerDTO`（出参）。

### 已合规（无需动）

- project：list（`ProjectDTO` + `ProjectInfoDTO`）、open、workspace get/save、全部写接口入参
- opener：list / intents 出参、全部写接口入参
- forge：全部写接口入参（Item + toXxx 模式）
- md / icon / system：纯基础类型 + handlers 类型

## 待决策

**D1：`icons.Icon` 是否豁免为基础值类型。** 它是跨域共享的值类型（lucide 图名或 base64 PNG 的包装），4 个域共用、行为接近 `time.Time`。若豁免，B 级只剩「出参补 DTO」；若不豁免，需在 handlers 定义 `IconDTO` 并在 4 个域的 Item/DTO 上统一替换。倾向豁免（它无行为演化风险，序列化形态即契约本身），解挂时定稿。

**D2：workbench 域类型的归属。** workbench 的 Info/Refs/TreeListResult 等本就是「为出口设计的形状」（包注释明示），迁移是纯机械搬移 + 转换层。备选：按本提案在 handlers 建 DTO；或反向把这些纯展示类型上移定义在 handlers、domain 只留业务实体。倾向前者（不动 domain 分层），量级约 10 个类型 + 10 个端点转换。

## 实施模式（已由 4454781 验证）

每个域照抄 ProjectInfoDTO 模式：

1. handlers 定义 `XxxDTO` + 嵌套 `YyyDTO`，字段取**前端实际消费的子集**（借机省略未消费字段），json tag 与现 wire 格式逐字段一致——前端运行时零感知；
2. 转换收敛在单个 `newXxxDTO` 函数，内部类型引用只剩这一个签名；
3. `just gen-api` 再生成 schema.d.ts，`pnpm test` + `pnpm build` 验证；
4. wire 不变，无版本兼容负担。

建议分批（每批独立可验证、独立提交）：

1. **forge.Account / config.Config**——有真实暴露语义的先行；
2. **scan-rules / clone-rules / tpl-source-list / recent-paths 出参**——模式一致，半天量级；
3. **workbench 全家**——端点最多，批量机械迁移；
4. forge list / overview——依赖 D1 决策后与 ① 同模式收尾。
