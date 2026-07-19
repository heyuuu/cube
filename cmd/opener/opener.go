package opener

import (
	"fmt"
	"strings"

	"github.com/heyuuu/cube/app"
	"github.com/heyuuu/cube/cmd/util/console"
	"github.com/heyuuu/cube/cmd/util/easycobra"
	"github.com/heyuuu/cube/opener"
)

var RootCmd = &easycobra.Command{
	Use: "opener",
	Children: []*easycobra.Command{
		openerListCmd,
		openerSearchCmd,
	},
}

var openerListCmd = &easycobra.Command{
	Use:   "list [query]",
	Short: "列出可用 Opener 列表",
	Run: func(args []string) error {
		service := app.Default().OpenerService()
		apps := service.Openers()
		showOpeners(apps)
		return nil
	},
}

var openerSearchCmd = &easycobra.Command{
	Use:   "search [query]",
	Short: "搜索可用 Opener 列表",
	Run: func(args []string) error {
		query := args

		// 获取匹配的命令列表
		service := app.Default().OpenerService()
		apps := service.Search(strings.Join(query, " "))

		// 返回结果
		showOpeners(apps)
		return nil
	},
}

func showOpeners(apps []*opener.Opener) {
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
