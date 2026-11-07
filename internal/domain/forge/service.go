package forge

import (
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"cube/internal/config"
	"cube/pkg/gitapi"
)

// clientFactory 构造平台 API 客户端的工厂签名（测试注入 fake 用）。
type clientFactory func(kind, host, token string) (gitapi.Client, error)

// Service forge 配置管理：settings.json forges / forgeAccounts 两节的读写（直读不缓存，
// 保存即生效），以及 account 远端拉取的编排（结果按 accountKey 缓存，见 persist.go）。
// 1044 起 namespace 模型移除，拉取以 account 为唯一维度。
type Service struct {
	// settings
	forgesStore  ForgesStore
	accountStore AccountStore
	// 拉取结果缓存（内存 + 落盘 cache/forge-repos.json）
	repoCache *repoCache

	newClient clientFactory // 可注入（测试）；nil 用 gitapi.NewClient
}

// NewService 由 Config 装配 Service（DI 注入用），拉取缓存落 cache/forge-repos.json。
func NewService(cfg *config.Config) *Service {
	forgesStore := newForgesStore(cfg.SettingsFile())
	accountsStore := newAccountsStore(cfg.SettingsFile(), forgesStore)

	return &Service{
		forgesStore:  forgesStore,
		accountStore: accountsStore,
		repoCache:    newRepoCache(filepath.Join(cfg.CacheDir(), "forge-repos.json")),
	}
}

// Forges 读全部 forge（直读不缓存）。坏条目（host 空 / kind 未知 / icon 非法）跳过不阻断。
func (s *Service) Forges() []Forge {
	return s.forgesStore.All()
}

// SaveForge 新增或按 host 替换一条 forge（host 归一化后是唯一键——
// 编辑 host 等价于删旧存新，前端按此语义提交）。校验收敛在写侧，坏数据中文错误不落文件。
func (s *Service) SaveForge(f Forge) error {
	return s.forgesStore.Save(f)
}

// SaveAllForges 以提交清单整节覆盖 forges 节（顺序即展示序，1042 forge 页沿用）。
// 逐项校验 + host 查重，任一失败整体不落盘；被移除 forge 下的 account 级联清理（同 DeleteForge）。
func (s *Service) SaveAllForges(forges []Forge) error {
	if err := s.forgesStore.SaveAll(forges); err != nil {
		return err
	}

	return s.purgeAccounts()
}

// DeleteForge 按唯一键（归一化 host）删除一条 forge，并级联清理该 host 下的 account
// （孤挂配置对任何功能都不可见，留了就是脏数据）；不存在时返回中文错误。
func (s *Service) DeleteForge(f Forge) error {
	if err := s.forgesStore.Delete(f); err != nil {
		return err
	}

	return s.purgeAccounts()
}

// --- account（1041，1044 起是 forge 下唯一的关联配置） ---

// Accounts 读全部 account（含 token 原文，供拉取取凭证；对外展示的打码在出口层做）。
func (s *Service) Accounts() []Account { return s.accountStore.All() }

// SaveAccount 新增或按 forgeHost+username 替换一条 account。
// token 提交掩码值（TokenMasked）视为「未修改」沿用旧值——前端表单掩码展示的配套语义。
func (s *Service) SaveAccount(a Account) error {
	return s.accountStore.Save(a)
}

// SaveAllAccounts 以提交清单整节覆盖 accounts 节（顺序即 forge 页展示序）。
// 逐项校验（含掩码 token 按唯一键回填旧值）+ 唯一键查重，任一失败整体不落盘。
func (s *Service) SaveAllAccounts(accounts []Account) error {
	return s.accountStore.SaveAll(accounts)
}

// DeleteAccount 按唯一键（forgeHost+username）删除一条 account；不存在时返回中文错误。
func (s *Service) DeleteAccount(a Account) error {
	return s.accountStore.Delete(a)
}

// --- 拉取与对账（account 维度） ---

// FetchAccount 拉取 account 名下的全部远端仓库（force=true 跳过缓存强制重拉）。
// 出站 API 调用是慢操作，手动/低频触发；成功结果写穿内存缓存与落盘文件（含 fetchedAt），
// 失败直接上抛（调用方展示错误，不影响既有缓存）。
func (s *Service) FetchAccount(forgeHost, username string, force bool) ([]gitapi.RemoteRepo, error) {
	key := accountKey(forgeHost, username)
	accounts := s.Accounts()
	i := slices.IndexFunc(accounts, func(a Account) bool { return accountKey(a.ForgeHost, a.Username) == key })
	if i < 0 {
		return nil, fmt.Errorf("未找到指定 account: %s", key)
	}
	a := accounts[i]

	if !force {
		if entry, ok := s.repoCache.get(key); ok {
			return entry.Repos, nil
		}
	}

	f, ok := s.forgesStore.Find(a.ForgeHost)
	if !ok {
		return nil, fmt.Errorf("account 所属 forge 未配置: %s", a.ForgeHost)
	}
	repos, err := fetchRepos(f, a, s.factory())
	if err != nil {
		return nil, err
	}

	live := make(map[string]bool, len(accounts))
	for _, a := range accounts {
		live[accountKey(a.ForgeHost, a.Username)] = true
	}
	s.repoCache.put(key, AcctCacheEntry{FetchedAt: time.Now(), Repos: repos}, live)
	return repos, nil
}

// Overview 聚合全部 account 的对账行与拉取元信息（forge 页数据源，1042/1044）。
// 只读既有缓存，不触发外呼；未拉取的 account 在 Accounts 元信息中 FetchedAt 为零值。
func (s *Service) Overview(local []LocalRepo) *Overview {
	return buildOverview(s.Forges(), s.Accounts(), func(a Account) (AcctCacheEntry, bool) {
		return s.repoCache.get(accountKey(a.ForgeHost, a.Username))
	}, local)
}

// --- 私有辅助 ---

func (s *Service) purgeAccounts() error {
	existHosts := make(map[string]bool)
	for _, f := range s.Forges() {
		existHosts[f.Host] = true
	}

	return s.accountStore.Retain(func(a Account) bool {
		return existHosts[a.ForgeHost]
	})
}

// factory 返回客户端工厂（未注入时用真实实现，httpClient 默认）。
func (s *Service) factory() clientFactory {
	if s.newClient != nil {
		return s.newClient
	}
	return func(kind, host, token string) (gitapi.Client, error) {
		return gitapi.NewClient(kind, host, token, nil)
	}
}
