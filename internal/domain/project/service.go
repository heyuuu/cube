package project

import (
	"errors"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/config"
	"cube/internal/domain/project/projcache"
	"cube/pkg/easycache"
	"cube/pkg/fuzzy"
	"cube/pkg/git"
	"cube/pkg/pathkit"
)

// Service project 领域服务：scan/clone 规则管理 + 项目列表查询（列表走 easycache 惰性缓存）。
type Service struct {
	// settings
	scanRulesStore  ScanRulesStore
	cloneRulesStore CloneRulesStore
	// scan
	scanCache *easycache.Item[[]*Project] // 项目扫描的缓存（新鲜度时间戳由缓存自身维护，见 UpdatedAt）
	// 项目状态快照（projcache：git/worktree/workspace，落盘 cache/git.json）
	projCache *projcache.Cache
	// 定时刷新（仅常驻 server 启用，CLI 不启用）
	stopCh chan struct{} // nil = 未启用；非 nil = 定时器在跑
}

// NewService 构造 Service：加载 git 缓存（失败降级空缓存）并挂载项目扫描缓存。
func NewService(cfg *config.Config) *Service {
	// 加载项目状态快照（降级优先：失败返回空缓存，不报错）
	projCache, err := projcache.Load(cfg.CacheDir())
	if err != nil {
		log.Printf("加载 git 缓存失败: %v", err)
	}

	s := &Service{
		scanRulesStore:  newScanRulesStore(cfg.SettingsFile()),
		cloneRulesStore: newCloneRulesStore(cfg.SettingsFile()),
		projCache:       projCache,
	}
	// 扫描时现读规则（规则直读 settings.json，外部修改后 Reload 即可反映）
	s.scanCache = easycache.NewItem(func() []*Project {
		projects, err := scan(s.ScanRules())
		if err != nil {
			slog.Error("扫描项目失败", "err", err)
			return nil
		}
		slog.Info("scan 项目完成", "count", len(projects))
		return projects
	})
	return s
}

// -- 规则（直读 settings.json，加载即转换校验） --

// ScanRules 现读 settings.json 的 scan 规则（外部改动 Reload 后即反映）。
func (s *Service) ScanRules() []ScanRule {
	return s.scanRulesStore.All()
}

// CloneRules 现读 settings.json 的 clone 规则。
func (s *Service) CloneRules() []CloneRule {
	return s.cloneRulesStore.All()
}

// SaveScanRule 新增或按 path 替换一条 scan 规则（校验在写侧，坏数据返回中文错误）。
// 成功后立即重扫项目列表（毫秒级），保证 Web 保存后列表即时反映。
func (s *Service) SaveScanRule(rule ScanRule) error {
	if err := s.scanRulesStore.Save(rule); err != nil {
		return err
	}
	s.scanCache.Reload()
	return nil
}

// DeleteScanRule 按唯一键（path）删除 scan 规则，成功后立即重扫。
func (s *Service) DeleteScanRule(rule ScanRule) error {
	if err := s.scanRulesStore.Delete(rule); err != nil {
		return err
	}
	s.scanCache.Reload()
	return nil
}

// SaveAllScanRules 以提交清单整节覆盖 scan 规则；顺序影响项目列表展示序
// （scan 按规则序遍历），成功后同样立即重扫。
func (s *Service) SaveAllScanRules(rules []ScanRule) error {
	if err := s.scanRulesStore.SaveAll(rules); err != nil {
		return err
	}
	s.scanCache.Reload()
	return nil
}

// SaveCloneRule 新增或按 host+prefix 替换一条 clone 规则（校验在写侧）。
func (s *Service) SaveCloneRule(rule CloneRule) error {
	return s.cloneRulesStore.Save(rule)
}

// DeleteCloneRule 按唯一键（host+prefix）删除 clone 规则。
func (s *Service) DeleteCloneRule(rule CloneRule) error {
	return s.cloneRulesStore.Delete(rule)
}

// SaveAllCloneRules 以提交清单整节覆盖 clone 规则。
func (s *Service) SaveAllCloneRules(rules []CloneRule) error {
	return s.cloneRulesStore.SaveAll(rules)
}

