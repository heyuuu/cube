package project

import (
	"fmt"
	"log/slog"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/opener"
)

// newOpenCmd `cube project open` 命令。
func newOpenCmd(a *app.App) *cobra.Command {
	var openerName string
	cmd := &cobra.Command{
		Use:   "open [query] [-o|--opener[=打开工具名]]",
		Short: "打开项目。默认 opener 直开，-o 指定/交互选择",
		Long: `用 opener 打开一个已收录的项目目录（intent: dir）。

query 支持项目名和项目列表模糊搜索，具体规则同 info 命令。
不传 -o 时直接使用 dir 意图的默认 opener（openerIntents 节配置）；
-o 不带值时交互选择；-o <name> 按名模糊匹配。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			query := xslices.Get(args, 0)

			// 匹配项目
			proj, err := pickProject(a.ProjectService(), query, a.Local())
			if err != nil {
				return err
			}

			// 选 opener（intent: dir，不带 -o 时用默认）
			o, err := cmdutil.PickOpener(a.OpenerService(), opener.IntentDir, openerName)
			if err != nil {
				return err
			}

			// 选打开目标（1032 worktree / 1030 workspace 归并为项目打开目标）：多目标时交互选择，单目标流程不变
			target, err := pickOpenTarget(a.ProjectService(), proj)
			if err != nil {
				return err
			}

			// 打开项目
			err = o.Open(opener.RoleOpenDir, target)
			if err != nil {
				return fmt.Errorf("打开失败: %w", err)
			}

			// 记录使用信号（best-effort：失败不影响打开结果）；project 恒记主项目路径，
			// dir 直接传实际打开的目标目录（等于根时由 RecordOpen 归一为空）
			if err := a.UsageService().RecordOpen(proj.Path(), o.Name(), target); err != nil {
				slog.Warn("记录 usage 失败", "err", err)
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&openerName, "opener", "o", "", "打开工具(opener)名；不带值时交互选择，缺省用默认 opener")
	cmd.Flags().Lookup("opener").NoOptDefVal = cmdutil.InteractivePick
	return cmd
}
