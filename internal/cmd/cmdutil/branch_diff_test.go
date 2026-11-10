package cmdutil

import (
	"reflect"
	"testing"

	"cube/pkg/git"
)

// TestBuildRemoteBranchMap 验证远程分支按短名组织成 map[branch]set[remote]，
// 含斜杠分支名（feature/fix-bug）在多 remote 下的归属。
func TestBuildRemoteBranchMap(t *testing.T) {
	m := BuildRemoteBranchMap(git.BuildRefs([]string{
		"refs/remotes/origin/master",
		"refs/remotes/origin/feature/fix-bug",
		"refs/remotes/upstream/feature/fix-bug",
	}))

	want := map[string]map[string]bool{
		"master":          {"origin": true},
		"feature/fix-bug": {"origin": true, "upstream": true},
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("BuildRemoteBranchMap = %v，期望 %v", m, want)
	}
}

// TestPickSharedBranches 验证交集选取：斜杠本地分支正确命中同名远程分支，
// 所有 remote 都没有的本地分支被排除，结果按名排序。
func TestPickSharedBranches(t *testing.T) {
	refs := git.BuildRefs([]string{
		git.RefHeadsPrefix + "feature/fix-bug",
		git.RefHeadsPrefix + "master",
		git.RefHeadsPrefix + "local-only",
	})
	m := map[string]map[string]bool{
		"master":          {"origin": true},
		"feature/fix-bug": {"origin": true, "upstream": true},
	}

	got := PickSharedBranches(refs, m)
	want := []string{"feature/fix-bug", "master"} // local-only 无远程同名分支，排除
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PickSharedBranches = %v，期望 %v", got, want)
	}
}