// --- project 读操作 ---

// Projects 返回扫描项目列表（easycache 惰性加载，首次触发扫描）。
func (s *Service) Projects() []*Project {
	return s.scanCache.Get()
}

// cleanAbsPath 路径入参的统一预处理：仅接受绝对路径（相对路径视为调用方传错，
// 按未找到处理而非报错，与「查无此项目」语义一致），返回清理后的路径。
// ~ / 相对路径的展开是出口层职责（cmd 用 ExtendPath）。
func cleanAbsPath(path string) (string, bool) {
	if !filepath.IsAbs(path) {
		return "", false
	}
	return filepath.Clean(path), true
}

// FindByPath 按绝对路径查项目，未命中返回 nil（相对路径视为未找到）。
func (s *Service) FindByPath(path string) *Project {
	path, ok := cleanAbsPath(path)
	if !ok {
		return nil
	}

	for _, proj := range s.Projects() {
		if proj.Path() == path {
			return proj
		}
	}
	// 兜底：符号链接口径二次比对。项目路径来自扫描（用户配置口径的字面路径），
	// 而调用方传入的可能是 git 规范化后的路径（macOS /var → /private/var，如 worktree
	// 归并链路），反之亦然。规范化两侧任一侧即可对齐；只有精确匹配落空才付这笔 syscall。
	realPath, realErr := filepath.EvalSymlinks(path)
	for _, proj := range s.Projects() {
		if proj.Path() == realPath {
			return proj
		}
		if realErr == nil {
			if projReal, err := filepath.EvalSymlinks(proj.Path()); err == nil && projReal == realPath {
				return proj
			}
		}
	}
	return nil
}

// SearchByName 按名称模糊搜索项目列表。
func (s *Service) SearchByName(query string) []*Project {
	return fuzzy.MatchBy(query, s.Projects(), (*Project).Name, nil)
}

// OpenTargets 返回项目的打开目标列表（1032 根目录 + worktrees；1030 主/worktree 的 workspaces）。
// 排序定稿：主项目根 > 主项目 workspaces > 每个 worktree 根 > 该 worktree 的 workspaces。
// 只读 projcache 快照，不现场跑 git、不读 cube.json（遵守「读路径不得阻塞采集」）；项目未找到返回 nil。
func (s *Service) OpenTargets(path string) []OpenTarget {
	proj := s.FindByPath(path)
	if proj == nil {
		return nil
	}
	targets := []OpenTarget{{Path: proj.Path(), Label: rootLabel}}
	info, _ := s.GitInfo(proj.Path())
	// 存在性兜底过滤：采集侧已过滤，但目录在两次采集之间被删时快照仍残留，
	// 打开目标必须是真实可打开的目录（os.Stat 廉价，不违反读路径不跑 git 的纪律）
	for _, t := range targetEntries(proj.Path(), info) {
		if _, err := os.Stat(t.Path); err == nil {
			targets = append(targets, t)
		}
	}
	return targets
}

// OwnsDir 判断 dir 是否位于项目领地内：主根或其子目录、任一 linked worktree（快照内）
// 或其子目录。供 Web 打开接口做目标归属校验——前端目录树可从任意子目录发起打开，
// 不限于 OpenTarget 精确集合；worktree 归属只读快照，不现场跑 git。
func (s *Service) OwnsDir(path, dir string) bool {
	proj := s.FindByPath(path)
	if proj == nil {
		return false
	}
	if pathkit.IsUnder(dir, proj.Path()) {
		return true
	}
	info, _ := s.GitInfo(proj.Path())
	if info == nil {
		return false
	}
	for _, wt := range info.Worktrees {
		if pathkit.IsUnder(dir, wt.Path) {
			return true
		}
	}
	return false
}

