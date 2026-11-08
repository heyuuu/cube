// Package projcache 提供项目状态快照的本地缓存（git 信息 + 打开目标元数据）。
//
// 对几十个 git 项目的全量采集是 IO 密集型操作，每次都跑会明显卡顿。
// 本包把采集结果缓存到 ~/.config/cube/cache/projcache.json：
//   - CLI（短命进程）启动时 Load 一次快照到内存，只读不刷新；
//   - 常驻 server 进程内 goroutine 定时 Refresh 采集，写内存 + flush projcache.json。
//
// projcache.json 的角色是「跨重启持久化缓存」：server 重启后秒恢复，CLI 读最近一次落盘。
// 原名 gitcache/git.json，v3 起快照含 workspace 信息更名 projcache，v4 起文件同步更名。
package projcache

import (
	"cmp"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/heyuuu/x/store"
	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/domain/project/workspace"
	"cube/pkg/git"
)

// 当前缓存文件格式版本；结构变更时递增，用于后续做兼容迁移。
// 版本号只在「上一版格式被真实使用过」后才递增——开发期连续演进不逐次 bump。
// v2：worktree 归并为项目打开目标（1032）——移除 WorktreeMain（worktree 不再是独立项目），
// 新增 Worktrees（主项目附带枚举 linked worktree）。
// v3：包更名 projcache + workspace 字段（1030）——Entry/WorktreeInfo 各增 Workspaces。
// v4：主工作区拆出为独立 Main 值字段（恒存在，类型强制），Worktrees 只含 linked worktree；
// 移除 Entry 顶级 RepoURL/CurrentBranch/Workspaces/Dirty（改经 OriginURL/Main 取用，
// dirty 迁入 WorktreeInfo 按工作区采集）；采集全有或全无（任一失败整条丢弃）；文件更名 projcache.json。
const cacheVersion = 4

// 缓存文件名（位于缓存目录 dir 下）。
const cacheFileName = "projcache.json"

// Entry 单个项目的 git 信息快照。
type Entry struct {
	Remotes       []RemoteInfo   `json:"remotes"`       // 全部 remote（无则空数组；origin 地址经 OriginURL 派生）
	DefaultBranch string         `json:"defaultBranch"` // 默认主分支名（master/main/...）
	Branches      []string       `json:"branches"`      // 本地+远程分支短名列表
	Ahead         int            `json:"ahead"`         // 默认分支相对 origin 的领先 commit 数
	Behind        int            `json:"behind"`        // 落后的 commit 数
	Main          WorktreeInfo   `json:"main"`          // 主工作区信息
	Worktrees     []WorktreeInfo `json:"worktrees"`     // 各工作区信息(linked worktree)
	UpdatedAt     time.Time      `json:"updatedAt"`     // 本次采集时间
}

// RemoteInfo 快照中的单个 remote（多 remote 项目的 forge 匹配需要全部 host，1042 修）。
type RemoteInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// WorktreeInfo 单个工作区信息(主项目根或单个 linked worktree)
type WorktreeInfo struct {
	Path       string                `json:"path"`       // worktree 绝对路径（git 输出经符号链接规范化）
	Branch     string                `json:"branch"`     // 检出分支短名；detached 为空
	Detached   bool                  `json:"detached"`   // HEAD 游离（展示名回退目录名的信号）
	Dirty      bool                  `json:"dirty"`      // 该工作副本是否有变更（staged/unstaged/untracked 任一）
	Workspaces []workspace.Workspace `json:"workspaces"` // 该 worktree 根的 workspace 成员（1030，Path 相对 worktree 根）
}

// OriginURL origin remote 的地址（采集侧已归一 fetch 优先 push）；无 origin 返回空。
func (e *Entry) OriginURL() string {
	for _, r := range e.Remotes {
		if r.Name == "origin" {
			return r.URL
		}
	}
	return ""
}

// cacheFile 缓存文件的磁盘序列化结构。
// key = 项目绝对路径。
type cacheFile struct {
	Version   int               `json:"version"`
	UpdatedAt time.Time         `json:"updatedAt"`
	Entries   map[string]*Entry `json:"entries"`
}

// Cache 内存态缓存。读多写少，用 RWMutex 保护 entries map。
//
// 并发模型（单写者）：
//   - 常驻 server 是 projcache.json 的唯一写方，CLI 只读不写，无跨进程并发写问题。
//   - 进程内：RWMutex 保护 map 读写；Refresh 每次整表重建（全新 map 覆盖 c.entries）。
//   - 落盘靠 Save() 的原子写（tmp + rename）保证，无需 flock。
type Cache struct {
	cacheFile string // 缓存文件路径
	mu        sync.RWMutex
	entries   map[string]*Entry // key = 项目绝对路径
	updatedAt time.Time         // 最近一次采集落盘时间（缓存整体刷新时间）
}

