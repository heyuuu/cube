# settings 清单接口整项化（delete 传整项 / reorder 改 save-all / 写后全量重拉）

> **状态**：✅ 已交付（2026-11-07）。实施偏差与顺带修复见文末「实施记录」。

## 背景

settings.json 的六个清单节（openers / forges / forgeAccounts / scanRules / cloneRules / tplSources）各有 `save` / `delete` / `reorder` 三个写接口。其中 `delete` 与 `reorder` 现在**按唯一键传参**：

| 接口 | 现入参 | 唯一键 |
|---|---|---|
| `POST /api/opener/delete` | `{name}` | name |
| `POST /api/opener/reorder` | `{names: string[]}` | name |
| `POST /api/forge/delete` | `{host}` | host（归一化） |
| `POST /api/forge/reorder` | `{hosts: string[]}` | host（归一化） |
| `POST /api/forge/account/delete` | `{forgeHost, username}` | forgeHost + username |
| `POST /api/forge/account/reorder` | `{keys: string[]}`，元素为 `"host/username"` 字符串协议 | forgeHost + username |
| `POST /api/project/scan-rule/delete` | `{path}` | path |
| `POST /api/project/scan-rule/reorder` | `{paths: string[]}` | path |
| `POST /api/project/clone-rule/delete` | `{repoHost, repoPrefix}` | repoHost + repoPrefix |
| `POST /api/project/clone-rule/reorder` | `{rules: CloneRuleKey[]}` | repoHost + repoPrefix |
| `POST /api/template/source/delete` | `{name}` | name |
| `POST /api/template/source/reorder` | `{names: string[]}` | name |

问题：

1. **唯一索引逻辑泄露到前端**。前端必须知道「clone 规则 = host+prefix 唯一」「account = host/username 唯一」这类领域规则并自行拼键（`forge-accounts.tsx` 拼 `` `${forgeHost}/${username}` ``、`clone-section.tsx` 拼 `{repoHost, repoPrefix}`）。唯一索引一旦调整，前后端要同步改。
2. **为对外传键而导出的 Key 类型**（`project.CloneRuleKey`、account 的 `"host/username"` 字符串协议 + `key2acct` 解析）。唯一索引逻辑本身是需要的，但它只该是各 domain settings 处理的内部实现，不该进 Service 公开签名、handler 入参和 `schema.d.ts`。
3. **reorder 的「按键重排 + 未列出殿后」是自作聪明的合并**。用户在配置页操作，语义就是「以我提交的为准」，覆盖比合并更可预期。既然要传整表，reorder 的实际功能就是「保存全部」，名字应如实反映。

## 关键决策

### D1：delete 传完整单项数据

- 六个 delete 接口入参改为**与 save 同构的完整单项**（前端直接把行数据交回去，不拼键）。
- domain Service 签名改为收实体：`DeleteOpener(spec Spec)` / `DeleteForge(f Forge)` / `DeleteAccount(a Account)` / `DeleteScanRule(rule ScanRule)` / `DeleteCloneRule(rule CloneRule)` / `DeleteTplSource(src TplSource)`。
- domain 内部用私有 `keyOf` 从实体算唯一键定位（含与 save 一致的归一化：forge host `NormalizeHost`、account `NormalizeHost`/`NormalizeUsername`、tplSource trim）；**只按唯一键匹配，不比对其他字段**——前端行数据可能是稍旧的展示数据，按唯一索引删已足够。
- 未找到仍返回中文错误（保持现行为）。
- 级联语义不变：删 forge 连带删其 account；删 opener 连带清理 `openerIntents` 引用。

### D2：reorder 改名 save-all，语义为整节覆盖

| 旧 | 新 |
|---|---|
| `POST /api/opener/reorder` | `POST /api/opener/save-all` |
| `POST /api/forge/reorder` | `POST /api/forge/save-all` |
| `POST /api/forge/account/reorder` | `POST /api/forge/account/save-all` |
| `POST /api/project/scan-rule/reorder` | `POST /api/project/scan-rule/save-all` |
| `POST /api/project/clone-rule/reorder` | `POST /api/project/clone-rule/save-all` |
| `POST /api/template/source/reorder` | `POST /api/template/source/save-all` |

