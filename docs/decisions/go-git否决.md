# go-git 否决：git 读操作维持原生子进程

> **状态**：已裁决，2026-11-08 实测复核维持原判。
>
> **一句话结论：go-git 的 `Status()` 问题使得真实项目上使用 go-git 基本不可行。** 元数据读取（refs/分支/remotes）在小仓库确实快 ~15x，但 status 是采集的必经步骤，它在真实前端项目上反向慢 1~2 个数量级，叠加全局忽略链缺失与 worktree 枚举缺口，不构成翻案理由。

## 这份文档的用途

未来再想起 go-git 时（诱因通常是「省 fork 开销」「纯库跨平台更可控」「避免子进程」），**先读本文再决定**。结论与数据截至 go-git v5.19.3（2026-11 时最新），复核条件见文末。

## 历史决策链（本仓库）

| commit | 动作 | 教训 |
|---|---|---|
| `22f43f9` | 引入 go-git 做 git 读操作 | 动机：降低 CLI 批量采集的 fork 进程数 |
| `f194884` | vendored 补全局忽略规则 | Status 不加载全局忽略链 → dirty 误报，得在 vendor 层修 |
| `acbfe19` | 读操作全部迁回原生子进程，移除依赖 | 双轨维护成本 + 大仓库性能不可接受，终局裁决 |

projcache 异步采集（快照 + 8 worker 并发）落地后，「避免 fork 开销」的原始动机已消失——spawn 成本被摊平到后台，读路径零感知。

## 2026-11-08 实测数据

环境：go-git **v5.19.3** / git 2.55.0 / go 1.27.1 / macOS arm64（darwin 25.6）。子进程侧严格复刻 `pkg/git` 的 7 条采集命令（`remote -v` / `for-each-ref` / `symbolic-ref HEAD` / `symbolic-ref origin/HEAD`(+fallback) / `rev-list --count` / `status --porcelain=v2 --branch` / `worktree list`）与 `runOut` 环境注入。3 个合成 fixture + 5 个真实仓库，预热后多轮取中位数。

### 性能（单项目单轮中位数）

| 仓库 | 磁盘文件数 | 子进程 | go-git | 倍率 |
|---|---:|---:|---:|---|
| fix-small（合成） | ~40 | 55.7ms | **2.2ms** | go-git 快 25.7x |
| x（真实） | 91 | 52.7ms | **3.5ms** | go-git 快 14.8x |
| navi（真实） | 9.5k | 52.1ms | 135ms | 慢 2.6x |
| cube（真实） | 29k | 56.7ms | 797ms | 慢 **14x** |
| cube-templates（真实） | 50k | 55.0ms | 1450ms | 慢 **26x** |
| xixi（真实） | 268k | 62.9ms | ~3000ms | 慢 **47x** |

要点：

1. **子进程耗时与仓库规模基本无关**（52~85ms 平坦）：spawn 基线实测 6.2ms/条 × 7 条 ≈ 50ms 是底座；git status 本体在 26.8 万文件的仓库上也只要 52ms（C 实现 + 目录级忽略剪枝 + `--untracked-files=normal` 折叠 + index stat 缓存）。
2. **go-git Status 随磁盘文件总数线性爆炸**，与 tracked 数量无关（cube tracked 仅 443 也要 800ms）——决定性变量是工作区全量文件数（node_modules 之类）。
3. ahead/behind：tip 相同时哈希比较 O(1)（0.05ms）；有分叉时需自实现对象遍历，600 commit 历史上 31ms vs `rev-list` 11ms（慢 3x），且正确实现必须走全量祖先，历史越大差距越大。
4. 复用 Repository 实例（warm）无收益：Status 不缓存 index，每轮全量重扫。

### 功能差异

| 能力 | 结果 |
|---|---|
| 分支/refs/remotes/默认分支/HEAD | 8/8 仓库一致 ✓ |
| **dirty 判定** | **2/8 误报**：只被全局忽略链（`~/.gitignore_global`）覆盖的 `.idea/`、`.zcode/`、`.DS_Store` 被报 untracked → dirty 判错 |
| **linked worktree 枚举** | **无 API**（v5.19.3 全包无任何 worktree 枚举能力），projcache 1032 的硬需求无法覆盖 |
| untracked 语义 | 不做目录折叠：git normal 模式整目录报 1 项，go-git 逐文件列出（dirty 布尔不受影响，但口径不同） |
| ahead/behind | 无 `rev-list` 等价物。复测时自实现的第一版染色算法就在公共祖先剪枝上把 behind 多算（1/3、300/359）——佐证正确性风险真实存在，`rev-list` 久经考验 |