// UpdatedAt 返回缓存最近一次落盘时间（整体刷新时间，区别于单项目的 Entry.UpdatedAt）。
func (c *Cache) UpdatedAt() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.updatedAt
}

// Load 从 dir 加载缓存。
// 行为约定（降级优先，绝不因缓存问题阻塞 CLI）：
//   - 文件缺失 / 解析失败 / 版本不符：一律当文件不存在，返回空缓存。
func Load(dir string) (*Cache, error) {
	c := &Cache{
		cacheFile: filepath.Join(dir, cacheFileName),
		entries:   make(map[string]*Entry),
	}

	// 尝试读取缓存文件，读取失败直接丢弃
	file, err := store.LoadJSON[cacheFile](c.cacheFile)
	if err != nil || file.Version != cacheVersion {
		return c, nil
	}

	c.mu.Lock()
	if file.Entries != nil {
		c.entries = file.Entries
	}
	c.updatedAt = file.UpdatedAt
	c.mu.Unlock()

	return c, nil
}

// Get 读取单个项目的缓存条目；未命中返回 (nil, false)。
func (c *Cache) Get(path string) (*Entry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[path]
	return e, ok
}

// Size 返回缓存条目数（最近一次采集成功的项目数）。
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Save 落盘 projcache.json（store.SaveJSON：Deterministic 字节稳定 + 缩进 + 尾换行 + 原子写）。
func (c *Cache) Save() error {
	c.mu.Lock()
	now := time.Now()
	c.updatedAt = now
	file := cacheFile{
		Version:   cacheVersion,
		UpdatedAt: now,
		Entries:   c.entries,
	}
	c.mu.Unlock()

	if err := store.SaveJSON(c.cacheFile, file); err != nil {
		return fmt.Errorf("写入 cache 文件失败: %w", err)
	}
	return nil
}

// Refresh 用给定的项目路径列表并发采集 git 信息，整表重建后写回内存 + 落盘。
//
// 整表重建语义：以本次采集结果为准，旧 entries 被完全覆盖——
//   - 采集成功的项目写入新 entry；
//   - 采集失败（collectEntry 返回 error）或已不在 paths 中的项目，其 entry 不进入新表，自然丢弃。
//
// 刻意不保留失败项目的旧 entry：避免某个项目长期异常、旧快照却一直存在而不被发现。
//
// 入参用 []string（项目绝对路径）而非 []*project.Project，刻意解耦对 project 包的依赖，
// 避免 project → projcache → project 的循环 import。
//
// 并发模型：固定 defaultWorkers 个 worker 从 tasks channel 抢活——worker 数即并发上限
// （采集是子进程 + IO 密集型，过高并发会与系统其他 IO 抢资源），无需额外信号量。
// 单项目采集失败（collectEntry 返回 error）会 Warn 记录后跳过。
func (c *Cache) Refresh(paths []string) error {
	start := time.Now()
	entries := collectEntries(paths)
	c.mu.Lock()
	c.entries = entries
	c.mu.Unlock()
	if err := c.Save(); err != nil {
		return err
	}
	// 采集汇总日志：排查「缓存为何没更新 / 采集耗时异常」时直接查这一条。
	// 失败数 = 项目数 - 成功数（失败项目另有逐条 Warn 日志）。
	// 耗时用 String() 而非 time.Duration 原值：JSON handler 会把 Duration 序列化成纳秒整数，不可读。
	slog.Info("git 缓存采集完成",
		"项目数", len(paths),
		"成功", len(entries),
		"失败", len(paths)-len(entries),
		"开始", start.Format("15:04:05.000"),
		"结束", time.Now().Format("15:04:05.000"),
		"耗时", time.Since(start).Round(time.Millisecond).String(),
	)
	return nil
}

// RefreshOne 定向刷新单个项目的快照条目并落盘（提案 1031：worktree/分支写操作
// 成功后即时可见，不等 TTL 整表重建）。与 Refresh 的整表重建语义不同：只覆盖
// 该路径的 entry，其余项目保持原样。采集失败不落盘、保留旧 entry，返回错误
// 由调用方决定是否上抛（写操作本身已成功，通常 Warn 后依赖 TTL 自愈即可）。
func (c *Cache) RefreshOne(path string) error {
	entry, err := collectEntry(path)
	if err != nil {
		return fmt.Errorf("定向刷新 git 缓存失败: %w", err)
	}
	c.mu.Lock()
	c.entries[path] = entry
	c.mu.Unlock()
	return c.Save()
}

// defaultWorkers 默认并发数。
// 偏保守：采集是子进程 + IO 密集型，过高并发会与系统其他 IO 抢资源。
const defaultWorkers = 8

