package alfred

import (
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/project"
)

// cmd `alfred project-search`
func newProjectSearchCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project-search {query?* : 项目名，支持模糊匹配}",
		Short: "搜索项目列表",
		RunE: func(_ *cobra.Command, args []string) error {
			// 获取输入参数
			query := strings.Join(args, " ")

			// 项目列表
			projects := a.ProjectService().SearchByName(query)

			// 最近使用的项目置顶
			latest := a.UsageService().LatestByProject()
			projects = sortByRecentUsage(projects, latest, 10)

			// 平铺直达（1032）：每个项目展开为「根目录 + worktrees + workspaces」多个条目，
			// Arg 直接传目标目录路径，保留一步打开体验（无需先选项目再选目标）
			items := make([]Item, 0, len(projects))
			for _, proj := range projects {
				items = append(items, Item{
					Title:    proj.Name(),
					SubTitle: proj.Path(),
					Arg:      proj.Path(),
				})

				// open targets
				snap := a.ProjectService().Snapshot(proj.Path())
				targets := cmdutil.OpenTargets(snap)
				if len(targets) > 0 {
					for _, target := range targets[1:] { // 跳过根目录（上面已输出）
						items = append(items, Item{
							Title:    proj.Name() + " (" + target.Label + ")",
							SubTitle: target.Path,
							Arg:      target.Path,
						})
					}
				}
			}
			return PrintItems(items)
		},
	}
	return cmd
}

// sortByRecentUsage 最近使用的项目置顶（按最近使用倒序，最多 limit 条），其余保持原序。
// CLI 与 alfred 出口共用的一分实现；Web 侧排序在前端做（list 接口返回原始序 + lastUsedAt）。
// latest 是 usage 读侧的 path → 最近使用时间；以 map 传参、纯函数实现，project 包不依赖 usage。
func sortByRecentUsage(projects []*project.Project, latest map[string]time.Time, limit int) []*project.Project {
	pinned := make([]string, 0, len(latest))
	for path := range latest {
		pinned = append(pinned, path)
	}
	slices.SortFunc(pinned, func(a, b string) int {
		return latest[b].Compare(latest[a])
	})
	if len(pinned) > limit {
		pinned = pinned[:limit]
	}

	weights := make(map[string]int, len(projects))
	for i, proj := range projects {
		weights[proj.Path()] = i + len(pinned)
	}
	for i, path := range pinned {
		weights[path] = i
	}

	slices.SortFunc(projects, func(a, b *project.Project) int {
		return weights[a.Path()] - weights[b.Path()]
	})

	return projects
}