// ResolveProject 把目标目录归并到所属主项目（「向上找当前项目」，与 SearchByPath 的
// 「向下列项目」方向相对）：目录是某项目根或其任意子目录直接命中；否则走 worktree
// 归并（ResolveMainProject）。workspace 成员是主根直系子目录，同样被第一步覆盖。
// 所有「拿一个实际目录反查项目」的出口（PickProject / alfred project-open /
// usage 路径归一）统一走这里，不要在调用方手拼 FindByPath + ResolveMainProject 的两段式。
func (s *Service) ResolveProject(path string) *Project {
	path, ok := cleanAbsPath(path)
	if !ok {
		return nil
	}

	// 逐项目查找
	for _, proj := range s.Projects() {
		// 判断要查找的目录是否为项目目录或子目录
		if pathkit.IsUnder(path, proj.Path()) {
			return proj
		}
	}

	// 尝试按 git worktree 目录反向找主项目
	return s.ResolveMainProject(path)
}

// ResolveMainProject 把任意目录归并到主项目（1032 路径归并链路）：
// 目录（或其祖先）是 linked worktree 时，顺着 .git 文件定位主仓库并返回对应项目；
// 非 worktree 或主仓库未收录（不在任何 scan-rule 下）返回 nil。
// 只做文件系统探测 + 列表查找，不跑 git 子进程。
func (s *Service) ResolveMainProject(dir string) *Project {
	main := git.WorktreeMain(dir)
	if main == "" {
		return nil
	}
	return s.FindByPath(main)
}

// SearchByPath 通过路径搜索项目列表（「向下列项目」，与 ResolveProject 的
// 「向上找当前项目」方向相对）：返回位于 path 之下（含 path 本身）的项目。
func (s *Service) SearchByPath(path string) []*Project {
	path, ok := cleanAbsPath(path)
	if !ok {
		return nil
	}

	var result []*Project
	for _, proj := range s.Projects() {
		if pathkit.IsUnder(proj.Path(), path) {
			result = append(result, proj)
		}
	}
	return result
}

// --- scan 相关 ---

// MatchScanRule 判断 absPath（git init 后）能否被 scan 收录为新项目（供 init 命令预检）。
// 返回匹配的规则与项目名；不满足收录条件时 ok=false。
func (s *Service) MatchScanRule(absPath string) (rule ScanRule, name string, ok bool) {
	return MatchScanRule(absPath, s.ScanRules())
}

// --- clone 相关 ---

// MatchCloneRule 以现读的 clone 规则做匹配（语义同包级 MatchCloneRule）。
func (s *Service) MatchCloneRule(repoURL string) (rule CloneRule, localPath string, ok bool) {
	return MatchCloneRule(repoURL, s.CloneRules())
}

// --- git 缓存相关 ---

// GitInfo 读取项目的 git 信息缓存条目；未命中返回 (nil, false)。
// 不阻塞、不触发采集 —— 调用方读取的是当前缓存快照（可能 stale）。
func (s *Service) GitInfo(path string) (*projcache.Entry, bool) {
	if s.projCache == nil {
		return nil, false
	}
	return s.projCache.Get(path)
}

// RefreshGitInfo 定向刷新单个项目的 git 快照并落盘（workbench 写操作后即时可见，
// 提案 1031）。经 app 装配点以方法值注入 workbench，避免 workbench 反向依赖 project。
func (s *Service) RefreshGitInfo(path string) error {
	if s.projCache == nil {
		return nil
	}
	return s.projCache.RefreshOne(path)
}

// ScanUpdatedAt 返回项目列表最近一次扫描完成时间；从未扫描过返回零值。
// 委托缓存的计算时间戳——数据与新鲜度由 scanCache 单点维护（对称：GitUpdatedAt 委托 projCache）。
func (s *Service) ScanUpdatedAt() time.Time { return s.scanCache.UpdatedAt() }

// GitUpdatedAt 返回 git 缓存最近一次落盘时间；无缓存返回零值。
// 区别于单项目的 CollectedAt：这是整个 cache 文件的刷新时间。
func (s *Service) GitUpdatedAt() time.Time {
	if s.projCache == nil {
		return time.Time{}
	}
	return s.projCache.UpdatedAt()
}

