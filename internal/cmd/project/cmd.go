// Package project 提供 `cube project` 命令组，提供项目相关命令
package project

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// NewCmd 是 `cube project` 命令组入口，纯分发。
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use: "project",
	}

	cmd.AddCommand(newListCmd(a)) // 项目列表
	cmd.AddCommand(newInfoCmd(a)) // 项目信息

	/* <<subcommand inject point>> */

	return cmd
}
