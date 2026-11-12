package git

import (
	"reflect"
	"testing"

	"cube/pkg/git"
)

// TestBuildRemoteBranchMap 验证远程分支按短名组织成 map[branch]set[remote]，
// 含斜杠分支名（feature/fix-bug）在多 remote 下的归属。
func TestBuildRemoteBranchMap(t *testing.T) {
	m := buildRemoteBranchMap(git.BuildRefs([]string{
		"refs/remotes/origin/master",
		"refs/remotes/origin/feature/fix-bug",
		"refs/remotes/upstream/feature/fix-bug",
	}))

	want := map[string]map[string]bool{
		"master":          {"origin": true},
		"feature/fix-bug": {"origin": true, "upstream": true},
	}
	if !reflect.DeepEqual(m, want) {
		t.Fatalf("buildRemoteBranchMap = %v，期望 %v", m, want)
	}
}
