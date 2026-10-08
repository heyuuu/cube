# 模板源映射与临时拉取（tplSources）

> **状态**：✅ 已交付（2026-10-08）。第一步（settings 页「模板源」分区 + tplSources 节 CRUD）见 commit 5ad3244；第二步（create 链路切换临时拉取 + @ 语法解析重写）同日落地。实施偏差回写：repoUrl 校验放行**本地绝对路径**第三形态（git 原生支持 clone 本地路径，模板仓库放本地盘是正当用法，也让 E2E 可用本地仓库测全链路）；顺带修复 `collectVariables` 拼写检查条件反写的既有 bug（cliVars 传已声明变量被误报）。

## 背景

cube create 的 git 模板源经历了三次定位摇摆：

1. **1009 原始定稿**：git 来源 clone 到临时目录，**不缓存**；但引用只能输完整 url，太长。
2. **a84d338**：为解决「url 太长」引入本地管理——clone 到 `~/.config/cube/tpl/<name>` 常驻缓存，`InitGitSource` 拉取与使用分离。但为此背上一整套状态机（更新时机、dirty 策略、缓存目录对账），且 tpl/ 目录违反项目 cache 哲学（删了要重新 add，「可整体删除且行为不变差」不成立）。
3. **本提案（回摆到轻量）**：url 太长的真需求只需要一个 **name ↔ repoUrl 映射**，不需要本地缓存机制。映射进 settings.json（forge 同款），create 时按映射临时 clone 到系统 tmp、用后即删——1009 原始形态 + 注册表，一整类管理问题（更新/dirty/对账/add 命令组）整体消失。

## 关键决策

### D1：映射存 settings.json `tplSources` 节（已落地）

`[{name, repoUrl}]` 数组（顺序即 create 交互选择序），name 唯一键。读写/校验/CRUD/API/Web 分区全部照抄 forge 模式（直读不缓存、写侧中文校验、坏条目跳过）。Web settings 页「模板源」分区管理，CLI 不加管理命令（forge 先例：配置管理走 Web）。

### D2：引用语法 `@<source>/<tpl>`，npm scope 式 @ 前缀

三形态由首字符零歧义区分：

```
@core/go-cli ~/x    @ 前缀 → 命名源 core 的 go-cli
@core ~/x           @ 前缀无 / → 命名源 core，交互选模板
go-cli ~/x          裸名 → 默认源 core（DefaultSource 常量）的 go-cli
~/code/tpl ~/x      . ~/ / 开头 → 本地路径直读（开发态：正在编辑的模板仓库）
```

**业内对照**（选型依据）：nix `registry#attr`（符号注册表同构但 `#` 是 shell 注释符，排除）；helm `repo/chart` / brew tap（裸 `/` 靠「首段命中注册表」隐式消歧，弱于显式标记）；npm `@scope/pkg`（@ 前缀对齐，肌肉记忆）；cookiecutter `gh:owner/repo`（无注册表，正是 url 太长的痛点本源）；go/npm 的 `pkg@version`（**@ 尾部 = 版本是业内共识**，`source@tpl` 语法会占用未来 ref 锁定空间，否决）。

`@` 尾部语法空间保留给未来的版本/ref 锁定（如 `@core/go-cli@v1.2`），本期不做。

### D3：create 时临时 clone，不缓存

- `os.MkdirTemp("", "cube-tpl-")` + `git.Clone(url, depth=1)`，生命周期 = 单次 create 进程（defer RemoveAll，失败也清理）。
- **clone 时机后移**：validateTarget 之后——不能让用户答完来源/变量、目标路径预检失败时白 clone 一次。
- 本地路径 source 不 clone 直读，是「开发态」的互补入口：**名字 = 发布态（永远最新），路径 = 开发态（未推送的改动）**。
- create 低频（周/月级），每次 1~3s 的浅 clone 换零状态管理，性价比成立。

### D4：默认源 core 不做自动初始化

缺失时报错指引设置页（缺 core 时报错文案附 `DefaultSourceRepoUrl` 建议值）。不做首次自动 clone（引擎无隐藏行为）、不做安装时 seed（settings 混默认值，url 变更要随版本迁移）。

## 实施清单

- [x] 第一步：template domain TplSource CRUD + handlers 四端点 + Web「模板源」分区（默认源徽标 + 缺失提醒）
- [x] domain：`loadSourceDir`（绝对路径直读 / 名字查注册表 → `cloneSource` 临时 clone + cleanup）；删 `Source`/`SourceType`/`NewSource`/`InitGitSource`（无消费者）
- [x] service：`Create` 预检先行 → clone 后移 → defer cleanup；`NewService(settingsFile)` 单参数；删 `Paths.TplSourceDir`
- [x] cmd：`parseCliTpl` 重写（@ 前缀结构校验：@ 后非空、tpl 名无 `/`、裸名无 `/`）+ 修 a84d338 引入的 `Cut("@")` bug + 表驱动单测
- [x] settings 旧节 `create.templateSource` 废弃（本就是死配置，不做迁移；dev 环境手删）
- [x] 测试：loadSourceDir 三形态 + cleanup 断言 + git 源 E2E（testfixture 本地仓库当 repoUrl）+ 预检先行（坏源不触发 clone）
- [x] 现状.md 同步（3.12 机制段 + 命令表 + Template API 表 + settings 示例）

## alternatives（被否方案）

- **本地缓存管理机制**（a84d338 方向的完全体）：tpl/ 目录 + `tpl add/list/update/remove` 命令组 + dirty/分叉策略 + 注册表对账。为低频场景常驻状态机，且「永远最新」反而做不到（要回答何时更新）。仅在「离线可用 + 重复使用零延迟」是硬需求时才值得，个人工具场景不是。
- **name↔url 映射读本地 clone 的 git remote**（零配置）：删 tpl/ 即失联（不可丢弃缓存），remote 被手动改后语义静默漂移。url 是「如何重建缓存」的知识，应存配置而非从缓存反推。
- **`source@tpl` 分隔符语法**：与版本后缀业内共识冲突 + npm 惯例反向，无消歧增益（见 D2）。
- **扁平命名空间**（apt 式：源注册后模板名全局可见）：同名模板跨源冲突需引入解析顺序，隐式规则成本高于 `@source/` 显式前缀。
