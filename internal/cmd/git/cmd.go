package git

import (
	"github.com/spf13/cobra"
)

// NewCmd 是 `cube git` 命令组入口，纯分发。
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "git",
	}

	cmd.AddCommand(newPushCmd()) // push
	cmd.AddCommand(newPullCmd()) // pull

	/* <<subcommand inject point>> */

	return cmd
}
