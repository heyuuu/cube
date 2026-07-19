package alfred

import (
	"strings"

	"github.com/heyuuu/cube/internal/app"
	"github.com/heyuuu/cube/internal/entities"
	"github.com/heyuuu/cube/internal/util/easycobra"
	"github.com/heyuuu/cube/internal/util/slicekit"
	"github.com/spf13/cobra"
)

// cmd `alfred app-search`
var appSearchCmd = &easycobra.Command{
	Use:   "app-search {query? : 命令名，支持模糊匹配} {--project= : 项目名}",
	Short: "搜索可用命令列表",
	InitRun: func(cmd *cobra.Command) func(cmd *cobra.Command, args []string) {
		// init flags
		var projectName string
		cmd.Flags().StringVar(&projectName, "project", "", "项目名")

		// run
		return func(cmd *cobra.Command, args []string) {
			query := args

			// sticky: alfred 选择项目后会以空参数调用此命令
			if len(query) == 0 && len(projectName) > 0 {
				app.Default().HistoryService().AddProjectSelectLog(projectName, true)
			}

			// 获取匹配的命令列表
			service := app.Default().ApplicationService()
			apps := service.Search(strings.Join(query, " "))

			// 若指定项目，且对应空间有指定命令优先级，则按优先级排序
			preferApps := getProjectPreferApps(projectName)
			apps = sortApps(apps, preferApps)

			// 返回结果
			PrintResultFunc(apps, func(item *entities.Application) Item {
				return Item{
					Title:    item.Name(),
					SubTitle: item.Bin(),
					Arg:      item.Name(),
				}
			})
		}
	},
}

func getProjectPreferApps(projectName string) []string {
	if len(projectName) == 0 {
		return nil
	}

	// 优先将历史记录作为偏好 app
	historyService := app.Default().HistoryService()
	history := historyService.LeastProjectOpenApps(projectName, 3, true)
	if len(history) > 0 {
		return history
	}

	// 使用工作区的编号 app 作为项目默认偏好
	wsService := app.Default().WorkspaceService()
	ws := wsService.FindByProjectName(projectName)
	if ws != nil {
		return ws.PreferApps()
	}

	return nil
}

func sortApps(apps []*entities.Application, preferAppNames []string) []*entities.Application {
	if len(apps) <= 1 || len(preferAppNames) == 0 {
		return apps
	}

	preferAppNameMap := make(map[string]int, len(preferAppNames))
	for i, appName := range preferAppNames {
		preferAppNameMap[appName] = i
	}

	return slicekit.SortByWithIndex(apps, func(i int, app *entities.Application) int {
		if idx, ok := preferAppNameMap[app.Name()]; ok {
			return idx
		} else {
			return i + len(preferAppNames)
		}
	})
}
