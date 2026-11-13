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

	// 单选来源 remote（flag 优先 → 单 remote 自动 → 多 remote 手选，与 push 共用）
	chosenRemote, err := pickRemote(repoPath)
	if err != nil {
		return err
	}

	// 先 fetch 一次刷新 remote-tracking：候选的领先/落后与后续本地快进都以它为
	// 数据源（唯一网络往返）；失败即终止——没有新数据的拉取没有意义
	if err = doFetch(repoPath, chosenRemote); err != nil {
		return err
	}

	// 候选分支：与该 remote 同名的本地分支及各自领先/落后数、检出位置
	currentBranch := git.CurrentBranch(repoPath)
	candidates, err := pullCandidates(repoPath, chosenRemote.Name)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return fmt.Errorf("remote %s 上没有与本地同名的分支，无可拉取项", chosenRemote.Name)
	}

	// 多选分支（flag 优先，默认勾选远端领先的分支）
	chosenBranches, err := pickPullBranches(candidates, currentBranch)
	if err != nil {
		return err
	}
	if len(chosenBranches) == 0 {
		return errors.New("未选择任何分支")
	}

	// 展示拉取计划 + 二次确认
	ok, err := confirmPullPlan(repoPath, chosenRemote, chosenBranches, candidates, currentBranch)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// 执行：逐条本地快进并汇总结果
	return runPull(repoPath, chosenRemote.Name, chosenBranches, candidates)
}

// branchDiff 描述一个本地分支相对某 remote 同名分支的领先/落后 commit 数，
// 及其检出位置——决定更新路由（检出分支 git 拒绝被 fetch 更新，须走其工作树 merge）。
type branchDiff struct {
	Branch       string
	Ahead        int    // 本地领先
	Behind       int    // 本地落后（远端有更新）
	CheckedOutAt string // 检出该分支的工作树绝对路径（含主工作树）；空串 = 未检出
}

// pullCandidates 计算与 remoteName 同名的本地分支列表及各自 ahead/behind 与检出位置。
// ahead/behind 基于调用方刚 Fetch 刷新过的 remote-tracking，是真实数据。
func pullCandidates(repoPath string, remoteName string) ([]branchDiff, error) {
	repoRefs, err := git.Refs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("读取 ref 列表失败: %w", err)
	}
	wts, err := git.WorktreeList(repoPath)
	if err != nil {
		return nil, fmt.Errorf("枚举工作树失败: %w", err)
	}
	checkedOut := make(map[string]string, len(wts))
	for _, wt := range wts {
		if wt.Branch != "" {
			checkedOut[wt.Branch] = wt.Path
		}
	}

	// 候选 = 与指定 remote 同名的本地分支（其它 remote 的同名分支不参与）
	remoteHasBranch := buildRemoteBranchMap(repoRefs)
	var diffs []branchDiff
	for _, ref := range repoRefs.Locals {
		b := ref.Branch
		if !remoteHasBranch[b][remoteName] {
			continue
		}
		ahead, behind, _ := git.AheadBehindRemote(repoPath, b, remoteName, b)
		diffs = append(diffs, branchDiff{Branch: b, Ahead: ahead, Behind: behind, CheckedOutAt: checkedOut[b]})
	}
	return diffs, nil
}

// pickPullBranches 决定待拉取的分支集合：flag 显式指定优先（校验属于候选，重复去重），
// 否则 TUI 多选，默认勾选 pullDefaultBranches 的结果。
func pickPullBranches(candidates []branchDiff, currentBranch string) ([]string, error) {
	picked, err := tui.MultiSelectItemWithDefaults(
		"选择待拉取的分支（空格勾选，回车确认）",
		candidates,
		func(d branchDiff) string { return pullBranchLabel(d, currentBranch) },
		pullDefaultBranches(candidates, currentBranch),
	)
	if err != nil {
		return nil, err
	}
	return xslices.Map(picked, func(d branchDiff) string { return d.Branch }), nil
}

