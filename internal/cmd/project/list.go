package project

import (
	"slices"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/project"
)

// newListCmd `cube project list` 命令。
func newListCmd(a *app.App) *cobra.Command {
	var group string
	cmd := &cobra.Command{
		Use:   "list [query] [-g|--group=组名]",
		Short: "项目列表(支持项目名或项目路径模糊搜索)",
		Long: `显示项目列表，支持按项目名称或项目路径进行搜索。

query 支持两种搜索模式：
  - 项目名称搜索：按关键词模糊匹配项目名称（默认）。
  - 项目路径搜索：当 query 以 '.'、'~' 或 '/' 开头时触发，
    搜索给定路径及其所有子目录中的项目。

不传入 query 时，显示所有项目。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// 获取输入参数
			query := xslices.Get(args, 0)

			// 项目列表
			projects, err := cmdutil.SearchProjects(a.ProjectService(), query)
			if err != nil {
				return err
			}

			// 按 group 过滤
			if len(group) > 0 {
				projects = slices.DeleteFunc(projects, func(p *project.Project) bool {
					return p.Group() != group
				})
			}

			// 展示项目列表（最近使用的项目置顶）
			latest := a.UsageService().LatestByProject()
			projects = project.SortByRecentUsage(projects, latest, 10)
			cmdutil.ShowProjects(a.ProjectService(), projects)

			return nil
		},
	}
	cmd.Flags().StringVarP(&group, "group", "g", "", "限定分组 group")
	return cmd
}
