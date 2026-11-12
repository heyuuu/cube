package git

import (
	"fmt"
	"strings"

	"github.com/heyuuu/x/xstd/xslices"

	"cube/internal/cmd/cmdutil"
	"cube/pkg/git"
)

// resolveGitRoot 把 path（缺省 "."）解析为 git 仓库根目录：
// linked worktree 归并到主工作树根；不要求是 cube 已收录项目。
func resolveGitRoot(path string) (string, error) {
	if path == "" {
		path = "."
	}

	absPath, err := cmdutil.AsPathArg(path)
	if err != nil {
		return "", err
	}

	root, ok := git.FindMainRoot(absPath)
	if !ok {
		return "", fmt.Errorf("目录不在 git 仓库中: %s", absPath)
	}

	return root, nil
}

// resolveRemotesByName 把 flag 传入的 remote 名解析成 git.Remote
// （找不到则报错；重名去重，避免 -r x -r x 同一 remote 重复执行）。
func resolveRemotesByName(all []git.Remote, names []string) ([]git.Remote, error) {
	byName := make(map[string]git.Remote, len(all))
	for _, r := range all {
		byName[r.Name] = r
	}
	var result []git.Remote
	for _, n := range xslices.Unique(names) {
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