- 入参统一为 `{items: Item[]}`，`Item` 与 save 单项同构；提交的列表（含顺序）**即落盘结果**，last-write-wins，不做合并、不做集合一致校验。未提交的条目即被删除，新增的条目即被新增。
- Service 方法：`SaveAllOpeners` / `SaveAllForges` / `SaveAllAccounts` / `SaveAllScanRules` / `SaveAllCloneRules` / `SaveAllTplSources`，删除对应 `ReorderXxx`。
- **整体原子**：每一项过与 save 相同的单项校验（含归一化）+ 节内唯一键查重 + 节级约束，任一失败整体拒绝、不落盘。
- **空数组合法**，等价清空该节（覆盖语义一致，不特判）。
- **读侧坏条目随覆盖丢弃**：现 `ReorderOpeners` 会把 list 不可见的坏条目原序保留殿后；save-all 以提交为准，坏条目（本就不可见不可用）不再保留。属有意的行为变化。

save-all 天然可增删改，副作用须与单项 save/delete 对齐（逐节）：

| 节 | 单项校验 / 归一化 | 节级约束 | 副作用 |
|---|---|---|---|
| openers | name 非空 + `InitActionOpener` 构造校验 | name 唯一 | 新列表中不存在的 opener，连带清理 `openerIntents` 的默认/候选引用（复用 `DeleteOpener` 的清理逻辑，按「被移除名集合」批量处理） |
| forges | `NormalizeHost` + 现 save 校验 | 归一化 host 唯一 | 被移除 forge 连带删其 account（复用 `retainAccounts`） |
| forgeAccounts | `NormalizeHost` / `NormalizeUsername` + `ValidateAccount`（forge 须存在） | host+username 唯一 | **token 掩码回填**：提交 `TokenMasked` 的项按唯一键取旧 token 回填；旧值不存在（新条目）则报错——同 `SaveAccount`。拉取缓存不清理（同现 `DeleteAccount`） |
| scanRules | group 非空 / maxDepth ≥ -1 / 绝对路径且目录存在 / icon 校验 / tags 归一化 | path 唯一 + **group 唯一** | 成功后立即重扫（同现 reorder） |
| cloneRules | repoHost 非空 / repoPrefix 以 `/` 开头或为空 / localPath 绝对路径 | host+prefix 唯一 | 无 |
| tplSources | trim + `ValidateTplSource` | name 唯一 | 无 |

实现上把「单项校验 + 归一化」从现 save 函数中抽成每节一个私有 `validateXxx(item) (Item, error)`，save / save-all 共用，避免两份校验漂移。scan 规则 group 唯一的现有检查改为对「新列表」整体查重（save 时等价于对 upsert 后的列表查重）。

### D3：唯一键收敛为 domain 私有

- 删除导出类型 `project.CloneRuleKey`（及 `formatCloneRuleKey` 若无他用）；clone 规则键改为私有（如私有 struct 或 `[2]string`）。
- account 的 `key2acct`（解析 `"host/username"` 对外字符串协议）随 reorder 删除；`acctCacheKey` 作为拉取缓存键仍保留（内部用途，不出 API）。
- `pkg/x/keyed.go`：`ReorderKeyed` 删除（无调用方）；新增节内唯一键查重原语（如 `DuplicateKeyed(list, keyOf) (K, bool)`，返回首个重复键供错误文案渲染），六节 save-all 共用。`UpsertKeyed` / `RemoveKeyed` 保留。

### D4：前端写后全量重拉

原则：settings 页**以服务端列表为唯一事实源**，任何写操作成功后重新拉取受影响的全部查询，不在本地做补丁式更新。

- 所有 save / delete / save-all mutation 的 `onSuccess` invalidate **本节 + 受级联/副作用影响的查询**；save-all 的 `onError` 同样 invalidate（回滚拖拽乐观顺序）：

| mutation | invalidate | 现状 |
|---|---|---|
| opener save / delete / save-all | `['opener']`（list + intents） | ⚠️ 现只刷 `['opener','list']`，delete 级联清 intents 后 intents 视图陈旧 |
| forge / account 全部写 | `['forge']`（list + accounts + overview） | 已满足（前缀覆盖） |
| scan-rule save / delete / save-all | `['project','scan-rules']` + `['project','list']` | ⚠️ 现漏刷项目列表——写后后端已重扫，项目页列表/分组陈旧 |
| clone-rule 全部写 | `['project','clone-rules']` | 已满足 |
| tplSource 全部写 | 模板源查询 | 已满足 |

- `onSuccess` **返回** invalidate 的 promise（不 `void` 掉），让 mutation 在重拉完成后才 settle。
- `useDragOrder`（`web/src/pages/settings/drag-order.ts`）的乐观顺序 `order` 现只在「列表长度变化」时失效——重拉回来的服务端顺序若与本地不同（他端改过）会被本地 `order` 掩盖。改为 save-all settle（成功或失败）后清空 `order`，回落到服务端列表；配合上条，清空时数据已是重拉后的新列表，不会闪回旧顺序。
- save-all 提交内容 = 拖拽后顺序的当前列表行；若 list DTO 含 save 入参之外的派生/展示字段，提交前映射为 Item 形状。delete 同理直接交回行数据。

