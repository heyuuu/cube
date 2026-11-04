package opener

import (
	"fmt"

	"github.com/heyuuu/x/xstd/xslices"
	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/domain/opener"
	"cube/pkg/tui"
)

// newListCmd `cube opener list` 命令。
func newListCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [query]",
		Short: "列出可用 Opener 列表(支持模糊搜索)",
		Long: `显示可用 opener 列表：名称、执行命令及声明的 roles。

query 按 opener 名称模糊搜索，不传时显示全部。`,
		RunE: func(_ *cobra.Command, args []string) error {
			var query string
			if len(args) > 0 {
				query = args[0]
			}

			service := a.OpenerService()
			openers := service.SearchAll(query)
			showOpeners(openers)
			return nil
		},
	}
	return cmd
}

func showOpeners(list []opener.Opener) {
	tui.PrintTable(
		[]string{
			fmt.Sprintf("Opener(%d)", len(list)),
			"Summary",
			"Roles",
		},
		xslices.Map(list, func(o opener.Opener) []string {
			return []string{
				o.Name(),
				o.Summary(),
				opener.RolesString(o.Roles()),
			}
		}),
	)
}
