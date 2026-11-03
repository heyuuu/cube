package git

import (
	"fmt"
	"strings"

	"cube/pkg/git"
)

// resolveRemotesByName 把 flag 传入的 remote 名解析成 git.Remote（找不到则报错）。
func resolveRemotesByName(all []git.Remote, names []string) ([]git.Remote, error) {
	byName := make(map[string]git.Remote, len(all))
	for _, r := range all {
		byName[r.Name] = r
	}
	var result []git.Remote
	for _, n := range names {
		r, ok := byName[n]
		if !ok {
			return nil, fmt.Errorf("未找到 remote: %s（可用: %s）", n, joinRemoteNames(all))
		}
		result = append(result, r)
	}
	return result, nil
}

// joinRemoteNames 把 remote 名列表拼成逗号分隔串，用于错误提示。
func joinRemoteNames(remotes []git.Remote) string {
	names := make([]string, len(remotes))
	for i, r := range remotes {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}
