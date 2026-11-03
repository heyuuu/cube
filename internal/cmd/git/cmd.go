// Package git 提供 `cube git` 命令组，提供git相关命令
package git

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// NewCmd 是 `cube git` 命令组入口，纯分发。
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use: "git",
	}

	cmd.AddCommand(newPushCmd(a)) // push
	cmd.AddCommand(newPullCmd(a)) // pull

	/* <<subcommand inject point>> */

	return cmd
}
