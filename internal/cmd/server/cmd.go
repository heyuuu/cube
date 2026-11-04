// Package server 提供 `cube server` 命令组，提供server相关命令
package server

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// NewCmd 是 `cube server` 命令组入口，纯分发。
//
// `cube server` 命令族：管理本地 web server 的生命周期。
//
// 命令族（进程管理走 HTTP API，参 docs/proposals/archived/1036-server进程管理定调/）：
//
//	cube server              # = cube server status（查看状态比启动更频繁，裸跑给高频动作）
//	cube server start        # 前台启动（常驻形态：prod launchd / dev air，不自 fork）
//	cube server stop         # 触发后台 server 平滑关闭（POST /api/system/shutdown）
//	cube server status       # 探活（GET /api/system/status）
func NewCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "管理本地 web server（start / stop / status / ui）",
	}

	cmd.AddCommand(newStartCmd(a))   // start
	cmd.AddCommand(newStopCmd(a))    // stop
	cmd.AddCommand(newStatusCmd(a))  // status
	cmd.AddCommand(newOpenapiCmd(a)) // openapi

	/* <<subcommand inject point>> */

	return cmd
}