## 范围外

- `POST /api/opener/intent-default/delete`：`openerIntents` 是 map 节而非清单，该接口语义是「清除某 intent 的默认 opener 字段」而非删除条目，定位参数 `intent` 是领域枚举本身、不存在拼键问题。不改。
- `POST /api/workbench/branch/delete`：非 settings 配置。
- CLI：无命令调用上述 delete / reorder，不受影响。
- 兼容期：server 与内嵌前端同版本发布、单用户本地使用，旧 reorder 路由直接删除，不保留别名。

## 实施步骤

1. **domain + pkg/x**：抽 `validateXxx`；Delete 签名改收实体；新增 `SaveAllXxx`（含 D2 副作用表）、删除 `ReorderXxx`；删 `CloneRuleKey` / `key2acct` / `ReorderKeyed`，加查重原语。
2. **handlers**：各节抽命名 Item DTO（save / delete / save-all 共用，save-all 为 `{items: Item[]}`）；路由 `reorder` → `save-all`。
3. `just gen-api` 再生成 `web/src/api/schema.d.ts`。
4. **前端**：queries 六节 mutation 改名/改入参 + D4 invalidate 表；`useDragOrder` settle 后清空 order；各 section 组件 delete 传行数据、拖拽调 save-all。
5. **文档**：`docs/spec/现状.md` API 表（delete 描述 + reorder 行改 save-all）与 3.3 节「三原语」描述（顺带纠正其中 `pkg/slicekit`（keyed.go）的过期位置，实际在 `pkg/x/keyed.go`）。
6. `just fmt` / `just lint` / `just test`；`web/` 下 `pnpm fix`。

## 测试要点

- domain（替换现 `TestReorder*` / `TestServiceReorder` / tplsource reorder 用例）：
  - save-all 覆盖：顺序即落盘序；未提交条目被删除；新增条目落盘；空数组清空节。
  - 拒绝：重复唯一键（含归一化后重复，如 forge host 大小写）；任一项校验失败整体不落盘（断言文件未变）。
  - 副作用：opener 移除连带清 intents；forge 移除连带删 account；account 掩码回填 / 新条目提交掩码报错 / forge 不存在报错；scan group 冲突报错；scan save-all 后重扫生效。
  - delete 传整项：仅按唯一键匹配（其他字段不同也能删）；未找到报错；归一化后匹配（forge host 大小写）。
- handlers：save-all / delete 新入参的路由与 envelope 契约；旧 reorder 路由 404。

## 实施记录

与上文设计的偏差：

- **account 的 `key2acct` 保留**：它同时服务于拉取缓存键反解（`loadRepos`），属 domain 内部用途；随 reorder 移除的只是「对外 `host/username` 字符串协议」。
- **单条 `SaveScanRule` 保留「只拦与本条冲突的 group」**，不对整节查重——手工编辑遗留的他人间 group 重复已由读侧降级，不应阻断无关规则的保存；整节查重只用于 save-all。
- **opener 的 save 也刷 `['opener']` 整个前缀**：intents 的候选清单由 openers 读侧合成，单条 save 同样会改变 intents 视图（D4 表原只列了 delete / save-all）。
- **opener 的 save-all / delete 提交前经编辑表单同款映射还原**：list DTO 带后端合成的默认 title，直接回交会把它固化落盘。
- **handler 的 Item DTO 与 save 入参合一**：各节一个命名 `XxxItem`，save / delete 的 `Body` 直接用它，save-all 为 `{items: XxxItem[]}`。

顺带修复（实施中暴露的既有问题）：

- **`pkg/xstore`（WIP 提交中的未引用包）**：依赖已删除的 `ReorderKeyed`，按本提案语义把 `Reorder` 改为 `SaveAll`，`Delete` 改收整项。
- **`gen:api` 脚本输出路径**：`just -f ../justfile` 在仓库根执行，spec 被写到仓库根 `openapi.json`，openapi-typescript 却读 `web/openapi.json` 的旧文件，生成结果静默倒退；改为 `-o web/openapi.json`。
- **`forge.LocalRepo` 的 json tag**：lint 清零重构（9711fbb）把 `repoUrl` 误改为 `repoURL`，前端 forges 页孤儿行读 `repoUrl` 取到 undefined；恢复为 `repoUrl`（与其他 API 一致）。