func collectEntries(paths []string) map[string]*Entry {
	if len(paths) == 0 {
		return make(map[string]*Entry)
	}

	type result struct {
		path  string
		entry *Entry
	}
	tasks := make(chan string)
	results := make(chan result, len(paths))

	// 固定数量 worker，从 tasks 取任务执行
	var wg sync.WaitGroup
	for range defaultWorkers {
		wg.Go(func() {
			for path := range tasks {
				entry, err := collectEntry(path)
				if err != nil {
					slog.Warn("采集 git entry 失败", "path", path, "err", err)
					continue // 采集失败跳过，该项目不进入本次结果（整表重建，不保留旧 entry）
				}
				results <- result{path: path, entry: entry}
			}
		})
	}

	// 派发任务（独立 goroutine，避免 tasks 写入与 worker 读取互锁）
	go func() {
		for _, p := range paths {
			tasks <- p
		}
		close(tasks)
	}()

	// worker 全部退出 = 所有结果已写入 results，可安全关闭
	go func() {
		wg.Wait()
		close(results)
	}()

	// 合并本次采集结果（只含采集成功的项目）
	entries := make(map[string]*Entry, len(paths))
	for r := range results {
		if r.entry != nil {
			entries[r.path] = r.entry
		}
	}
	return entries
}

// collectEntry 采集单个项目的 git 信息。全有或全无：任一 git 子命令失败即整体失败
// （返回 error，调用方丢弃该条目），不接受残缺数据——残缺快照在消费侧（打开目标、
// forge 对账）比没有快照更糟。业务空值（无 remote、detached 等）不是失败，落零值。
func collectEntry(path string) (*Entry, error) {
	// Remotes 一次拿到全部 remote 快照（origin 地址经 OriginURL 派生）
	gitRemotes, err := git.Remotes(path)
	if err != nil {
		return nil, err
	}
	remotes := xslices.Map(gitRemotes, func(r git.Remote) RemoteInfo {
		return RemoteInfo{
			Name: r.Name,
			URL:  cmp.Or(r.Fetch, r.Push),
		}
	})

	// Refs 信息
	refs, err := git.Refs(path)
	if err != nil {
		return nil, err
	}
	branches := xslices.Map(refs.Locals, func(r git.Ref) string { return r.ShortName })
	defaultBranch := calcDefaultBranch(branches)

	// ahead/behind 用仓库的默认分支（master/main/...）做本地 vs 远程比较——
	// 刻意不用 LoadRepoStatus 的 branch.ab：那是「当前检出分支 vs 其 upstream」，
	// 在 feature 分支上时与列表页要的口径不同
	var ahead, behind int
	if defaultBranch != "" {
		ahead, behind, _ = git.AheadBehindRemote(path, defaultBranch, "origin", defaultBranch)
	}

	// 收集各工作区信息（dirty 按工作区各自探测，主区与 linked 同路径）
	worktrees, err := collectWorktrees(path)
	if err != nil {
		return nil, err
	}
	if len(worktrees) == 0 {
		return nil, fmt.Errorf("工作区信息错误，至少应该包含主工作区: path = %s", path)
	}

	return &Entry{
		Remotes:       remotes,
		DefaultBranch: defaultBranch,
		Branches:      branches,
		Ahead:         ahead,
		Behind:        behind,
		Main:          worktrees[0],
		Worktrees:     worktrees[1:],
		UpdatedAt:     time.Now(),
	}, nil
}

// collectWorktrees 枚举全部工作区（主目录在前，调用方拆分为 Main 与 linked Worktrees）。
// WorktreeList 输出主目录在前；枚举失败上抛（collectEntry 全有或全无，不降级）。
func collectWorktrees(path string) ([]WorktreeInfo, error) {
	list, err := git.WorktreeList(path)
	if err != nil {
		return nil, fmt.Errorf("枚举 worktree 失败: path=%s, err=%w", path, err)
	}
	worktrees := make([]WorktreeInfo, 0, len(list))
	for _, wt := range list {
		// 目录已删但 git 元数据未 prune 的 worktree 仍会被列出，按存在性过滤，
		// 避免失联路径进快照成为打不开的幽灵目标
		if _, err := os.Stat(wt.Path); err != nil {
			continue
		}
		dirty, err := git.IsDirty(wt.Path)
		if err != nil {
			return nil, fmt.Errorf("探测工作区 dirty 失败: path=%s, err=%w", wt.Path, err)
		}
		worktrees = append(worktrees, WorktreeInfo{Path: wt.Path, Branch: wt.Branch, Detached: wt.Detached, Dirty: dirty, Workspaces: workspace.Resolve(wt.Path)})
	}
	return worktrees, nil
}

// calcDefaultBranch 从本地分支清单推断默认分支：master/main 优先（覆盖绝大多数仓库），
// 都不存在返回空（ahead/behind 放弃比较、前端展示回退）。比远端探测（git.DefaultBranch
// 再起一个子进程）省一次调用，代价是 develop 等非惯例默认分支的仓库拿不到该数据。
func calcDefaultBranch(branches []string) string {
	for _, branch := range []string{"master", "main"} {
		if slices.Contains(branches, branch) {
			return branch
		}
	}
	return ""
}
