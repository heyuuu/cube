package git

import (
	"errors"
	"fmt"
	"slices"
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

// pickRemote 决定目标 remote：flag 指定优先；仅 1 个自动选中；多个时交互单选
// （push/pull 每次只面对一个 remote，remote 列表的收集也收敛在此）。
func pickRemote(repoPath string, flagRemote string) (git.Remote, error) {
	var zero git.Remote

	remotes, err := git.Remotes(repoPath)
	if err != nil {
		return zero, fmt.Errorf("读取 remote 列表失败: %w", err)
	}

	// 指定了 --remote 参数的情况
	if flagRemote != "" {
		index := slices.IndexFunc(remotes, func(r git.Remote) bool { return r.Name == flagRemote })
		if index < 0 {
			return zero, fmt.Errorf("未找到 remote: %s（可用: %s）", flagRemote, joinRemoteNames(remotes))
		}
		return remotes[index], nil
	}

	// 未指定 --remote 参数的情况
	if len(remotes) == 0 {
		return zero, errors.New("仓库未配置任何 remote，无任何可选远端")
	} else if len(remotes) == 1 {
		return remotes[0], nil
	}

	if !tui.IsTTY() {
		return zero, fmt.Errorf("非交互环境(tty)下必须通过 --remote 指定 remote（可选: %s）", joinRemoteNames(remotes))
	}
	return tui.SelectItem("选择远端 remote", remotes, func(r git.Remote) string {
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
