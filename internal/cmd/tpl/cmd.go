// Package tpl 提供 `cube tpl` 命令组，提供tpl相关命令
package tpl

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// NewCmd 是 `cube tpl` 命令组入口，纯分发。
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use: "tpl",
	}

	cmd.AddCommand(newCreateCmd(a)) // 使用模板初始化项目

	/* <<subcommand inject point>> */

	return cmd
}
