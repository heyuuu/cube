// Package opener 提供 `cube opener` 命令组，提供opener相关命令
package opener

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// NewCmd 是 `cube opener` 命令组入口，纯分发。
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "opener",
		Aliases: []string{"o", "open"},
	}

	cmd.AddCommand(newListCmd(a)) // list
	cmd.AddCommand(newPathCmd(a)) // path
	cmd.AddCommand(newDiffCmd(a)) // diff

	/* <<subcommand inject point>> */

	return cmd
}
