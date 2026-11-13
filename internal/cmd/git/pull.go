package git

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/pkg/git"
	"cube/pkg/tui"
)

// newPullCmd 是 `cube git pull` 命令入口（push 的反向：从单个 remote 批量更新本地分支）。
//
// 行为：path 定位 git 仓库（规则同 push），选择一个来源 remote
// （仅 1 个 remote 时自动选中，多个时交互单选），先 fetch 一次刷新跟踪分支
// （唯一网络往返），多选分支后逐条本地快进：
//   - 检出分支：git merge --ff-only <remote>/<branch>（在其检出所在的工作树执行）
//   - 未检出分支：git fetch . <remote>/<branch>:<branch>
//
// 仅快进：本地与远端分叉的分支失败跳过，绝不产生 merge commit。
// 单条失败不中断（同 push），最后汇总结果。
func newPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull [path]",
		Short: "从选中的 remote 批量快进更新本地分支",
		Long: `从单个 remote 批量拉取更新选中的分支（push 的反向命令）。

path 指定目标 git 仓库（绝对路径或以 . 开头的相对路径，裸短名不识别；
不传为当前目录；linked worktree 归并到主工作树根），不要求是 cube 已收录的项目。

remote 选择：仅 1 个 remote 时自动选中；多个 remote 时交互单选。
分支默认通过 TUI 多选交互确认：候选为与该 remote 同名的本地分支，
默认勾选远端领先的分支（都不落后时默认当前分支）。

执行模型：先对选中 remote 做一次 git fetch 刷新跟踪分支（唯一的网络
往返，领先/落后与分叉判定均基于刷新后的真实数据），分支更新全部本地完成：
  - 检出分支（含主工作树与各 linked worktree 检出的）：
    git merge --ff-only <remote>/<branch>，在其检出所在的工作树执行
  - 未检出分支：git fetch . <remote>/<branch>:<branch>
仅快进：分叉的分支失败跳过，不产生 merge commit。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !tui.IsTTY() {
				return errors.New("此命令只支持交互环境(tty)下使用")
			}

			err := pullFlow(xslices.Get(args, 0))
			if errors.Is(err, tui.ErrUserAborted) {
				return nil
			}
			return err
		},
	}
	return cmd
}

// pullFlow 拉取主流程：定位仓库 → 选 remote → fetch 一次刷新跟踪 → 候选/多选
// → 确认 → 逐条本地快进。交互取消（ErrUserAborted）上抛，由 RunE 统一转 nil。
func pullFlow(path string) error {
	// 定位 git 仓库主工作树根（worktree 归并）
	repoPath, err := resolveGitRoot(path)
	if err != nil {
		return err
	}

	// 选择目标 remote（单 remote 自动 → 多 remote 手选）
	remote, err := pickRemote(repoPath)
	if err != nil {
		return err
	}

	// 获取当前主目录分支
	currentBranch := git.CurrentBranch(repoPath)

	// 多选 ref（TUI 多选，默认当前分支；分支 ref 带该 remote 的真实 ahead/behind）
	refs, err := pickPullRefs(repoPath, remote)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return errors.New("未选择任何分支/tag")
	}

	// 展示拉取计划 + 二次确认
	ok, err := confirmPullPlan(repoPath, remote, refs, currentBranch)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// 执行：逐条本地快进并汇总结果
	return runPull(repoPath, refs)
}

// pickPullBranches 决定待拉取的分支集合 TUI 多选，默认勾选 pullDefaultBranches 的结果。
func pickPullRefs(repoPath string, remote git.Remote) ([]refInfo, error) {
	refInfos, err := allRefInfos(repoPath, remote.Name)
	if err != nil {
		return nil, err
	}

	// 默认展示的 pull 列表为 本地和远端都存在的分支 + 本地不存在且远端存在的 tag
	candidates := xslices.Filter(refInfos, func(ref refInfo) bool {
		if ref.IsLocalBranch() && ref.IsRemoteBranch() {
			return true
		} else if !ref.LocalExists() && ref.IsRemoteTag() {
			return true
		}
		return false
	})
	if len(candidates) == 0 {
		return nil, errors.New("仓库无任何待选推送分支或 tag")
	}

	// 补充 ref 额外信息用于展示
	candidates, err = resolveRefExtraInfos(repoPath, remote.Name, candidates)
	if err != nil {
		return nil, err
	}

	return tui.MultiSelectItem("选择待拉取的分支/tag（空格勾选，回车确认）", candidates, refLabel)
}

// confirmPullPlan 打印拉取计划表并要求二次确认。返回 (是否确认, 错误)。
func confirmPullPlan(repoPath string, remote git.Remote, refs []refInfo, currentBranch string) (bool, error) {
	fmt.Println()
	fmt.Printf("仓库: %s\n", repoPath)
	fmt.Printf("Remote: %s  (%s)\n", remote.Name, remote.Fetch)
	fmt.Println("拉取计划（仅快进，分叉的分支会失败跳过）：")
	tui.PrintTable([]string{"Ref", "方式"}, xslices.Map(refs, func(r refInfo) []string {
		return []string{
			r.name,
			pullMethodText(r.name, r.worktree, currentBranch),
		}
	}))
	fmt.Println()
	return tui.ConfirmInlineDefault(fmt.Sprintf("确认从 %s 拉取以上 %d 个分支/tag？", remote.Name, len(refs)), true)
}

// pullMethodText 返回分支在计划表中的更新方式：检出分支在其工作树 merge --ff-only，
// 未检出分支本地 fetch 快进。
func pullMethodText(branch string, checkoutAt string, currentBranch string) string {
	switch {
	case checkoutAt == "":
		return "fetch . 快进"
	case branch == currentBranch:
		return "merge --ff-only（当前分支）"
	default:
		return fmt.Sprintf("merge --ff-only（worktree: %s）", filepath.Base(checkoutAt))
	}
}

// runPull 逐条执行本地快进，best-effort：单条失败不中断后续（同 push），最后汇总。
// 检出分支在其检出的工作树 merge --ff-only（git 拒绝 fetch 更新任何检出的分支），
// 未检出分支从本地 remote-tracking fetch 快进。
func runPull(repoPath string, refs []refInfo) error {
	failed := 0
	for i, ref := range refs {
		fmt.Printf("\n[%d/%d] ", i+1, len(refs))
		var err error
		remote := ref.remote
		if ref.IsLocalBranch() {
			branch := ref.name
			if ref.worktree != "" {
				fmt.Printf("git -C %s merge --ff-only %s/%s\n", ref.worktree, remote, branch)
				err = git.MergeFfOnly(ref.worktree, remote+"/"+branch)
			} else {
				fmt.Printf("git fetch . %s/%s:%s\n", remote, branch, branch)
				err = git.FastForwardFromTracking(repoPath, remote, branch)
			}
		} else if ref.IsLocalTag() {
			tag := ref.name
			fmt.Printf("git fetch %s tag %s\n", remote, tag)
			err = git.FetchTag(ref.worktree, remote, tag)
		}

		if err != nil {
			failed++
			fmt.Printf("  ✗ 失败: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Printf("完成：%d 条成功，%d 条失败。\n", len(refs)-failed, failed)
	if failed > 0 {
		return fmt.Errorf("有 %d 条拉取失败，请检查上方输出", failed)
	}
	return nil
}
