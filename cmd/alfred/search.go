package alfred

import (
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/entities"
	"github.com/heyuuu/cube/util/easycobra"
)

// cmd `alfred project-search`
var projectSearchCmd = &easycobra.Command{
	Use:   "project-search {query?* : 项目名，支持模糊匹配}",
	Short: "搜索项目列表",
	Run: func(cmd *cobra.Command, args []string) {
		// 获取输入参数
		query := strings.Join(args, " ")

		// 项目列表
		service := app.Default().ProjectService()
		projects := service.Search(query)

		// 最近打开日志
		historyService := app.Default().HistoryService()
		history := historyService.LeastSelectedProjects(10, true)
		sortProjectsWithHistory(projects, history)

		// 返回结果
		PrintResultFunc(projects, func(proj *entities.Project) Item {
			return Item{
				Title:    proj.Name(),
				SubTitle: proj.RepoUrl(),
				Arg:      proj.Name(),
			}
		})
	},
}

// 优先将 history 排在前面，保持其他顺序不变
func sortProjectsWithHistory(projects []*entities.Project, history []string) []*entities.Project {
	weights := make(map[string]int, len(history))
	for i, proj := range projects {
		weights[proj.Name()] = i + len(history)
	}
	for i, proj := range history {
		weights[proj] = i
	}

	slices.SortFunc(projects, func(a, b *entities.Project) int {
		return weights[a.Name()] - weights[b.Name()]
	})

	return projects
}