// maxExpireTime 缓存可容忍的最大过期时间：任一时间戳（scanCache / projCache 的
// UpdatedAt）距今超过它就触发刷新。定性约束「数据最多旧 5 分钟」，取代固定周期。
const maxExpireTime = 5 * time.Minute

// StartRefreshTicker 启动后台刷新 goroutine（仅常驻 server 调用）。重复调用安全：已在跑则直接返回。
//
// 刷新节奏由数据新旧决定：每轮取两类时间戳（scanCache / projCache 的 UpdatedAt）
// 中更过期者计算下次时机，保证距今不超过 maxExpireTime——
//   - 冷启动（从未扫描/采集）：时间戳零值，立即刷新一次，避免空窗；
//   - 热重载（两时间戳均新鲜，如 air 重启进程）：自然算出整周期等待，不重复全量重采；
//   - 刷新失败（时间戳不推进）：按整周期退避重试，避免 delay 恒为 0 热循环。
//
// 刷新动作 = Refresh（重扫毫秒级 + git 采集数十秒级，各自时间戳由缓存自记），
// 前端据两个时间戳分别判断「项目列表新鲜度」和「git 状态新鲜度」。
//
// CLI 不调用此方法（CLI 短命，只读启动时 Load 的快照）。
func (s *Service) StartRefreshTicker() {
	if s.projCache == nil || s.stopCh != nil {
		return // 无缓存或已在跑
	}

	stopCh := make(chan struct{})
	s.stopCh = stopCh

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("project 视图刷新 goroutine panic", "err", r)
			}
		}()

		failed := false // 上轮刷新是否失败（失败退避整周期）
		for {
			now := time.Now()
			staleness := max(now.Sub(s.scanCache.UpdatedAt()), now.Sub(s.projCache.UpdatedAt()))
			delay := max(maxExpireTime-staleness, 0)
			if failed {
				delay = maxExpireTime
			}
			// time.After 每轮新建：Timer.Reset 在已触发的 timer 上有未读陈旧值陷阱
			select {
			case <-time.After(delay):
				failed = s.refresh() != nil
			case <-stopCh:
				return
			}
		}
	}()
}

// OnServerStart 启动后台刷新（app 层钩子，仅常驻 server 调用）。
func (s *Service) OnServerStart() {
	s.StartRefreshTicker()
}

// OnServerStop 停止后台定时刷新 goroutine（app 层钩子，server shutdown 时调）。
func (s *Service) OnServerStop() { s.StopRefreshTicker() }

// StopRefreshTicker 停止定时刷新 goroutine（server shutdown 时调）。
// 未启用时调用安全（空操作）。
func (s *Service) StopRefreshTicker() {
	if s.stopCh == nil {
		return
	}
	close(s.stopCh)
	s.stopCh = nil
}

// refresh 完整刷新 project 视图（后台刷新循环调用）：重扫 + git 采集。
// 异常不抛出（降级优先）：失败记日志并返回错误，由调用方决定退避策略。
func (s *Service) refresh() error {
	total, collected, err := s.Refresh()
	if err != nil {
		slog.Warn("刷新 git 缓存失败", "err", err, "projects", total)
		return err
	}
	slog.Debug("project 视图刷新完成", "projects", total, "collected", collected)
	return nil
}

// Refresh 立即完整刷新 project 视图：重扫项目列表 → 整表采集 git 信息
// （写内存 + 落盘 git.json）。返回 (项目总数, 采集成功数, 错误)。
//
// 与 server 定时刷新同一逻辑。CLI 平时只读缓存不写（单写者模型：server 是唯一写方），
// 本方法仅供 dev 调试命令手动触发，用于开发期实测全量采集的时间成本。
func (s *Service) Refresh() (total int, collected int, err error) {
	projects := s.scanCache.Reload()

	paths := xslices.Map(projects, (*Project).Path)
	if s.projCache == nil {
		return len(paths), 0, errors.New("git 缓存未初始化")
	}
	if err := s.projCache.Refresh(paths); err != nil {
		return len(paths), 0, err
	}
	return len(paths), s.projCache.Size(), nil
}
