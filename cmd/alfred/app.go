package alfred

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/util/slicekit"
)

// cmd `alfred app-search`
var appSearchCmd = &easycobra.Command{
	Use:   "app-search {query? : 命令名，支持模糊匹配} {--project= : 项目名}",
	Short: "搜索可用命令列表",
	InitRun: func(cmd *cobra.Command) easycobra.Run {
		// init flags
		var projectName string
		cmd.Flags().StringVar(&projectName, "project", "", "项目名")

		// run
		return func(args []string) error {
			query := args

			// sticky: alfred 选择项目后会以空参数调用此命令
			if len(query) == 0 && len(projectName) > 0 {
				app.Default().HistoryService().AddProjectSelectLog(projectName, true)
			}

			// 获取匹配的命令列表
			service := app.Default().OpenerService()
			apps := service.Search(strings.Join(query, " "))

			// 若指定项目，且对应空间有指定命令优先级，则按优先级排序
			preferApps := getProjectPreferApps(projectName)
			apps = sortApps(apps, preferApps)

			// 返回结果
			return PrintResult(apps, func(item *opener.Opener) Item {
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

	return nil
}

func sortApps(apps []*opener.Opener, preferAppNames []string) []*opener.Opener {
	if len(apps) <= 1 || len(preferAppNames) == 0 {
		return apps
	}

	preferAppNameMap := make(map[string]int, len(preferAppNames))
	for i, appName := range preferAppNames {
		preferAppNameMap[appName] = i
	}

	return slicekit.SortByWithIndex(apps, func(i int, app *opener.Opener) int {
		if idx, ok := preferAppNameMap[app.Name()]; ok {
			return idx
		} else {
			return i + len(preferAppNames)
		}
	})
}
