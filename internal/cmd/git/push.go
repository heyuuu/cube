package git

import (
	"errors"
	"fmt"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/pkg/git"
	"cube/pkg/tui"
)

// newPushCmd 是 `cube git push` 命令入口。
//
// 行为：path 定位 git 仓库，选择单个目标 remote，把多个分支 / tag 批量推送。
// ref 通过 TUI 多选交互确认，当前分支默认勾选，分支标签带相对该 remote 的真实 ahead/behind（选择前 fetch 刷新，失败降级不显示）。
// 最后展示推送计划并二次确认后执行。
func newPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push [path]",
		Short: "把选中的分支/tag 批量推送到选中的 remote",
		Long: `把多个分支 / tag 批量推送到选中的单个 remote。

path 指定目标 git 仓库（绝对路径或以 . 开头的相对路径，裸短名不识别；不传为当前目录；linked worktree 归并到主工作树根），不要求是 cube 已收录的项目。

remote 选择：仅 1 个 remote 时自动选中；多个 remote 时交互单，一次只推一个 remote；要推多个 remote 分次执行。
ref 默认通过 TUI 多选交互确认：默认勾选当前分支；分支标签带相对该 remote 的真实 ahead/behind（选择前自动 fetch 刷新，失败降级为不显示）。
执行前展示推送计划并二次确认。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !tui.IsTTY() {
				return errors.New("此命令只支持交互环境(tty)下使用")
			}

			err := pushFlow(xslices.Get(args, 0))
			if errors.Is(err, tui.ErrUserAborted) {
				return nil
			}
			return err
		},
	}
	return cmd
}

// pushFlow 推送主流程：定位仓库 → 收集候选 → 选 remote/ref → 确认 → 逐条推送。
// 交互取消（ErrUserAborted）上抛，由 RunE 统一转 nil。
func pushFlow(path string) error {
	// 定位 git 仓库主工作树根（worktree 归并）
	repoPath, err := resolveGitRoot(path)
	if err != nil {
		return err
	}

	// 收集 refs 候选
	headRef := git.HeadRef(repoPath)
	repoRefs, err := git.Refs(repoPath)
	if err != nil {
		return fmt.Errorf("读取 ref 列表失败: %w", err)
	}

	// 选择目标 remote（同 pull：flag 优先 → 单 remote 自动 → 多 remote 手选）
	chosenRemote, err := pickRemote(repoPath)
	if err != nil {
		return err
	}

	// 选择 ref（flag 优先，否则 TUI 多选，默认当前分支；标签带该 remote 的真实 ahead/behind）
	chosenRefs, err := pickRefs(repoPath, repoRefs, headRef, chosenRemote)
	if err != nil {
		return err
	}
	if len(chosenRefs) == 0 {
		return errors.New("未选择任何分支/tag")
	}

	// 展示推送计划 + 二次确认
	ok, err := confirmPlan(repoPath, chosenRemote, chosenRefs)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// 执行：逐条 push 并汇总结果
	return runPush(repoPath, chosenRemote, chosenRefs)
}

// refAllTags 候选哨兵值：代表一次性推送全部本地 tag（git push --tags），非真实 git ref。
const refAllTags = "--tags"

// pickRefs 决定要推送的 ref 集合：flag 显式指定优先（校验存在于本地，重复去重），
// 否则 TUI 多选（默认当前分支）。
//   - 候选 = 本地分支 + 一条「全部 tags」合并项（tag 数量随时间线性增长，逐条列出
//     只会让候选列表越来越长，故合并成 refAllTags 一项整体推送）。
//   - 默认勾选当前分支（若存在）。
//   - 分支标签带相对该 remote 的真实 ahead/behind（先 fetch 刷新 remote-tracking，
//     显示 "↑2 ↓0"）；tag 无此语义。fetch 失败（网络不通等）打警告降级为不显示，
//     不阻断选择。
func pickRefs(repoPath string, refs *git.RefsResult, headRef string, remote git.Remote) ([]string, error) {
	if err := doFetch(repoPath, remote); err != nil {
		return nil, err
	}

	// fetch 后重读 refs，拿到最新 remote-tracking；远端没有同名分支的显示 ↑0 ↓0 会误导（像是已同步），跳过
	remoteBranches := map[string]map[string]bool{}
	if fresh, err := git.Refs(repoPath); err == nil {
		remoteBranches = buildRemoteBranchMap(fresh)
	}

	type refItem struct {
		label string
		ref   string
	}
	var items []refItem
	for _, ref := range refs.Locals {
		items = append(items, refItem{label: "branch: " + ref.ShortName + aheadBehindSuffix(repoPath, ref.ShortName, remote, remoteBranches), ref: ref.Name})
	}
	if len(refs.Tags) > 0 {
		items = append(items, refItem{
			label: fmt.Sprintf("tags:   全部 %d 个", len(refs.Tags)),
			ref:   refAllTags,
		})
	}
	if len(items) == 0 {
		return nil, errors.New("仓库无任何本地分支或 tag")
	}

	// 默认勾选当前分支（匹配 ref == currentBranch）
	var defaults []refItem
	if headRef != "" {
		for _, it := range items {
			if it.ref == headRef {
				defaults = append(defaults, it)
				break
			}
		}
	}

	picked, err := tui.MultiSelectItemWithDefaults(
		"选择待推送的分支/tag（空格勾选，回车确认）",
		items,
		func(it refItem) string { return it.label },
		defaults,
	)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(picked))
	for i, it := range picked {
		result[i] = it.ref
	}
	return result, nil
}

// aheadBehindSuffix 生成分支标签的 ahead/behind 后缀（真实值，依赖调用方已 fetch）。
// 远端没有同名分支、或 fetch 失败时返回空串——显示 ↑0 ↓0 会误导为已同步。
func aheadBehindSuffix(repoPath, branch string, remote git.Remote, remoteBranches map[string]map[string]bool) string {
	if !remoteBranches[branch][remote.Name] {
		return ""
	}
	ahead, behind, _ := git.AheadBehindRemote(repoPath, branch, remote.Name, branch)
	return fmt.Sprintf("  ↑%d ↓%d", ahead, behind)
}

// confirmPlan 打印推送计划表并要求二次确认。返回 (是否确认, 错误)。
func confirmPlan(repoPath string, remote git.Remote, refs []string) (bool, error) {
	fmt.Println()
	fmt.Printf("仓库: %s\n", repoPath)
	fmt.Printf("Remote: %s  (%s)\n", remote.Name, remote.Push)
	fmt.Println("推送计划：")
	rows := make([][]string, 0, len(refs))
	for _, ref := range refs {
		rows = append(rows, []string{refDisplayName(ref)})
	}
	tui.PrintTable([]string{"Ref"}, rows)
	fmt.Println()
	return tui.ConfirmInlineDefault(fmt.Sprintf("确认向 %s 推送以上 %d 个 ref？", remote.Name, len(refs)), true)
}

// refDisplayName 返回 ref 在计划表中的展示名；哨兵 refAllTags 展示为更友好的说明。
func refDisplayName(ref string) string {
	if ref == refAllTags {
		return "全部本地 tag (--tags)"
	}
	return ref
}

// runPush 逐条执行 git push 到选中的 remote，并打印汇总。
//
// 单条 push 失败不中断后续，全部跑完后若有失败则汇总返回一个 error。
func runPush(repoPath string, remote git.Remote, refs []string) error {
	failed := 0
	for i, ref := range refs {
		fmt.Printf("\n[%d/%d] git push %s %s\n", i+1, len(refs), remote.Name, refDisplayName(ref))
		var err error
		if ref == refAllTags {
			err = git.PushAllTags(repoPath, remote.Name, false)
		} else {
			err = git.Push(repoPath, remote.Name, ref, false)
		}
		if err != nil {
			failed++
			fmt.Printf("  ✗ 失败: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Printf("完成：%d 条成功，%d 条失败。\n", len(refs)-failed, failed)
	if failed > 0 {
		return fmt.Errorf("有 %d 条 push 失败，请检查上方输出", failed)
	}
	return nil
}
