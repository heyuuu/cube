package project

import (
	"fmt"
	"os"
	"path/filepath"

	"cube/internal/domain/icons"
	"cube/pkg/xstore"

	"github.com/heyuuu/x/store"
)

// settings.json 中 project 域的两个规则节：分节存储，可独立读取与写入。
const (
	scanRulesSection  = "scanRules"
	cloneRulesSection = "cloneRules"
)

type ScanRulesStore = *xstore.KeyedSliceStore[ScanRule]
type CloneRulesStore = *xstore.KeyedSliceStore[CloneRule]

func newScanRulesStore(settingsFile string) ScanRulesStore {
	// ScanRules 的存储
	// - 使用 path 作为唯一标识
	// - Path 必须是绝对路径，且是已存在的文件夹
	// - Group 不允许重复
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[ScanRule]{
		Name:  "scan 规则",
		Store: store.SectionStore[[]ScanRule](settingsFile, scanRulesSection),
		KeyOf: func(r ScanRule) string { return r.Path },
		Normalize: func(r ScanRule) (ScanRule, error) {
			if r.Group == "" {
				return r, fmt.Errorf("scan 规则 group 不得为空")
			}
			if r.MaxDepth < -1 {
				return r, fmt.Errorf("scan 规则 maxDepth 不得小于 -1（-1 不限深、0 仅根下一层）: %d", r.MaxDepth)
			}
			if !filepath.IsAbs(r.Path) {
				return r, fmt.Errorf("scan 规则路径必须为绝对路径: path=%s", r.Path)
			}
			if info, err := os.Stat(r.Path); err != nil || !info.IsDir() {
				return r, fmt.Errorf("scan 规则路径不存在或非目录: %s", r.Path)
			}
			if err := icons.ValidateIcon(r.Icon); err != nil {
				return r, fmt.Errorf("scan 规则 %s", err)
			}
			r.Path = filepath.Clean(r.Path)
			r.Tags = normalizeTags(r.Tags)
			return r, nil
		},
		ConflictCheck: func(new ScanRule, exists ScanRule) error {
			// 判断 group 是否冲突
			if new.Group == exists.Group {
				return fmt.Errorf("scan 规则 group 已被其它规则占用: group=%s 已用于 %s", new.Group, new.Path)
			}
			return nil
		},
	})
}

func newCloneRulesStore(settingsFile string) CloneRulesStore {
	return xstore.NewKeyedSliceStore(xstore.KeyedSliceStoreOpts[CloneRule]{
		Name:  "clone 规则",
		Store: store.SectionStore[[]CloneRule](settingsFile, scanRulesSection),
		KeyOf: func(r CloneRule) string { return r.RepoHost + "/" + r.RepoPrefix },
		Normalize: func(r CloneRule) (CloneRule, error) {
			if r.RepoHost == "" {
				return r, fmt.Errorf("clone 规则 repoHost 不得为空")
			}
			if r.RepoPrefix != "" && r.RepoPrefix[0] != '/' {
				return r, fmt.Errorf("clone 规则 repoPrefix 须以 / 开头或为空: %s", r.RepoPrefix)
			}
			if !filepath.IsAbs(r.LocalPath) {
				return r, fmt.Errorf("clone 规则 localPath 不合法（须绝对路径）: %s", r.LocalPath)
			}

			r.LocalPath = filepath.Clean(r.LocalPath)
			return r, nil
		},
	})
}
