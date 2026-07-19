package opener

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/opener"
	"github.com/heyuuu/cube/util/console"
	"github.com/heyuuu/cube/util/easycobra"
)

var RootCmd = &easycobra.Command{
	Use:     "opener",
	Aliases: []string{"app"},
}

func init() {
	RootCmd.AddCommand(appListCmd)
	RootCmd.AddCommand(appSearchCmd)
}

// cmd `app list`
var appListCmd = &easycobra.Command{
	Use:   "list",
	Short: "列出可用命令列表",
	Run: func(cmd *cobra.Command, args []string) {
		service := app.Default().OpenerService()
		apps := service.Openers()
		showApps(apps)
	},
}

// cmd `app search`
var appSearchCmd = &easycobra.Command{
	Use:   "search {query? : 命令名，支持模糊匹配}",
	Short: "搜索可用命令列表",
	Run: func(cmd *cobra.Command, args []string) {
		query := args

		// 获取匹配的命令列表
		service := app.Default().OpenerService()
		apps := service.Search(strings.Join(query, " "))

		// 返回结果
		showApps(apps)
	},
}

func showApps(apps []*opener.Opener) {
	console.PrintTableFunc(apps, []string{
		fmt.Sprintf("项目(%d)", len(apps)),
		"路径",
	}, func(app *opener.Opener) []string {
		return []string{
			app.Name(),
			app.Bin(),
		}
	})
}
