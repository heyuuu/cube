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

	// 选择目标 remote（单 remote 自动 → 多 remote 手选）
	remote, err := pickRemote(repoPath)
	if err != nil {
		return err
	}

	// 多选 ref（TUI 多选，默认当前分支；分支 ref 带该 remote 的真实 ahead/behind）
	refs, err := pickPushRefs(repoPath, remote)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return errors.New("未选择任何分支/tag")
	}

	// 展示推送计划 + 二次确认
	ok, err := confirmPushPlan(repoPath, remote, refs)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// 执行：逐条 push 并汇总结果
	return runPush(repoPath, refs)
}

// pickPushRefs 决定要推送的 ref 集合: TUI 多选（默认当前分支）。
func pickPushRefs(repoPath string, remote git.Remote) ([]refInfo, error) {
	refInfos, err := allRefInfos(repoPath, remote.Name)
	if err != nil {
		return nil, err
	}

	// 默认展示的 push 列表为 本地和远端都存在的分支 + 本地存在远端不存在的 tag
	candidates := xslices.Filter(refInfos, func(ref refInfo) bool {
		if ref.IsLocalBranch() && ref.IsRemoteBranch() {
			return true
		} else if ref.IsLocalTag() && !ref.RemoteExists() {
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

	return tui.MultiSelectItem("选择待推送的分支/tag（空格勾选，回车确认）", candidates, refLabel)
}

// confirmPushPlan 打印推送计划表并要求二次确认。返回 (是否确认, 错误)。
func confirmPushPlan(repoPath string, remote git.Remote, refs []refInfo) (bool, error) {
	fmt.Println()
	fmt.Printf("仓库: %s\n", repoPath)
	fmt.Printf("Remote: %s  (%s)\n", remote.Name, remote.Push)
	fmt.Println("推送计划：")
	tui.PrintTable([]string{"Ref"}, xslices.Map(refs, func(ref refInfo) []string {
		return []string{ref.name}
	}))
	fmt.Println()
	return tui.ConfirmInlineDefault(fmt.Sprintf("确认向 %s 推送以上 %d 个 ref？", remote.Name, len(refs)), true)
}

// runPush 逐条执行 git push 到选中的 remote，并打印汇总。
//
// 单条 push 失败不中断后续，全部跑完后若有失败则汇总返回一个 error。
func runPush(repoPath string, refs []refInfo) error {
	failed := 0
	for i, ref := range refs {
		remote := ref.remote
		fmt.Printf("\n[%d/%d] git push %s %s\n", i+1, len(refs), remote, ref.name)
		err := git.Push(repoPath, remote, ref.name, false)
		if err != nil {
			failed++
			fmt.Printf("  ✗ 失败: %v\n", err)
		}
	}

	fmt.Println()
	fmt.Printf("完成：%d 条成功，%d 条失败。\n", len(refs)-failed, failed)
	if failed > 0 {
		return fmt.Errorf("有 %d 条推送失败，请检查上方输出", failed)
	}
	return nil
}
