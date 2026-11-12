package git

import (
	"fmt"
	"strings"

	"cube/internal/cmd/cmdutil"
	"cube/pkg/git"
	"cube/pkg/tui"
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

// resolveRemoteByName 把 flag 传入的 remote 名解析成 git.Remote（找不到则报错）。
func resolveRemoteByName(all []git.Remote, name string) (git.Remote, error) {
	byName := make(map[string]git.Remote, len(all))
	for _, r := range all {
		byName[r.Name] = r
	}
	r, ok := byName[name]
	if !ok {
		return git.Remote{}, fmt.Errorf("未找到 remote: %s（可用: %s）", name, joinRemoteNames(all))
	}
	return r, nil
}

// pickRemote 决定目标 remote：flag 指定优先；仅 1 个自动选中；多个时交互单选
// （push/pull 每次只面对一个 remote，title 传入场景文案）。
func pickRemote(all []git.Remote, flagRemote string, title string) (git.Remote, error) {
	if flagRemote != "" {
		return resolveRemoteByName(all, flagRemote)
	}
	if len(all) == 1 {
		return all[0], nil
	}
	if !tui.IsTTY() {
		return git.Remote{}, fmt.Errorf("非交互环境(tty)下必须通过 --remote 指定 remote（可选: %s）", joinRemoteNames(all))
	}
	return tui.SelectItem(title, all, func(r git.Remote) string {
		return fmt.Sprintf("%s  (%s)", r.Name, r.Fetch)
	})
}

// joinRemoteNames 把 remote 名列表拼成逗号分隔串，用于错误提示。
func joinRemoteNames(remotes []git.Remote) string {
	names := make([]string, len(remotes))
	for i, r := range remotes {
		names[i] = r.Name
	}
	return strings.Join(names, ", ")
}
