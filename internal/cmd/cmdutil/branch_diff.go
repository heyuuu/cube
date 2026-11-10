package cmdutil

// 本文件收敛「本地分支 × remote 同名分支」差距计算的共享助手，
// 供 info -v 的分支同步状态段与 pull 的候选分支计算复用。
// 口径：基于本地记录的 remote 跟踪分支（refs/remotes/*，不联网），
// 即上次 fetch 时的远端状态。

import (
	"sort"

	"cube/pkg/git"
)

// BuildRemoteBranchMap 把远程分支列表组织成 map[branch]set[remote]，便于 O(1) 查询
// 「某 remote 是否有某分支」。
func BuildRemoteBranchMap(refs *git.RefsResult) map[string]map[string]bool {
	m := make(map[string]map[string]bool)
	for _, ref := range refs.Remotes {
		if m[ref.Branch] == nil {
			m[ref.Branch] = make(map[string]bool)
		}
		m[ref.Branch][ref.Remote] = true
	}
	return m
}

// PickSharedBranches 取「本地分支 ∩ 任一 remote 同名分支」的并集，结果按名排序。
// 本地有但所有 remote 都没有的分支被排除（无对比对象）。
func PickSharedBranches(refs *git.RefsResult, remoteHasBranch map[string]map[string]bool) []string {
	var shared []string
	for _, ref := range refs.Locals {
		if len(remoteHasBranch[ref.Branch]) > 0 {
			shared = append(shared, ref.Branch)
		}
	}
	sort.Strings(shared)
	return shared
}
