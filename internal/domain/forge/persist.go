package forge

import (
	"log/slog"
	"sync"
	"time"

	"github.com/heyuuu/x/store"

	"cube/pkg/gitapi"
)

// account 拉取结果的持久化形态（cache/forge-repos.json，提案 1042，1044 改按 account 键）。
// 纯缓存语义：可整体删除且行为不变差（重拉即重建）；落盘是为了 forge 页在 server
// 重启后仍有数据可看——拉取是显式动作，重启即空会让「数据过期而不自知」。
// fetchedAt 随数据落盘，前端据此展示新鲜度。

// AcctCacheEntry 单个 account 的拉取结果快照。
type AcctCacheEntry struct {
	FetchedAt time.Time           `json:"fetchedAt"` // 最近一次成功拉取时间（零值不出现：只有成功才落盘）
	Repos     []gitapi.RemoteRepo `json:"repos"`     // 该账号名下的远端仓库全量
}

// acctDiskCache 落盘文档形态：accountKey → 快照。
type acctDiskCache struct {
	Accounts map[string]AcctCacheEntry `json:"accounts"`
}

// repoCache account 拉取结果缓存：内存为一级，写穿落盘。常驻 server 是唯一写方，
// 内存即落盘的完整镜像，写入直接落内存快照、不回读合并。首次访问懒加载落盘文件，
// 文件缺失 / 坏 JSON 降级为空缓存（读路径不阻断）。
type repoCache struct {
	store store.Store[acctDiskCache]

	mu      sync.Mutex
	entries map[string]AcctCacheEntry // nil = 尚未从落盘加载
}

func newRepoCache(file string) *repoCache {
	return &repoCache{store: store.JSONStore[acctDiskCache](file)}
}

func (c *repoCache) get(key string) (AcctCacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureLoadedLocked()
	entry, ok := c.entries[key]
	return entry, ok
}

// put 写入一个 account 的快照并写穿落盘。
func (c *repoCache) put(key string, entry AcctCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureLoadedLocked()

	c.entries[key] = entry
	c.saveLocked()
}

// retain 只保留 keep 判定为真的条目，有删动才落盘。
func (c *repoCache) retain(keep func(key string) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ensureLoadedLocked()

	removed := false
	for k := range c.entries {
		if !keep(k) {
			delete(c.entries, k)
			removed = true
		}
	}
	if removed {
		c.saveLocked()
	}
}

// saveLocked 落盘失败只记日志：缓存语义，下次写入会重写。
func (c *repoCache) saveLocked() {
	if err := c.store.Save(acctDiskCache{Accounts: c.entries}); err != nil {
		slog.Warn("forge 拉取缓存落盘失败", "err", err)
	}
}

func (c *repoCache) ensureLoadedLocked() {
	if c.entries != nil {
		return
	}

	c.entries = c.store.LoadOrZero().Accounts
	if c.entries == nil {
		c.entries = map[string]AcctCacheEntry{}
	}
}