// pullDefaultBranches 计算多选列表的默认勾选项：远端领先的分支；
// 都不落后时退回当前分支（拉取最常见的就是更新当前分支）。
func pullDefaultBranches(candidates []branchDiff, currentBranch string) []branchDiff {
	var defaults []branchDiff
	for _, d := range candidates {
		if d.Behind > 0 {
			defaults = append(defaults, d)
		}
	}
	if len(defaults) == 0 && currentBranch != "" {
		for _, d := range candidates {
			if d.Branch == currentBranch {
				defaults = append(defaults, d)
				break
			}
		}
	}
	return defaults
}

// pullBranchLabel 构造分支在多选列表里的展示文案：分支名 + 检出标记 + 同步状态。
// * = 主工作树当前分支；⎇ = 检出于其它工作树（与前端 worktree 徽标同形）。
func pullBranchLabel(d branchDiff, currentBranch string) string {
	name := d.Branch
	switch {
	case d.Branch == currentBranch:
		name = "* " + name
	case d.CheckedOutAt != "":
		name = "⎇ " + name
	}
	var state string
	switch {
	case d.Ahead > 0 && d.Behind > 0:
		state = fmt.Sprintf("分叉：本地领先 %d / 落后 %d", d.Ahead, d.Behind)
	case d.Behind > 0:
		state = fmt.Sprintf("落后 %d", d.Behind)
	case d.Ahead > 0:
		state = fmt.Sprintf("本地领先 %d", d.Ahead)
	default:
		state = "已同步"
	}
	return fmt.Sprintf("branch: %s  (%s)", name, state)
}

// confirmPullPlan 打印拉取计划表并要求二次确认。返回 (是否确认, 错误)。
func confirmPullPlan(repoPath string, remote git.Remote, branches []string, candidates []branchDiff, currentBranch string) (bool, error) {
	checkoutAt := branchCheckoutMap(candidates)

	fmt.Println()
	fmt.Printf("仓库: %s\n", repoPath)
	fmt.Printf("Remote: %s  (%s)\n", remote.Name, remote.Fetch)
	fmt.Println("拉取计划（仅快进，分叉的分支会失败跳过）：")
	rows := make([][]string, 0, len(branches))
	for _, b := range branches {
		rows = append(rows, []string{b, pullMethodText(b, checkoutAt[b], currentBranch)})
	}
	tui.PrintTable([]string{"Ref", "方式"}, rows)
	fmt.Println()
	return tui.ConfirmInlineDefault(fmt.Sprintf("确认从 %s 拉取以上 %d 个分支？", remote.Name, len(branches)), true)
}

// branchCheckoutMap 从候选构建 分支→检出工作树路径 映射。
func branchCheckoutMap(candidates []branchDiff) map[string]string {
	return xslices.ToMap(candidates, func(d branchDiff) (string, string) {
		return d.Branch, d.CheckedOutAt
	})
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
func runPull(repoPath string, remoteName string, branches []string, candidates []branchDiff) error {
	checkoutAt := branchCheckoutMap(candidates)

	failed := 0
	for i, b := range branches {
		fmt.Printf("\n[%d/%d] ", i+1, len(branches))
		var err error
		if wt := checkoutAt[b]; wt != "" {
			fmt.Printf("git -C %s merge --ff-only %s/%s\n", wt, remoteName, b)
			err = git.MergeFfOnly(wt, remoteName+"/"+b)
		} else {
			fmt.Printf("git fetch . %s/%s:%s\n", remoteName, b, b)
			err = git.FastForwardFromTracking(repoPath, remoteName, b)
		}
		if err != nil {
			failed++
			fmt.Printf("  ✗ 失败: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Printf("完成：%d 条成功，%d 条失败。\n", len(branches)-failed, failed)
	if failed > 0 {
		return fmt.Errorf("有 %d 条拉取失败，请检查上方输出", failed)
	}
	return nil
}
