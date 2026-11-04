// Package forge 提供 `cube forge` 命令组，提供forge相关命令
package forge

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// NewCmd 是 `cube forge` 命令组入口，纯分发。
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forge",
		Short: "git 托管平台（forge）配置管理",
		Long: `管理 forge 配置（git 托管平台实例，host 级一条）：
域名（如 github.com、自建 gitea.example.com）、API 方言 kind、图标。
数据存 settings.json 的 forges 节，增删改也可在 Web 设置页完成。`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runForgeList(a)
		},
	}

	cmd.AddCommand(newListCmd(a)) // list

	/* <<subcommand inject point>> */

	return cmd
}
