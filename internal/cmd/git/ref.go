package git

import (
	"fmt"

	"github.com/heyuuu/x/xstd/xmaps"

	"cube/pkg/git"
)

// RefKind ref 种类
type RefKind uint8

const (
	RefNotExists RefKind = iota
	RefBranch
	RefTag
)

func (k RefKind) Exists() bool   { return k != RefNotExists }
func (k RefKind) IsBranch() bool { return k == RefBranch }
func (k RefKind) IsTag() bool    { return k == RefTag }

// refInfo ref 信息，包含本地和远端信息
type refInfo struct {
	// 基础信息
	name       string  // ref 短名(即 branch 或 tag 名)
	localKind  RefKind // 本地 ref 种类
	remoteKind RefKind // 远端 ref 种类
	// 额外信息，不一定有值
	ahead    int
	behind   int
	worktree string
}

// allRefInfos 构建所有 refInfo
// - 基于本地 branch / tag 不可能同名，远端 branch / tag 不可能同名，但本地和远端可能出现 branch 和 tag 同名的假设，
// 输出的 []refInfo 不会有 name 相同的 refInfo，但是 localKind 和 remoteKind 可能分别是 RefBranch 和 RefTag，需要注意
// - 本函数只构建基础信息，额外信息不一定有值；如需额外信息，需要调用 resolveRefExtraInfos 补充额外信息
func allRefInfos(repoPath string, remote git.Remote) ([]refInfo, error) {
	refs, err := git.Refs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("读取 ref 列表失败: %w", err)
	}

	remoteTags, err := git.RemoteTags(repoPath, remote.Name)
	if err != nil {
		return nil, fmt.Errorf("读取远程 ref 列表失败: %w", err)
	}

	// 构建 ref 名列表，各 ref 本地、远端的 kind 映射
	maxSize := len(refs.Locals) + len(refs.Remotes) + len(refs.Tags) + len(remoteTags)
	refNames := make(map[string]struct{}, maxSize)
	localKinds := make(map[string]RefKind, maxSize)
	remoteKinds := make(map[string]RefKind, maxSize)

	for _, ref := range refs.Locals {
		refNames[ref.ShortName] = struct{}{}
		localKinds[ref.ShortName] = RefBranch
	}
	for _, ref := range refs.Tags {
		refNames[ref.ShortName] = struct{}{}
		localKinds[ref.ShortName] = RefTag
	}
	for _, ref := range refs.Remotes {
		refNames[ref.ShortName] = struct{}{}
		remoteKinds[ref.ShortName] = RefBranch
	}
	for _, tag := range remoteTags {
		refNames[tag.Name] = struct{}{}
		remoteKinds[tag.Name] = RefTag
	}

	// 构建数据数组
	refInfos := make([]refInfo, 0, len(refNames))
	for _, name := range xmaps.SortedKeys(refNames) {
		refInfos = append(refInfos, refInfo{
			name:       name,
			localKind:  localKinds[name],
			remoteKind: remoteKinds[name],
		})
	}
	return refInfos, nil
}

// resolveRefExtraInfos 完善 refInfos 的额外信息
func resolveRefExtraInfos(repoPath string, remote git.Remote, refInfos []refInfo) {
	panic("todo")
}
