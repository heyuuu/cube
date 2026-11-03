package server

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/version"
)

// newStartCmd `cube server start` —— 前台启动 server（Ctrl+C 退出）。
//
// 不提供后台 detach 形态：常驻由系统级保活承担（prod launchd / dev air），
// 自 fork 曾有 argv 不透传与启动失败无声的结构性问题，已移除（1036）。
func newStartCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "前台启动 server（Ctrl+C 退出）",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			// 常驻 server 启动后台任务（项目视图定时刷新等）；server 退出（Start 返回）时停止。
			a.StartBackgroundJobs()
			defer a.StopBackgroundJobs()

			fmt.Printf("cube version: %s\n", version.Info())
			fmt.Printf("server 启动中\n")
			fmt.Printf("  访问地址：%s\n", a.Server().ServerURL())
			slog.Info("server 启动中", "url", a.Server().ServerURL())
			return a.Server().Start()
		},
	}
	return cmd
}
