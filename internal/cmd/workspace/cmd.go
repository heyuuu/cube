// Package workspace 提供 `cube workspace` 命令组，提供workspace相关命令
package workspace

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// NewCmd 是 `cube workspace` 命令组入口，纯分发。
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"work"},
		Short:   "项目 workspace 管理(项目 .cube/cube.json 管理)",
	}

	cmd.AddCommand(newInitCmd(a)) // init
	/* <<subcommand inject point>> */

	return cmd
}
