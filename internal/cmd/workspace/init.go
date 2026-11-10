package workspace

import (
	"errors"
	"fmt"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/project/cubefile"
	"cube/internal/domain/project/workspace"
	"cube/pkg/tui"
)

// newInitCmd `cube workspace init` 命令。
func newInitCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [query]",
		Short: "解析 workspace 规则并挑选成员，固化为显式声明写入 .cube/cube.json",
		Long: `按当前生效的 workspace 规则解析成员（cube.json 的 workspaces 节；无声明时
默认规则 <npm>+<pnpm> 读 pnpm-workspace.yaml / package.json workspaces），
交互挑选后固化为显式规则清单。声明进 git 跟仓库走。

query 支持项目名模糊搜索，规则同 open 命令。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			proj, err := cmdutil.PickProject(a.ProjectService(), xslices.Get(args, 0), a.Local())
			if err != nil {
				return err
			}
			root := proj.Path()

			// 已有显式声明时确认覆盖（解析产的是候选全集，覆盖会丢失人工裁剪）
			if cubefile.WorkspaceRulesDeclared(root) {
				overwrite, err := tui.Confirm("已存在显式 workspaces 规则，重新解析并覆盖？")
				if err != nil {
					return err
				}
				if !overwrite {
					return errors.New("用户取消了覆盖")
				}
			}

			// 用「与采集一致的规则」解析，保证 init 看到的候选就是当前生效全集
			members := workspace.Resolve(root)
			if len(members) == 0 {
				return fmt.Errorf("未解析到 workspace 成员（默认规则读 pnpm-workspace.yaml / package.json workspaces），可手写 %s/.cube/cube.json", root)
			}

			picked, err := tui.MultiSelectItemWithDefaults(
				"挑选 workspace 成员（固化为显式声明）",
				members,
				func(w workspace.Workspace) string { return w.Name + "  (" + w.Path + ")" },
				members, // 默认全选：解析命中即认为都是成员，裁剪是减法
			)
			if err != nil {
				return err
			}
			if len(picked) == 0 {
				return errors.New("未选择任何成员，未写入")
			}

			if err := workspace.Save(root, picked); err != nil {
				return err
			}
			// 定向重采集：写盘即刻反映到打开目标，不等 TTL
			if _, err := a.ProjectService().RefreshSnapshot(root); err != nil {
				fmt.Printf("cube.json 已写入，但刷新快照失败（等下次采集自愈）: %v\n", err)
			}
			fmt.Printf("已写入 %s/.cube/cube.json（%d 个 workspace），记得提交进 git\n", root, len(picked))
			return nil
		},
	}
	return cmd
}