### 全局忽略链修复重测（正确性可修，性能不可修）

库留了现成接线点（`gitignore.LoadGlobalPatterns` 读 `~/.gitconfig` 的 `core.excludesFile`，与 git 同口径），注入后：

```go
ps, _ := gitignore.LoadGlobalPatterns(osfs.New("/"))
wt.Excludes = ps // excludeIgnoredChanges 里 patterns = append(patterns, w.Excludes...)
```

- dirty 判定 **8/8 全部对齐**（x、cube-templates 误报消除；正是历史 `f194884` 的同款修法）；
- 性能**不变或略慢**：cube 881ms、cube-templates 1568ms、xixi 3406ms——`Excludes` 只喂给后置过滤器，遍历成本一分没省。

## Status 为什么慢：源码根因（v5.19.3）

**架构是「先全量 diff、后过滤」，遍历成本在过滤发生前已全额支付**：

1. **过滤后置**：`worktree_status.go` 的 `diffStagingWithWorktree` 先 `merkletrie.DiffTree`（index 树 vs 文件系统树，**没有任何 ignore 概念**）跑完全量 diff，`excludeIgnoredChanges` 才拿 `.gitignore` 模式过滤 Changes 列表。
2. **walker 零剪枝**：`utils/merkletrie/filesystem/node.go` 的 `calculateChildren` 只跳过 `.git` 和 socket，每个目录无条件 ReadDir 下探（含 `node_modules/` 内部）；billy `osfs` 的 `readDir` 对每条目调 `entry.Info()` = **每文件一次 lstat**。
3. **双重遍历**：`gitignore/dir.go` 的 `ReadPatterns` 为收集嵌套 `.gitignore` 自身再做一次全树递归 ReadDir。
4. **对照 git 为什么不慢**：目录级忽略匹配发生在遍历中（命中即整目录跳过）、`--untracked-files=normal` 对无 tracked 文件的目录折叠不深入、全在 C。xixi 上 git 只 lstat 顶层几十个条目，go-git lstat 了 26.8 万个 ×2 趟。
5. 边界澄清：Insert-only 节点不读文件内容（merkletrie 只对两侧同名节点比哈希），且 v5.19 有 index metadata 快捷（size+mtime+mode 命中免内容哈希）——遍历是 lstat 级而非内容级，否则 xixi 的 3 秒会是 30 秒。
6. **API 内无解**：`StatusOptions` 只有 `Strategy`（决定 Status map 初始化，不影响遍历）；`w.Excludes` 只喂后置过滤器；没有 untracked 折叠模式、没有 walk 阶段剪枝开关。walk 剪枝接进 `calculateChildren` 理论可行，但要处理 `!` 反选模式（`dist/` + `!dist/keep/`）与模式优先级序，且那是改库/fork——`status.go` 里的 `TODO: (V6) Review the default status strategy` 表明上游自己也知道要重做。

## 判定

- go-git 唯一成立的优势场景：**工作区文件数少**的仓库元数据 + status 快 ~15x（2~3.5ms vs ~53ms）；在典型前端项目（node_modules 巨大）上反向慢 1~2 个数量级。
- 修掉全局忽略（3 行注入）只解决正确性：大仓库 status 仍 3.4s vs git 52ms（65x），worktree 缺口仍在，untracked 口径差仍在。
- 子进程方案耗时平坦、与仓库规模解耦，spawn 开销已被 projcache 异步采集摊平——**维持原生子进程，本项目不做双轨**（双轨维护坑见 [references/gitbatch.md](../references/gitbatch.md)；模板规范同步定为「系统 git 单轨」，见 [misc/project-template-spec.md](../misc/project-template-spec.md)）。
- 若未来要压采集耗时：方向是减少命令条数（如 `status --branch` 已含 branch.head/up/ab，可替代两条 `symbolic-ref`），不是换库。

## 复核条件（何时才重新评估）

同时满足才值得重测：

1. go-git **v6**（或 v5 新版）重写了 status 遍历：walk 阶段目录级剪枝 + untracked 折叠 + 默认加载全局忽略链；
2. 出现「目标环境无 git 二进制」的新部署形态（目前不存在）。

复测方法与判定标准：重建基准（子进程侧复刻上述 7 条命令，go-git 侧等价采集，预热后取中位数；基准代码曾在 `tmp/gitbench/`，gitignored 不入库，数据可从本文重建），大仓库（磁盘 >5 万文件）status 单步 < 100ms 且 dirty/ahead-behind/worktree 全对齐才进入候选。
