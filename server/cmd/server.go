package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"cube/app"
	"cube/cmd/env"
	"cube/core/server"
	"cube/core/version"
	"cube/util/tui"
)

// `cube server` 命令族：管理本地 web server 的生命周期。
//
// 命令族（进程管理走 HTTP API，参 docs/proposals/archived/1036-server进程管理定调/）：
//
//	cube server              # = cube server status（查看状态比启动更频繁，裸跑给高频动作）
//	cube server start        # 前台启动（常驻形态：prod launchd / dev air，不自 fork）
//	cube server stop         # 触发后台 server 平滑关闭（POST /api/system/shutdown）
//	cube server status       # 探活（GET /api/system/status）

// newServerCmd 构建 `cube server` 父命令及其子命令。
func newServerCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "管理本地 web server（start / stop / status）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServerStatus(env.App())
		},
	}

	cmd.AddCommand(newServerStartCmd(env))
	cmd.AddCommand(newServerStopCmd(env))
	cmd.AddCommand(newServerStatusCmd(env))

	return cmd
}

// newServerStartCmd `cube server start` —— 前台启动 server（Ctrl+C 退出）。
//
// 不提供后台 detach 形态：常驻由系统级保活承担（prod launchd / dev air），
// 自 fork 曾有 argv 不透传与启动失败无声的结构性问题，已移除（1036）。
func newServerStartCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "前台启动 server（Ctrl+C 退出）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 常驻 server 启动后台任务（项目视图定时刷新等）；server 退出（Start 返回）时停止。
			env.App().StartBackgroundJobs()
			defer env.App().StopBackgroundJobs()

			fmt.Printf("cube version: %s\n", version.VersionInfo())
			fmt.Printf("server 启动中\n")
			fmt.Printf("  访问地址：%s\n", env.App().Server().ServerURL())
			slog.Info("server 启动中", "url", env.App().Server().ServerURL())
			return env.App().Server().Start()
		},
	}
	return cmd
}

// newServerStopCmd `cube server stop` —— 触发后台 server 平滑关闭（POST /api/system/shutdown）。
func newServerStopCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "停止后台 server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := server.NewClient(env.App().Server().ServerURL())
			stopped, replaced, err := client.Stop()
			if err != nil {
				return err
			}
			switch {
			case !stopped:
				fmt.Println("server 未在运行")
			case replaced:
				fmt.Println("旧实例已停止（端口已被新实例接管，可能由系统保活拉起）")
			default:
				fmt.Println("server 已停止")
			}
			return nil
		},
	}
	return cmd
}

// newServerStatusCmd `cube server status` —— 探活（GET /api/system/status）。
func newServerStatusCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "查看 server 运行状态",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServerStatus(env.App())
		},
	}
	return cmd
}

func runServerStatus(a *app.App) error {
	client := server.NewClient(a.Server().ServerURL())
	st := client.Status()

	// print server status
	state := "未运行"
	version := "-"
	instance := "-"
	url := "-"
	if st.Running {
		state = "运行中"
		version = st.Version
		instance = st.Instance
		url = a.Server().ServerURL()
	}
	tui.PrintTable(
		[]string{"状态", "端口", "版本", "实例", "访问地址"},
		[][]string{{state, fmt.Sprintf("%d", a.Server().Port()), version, instance, url}},
	)
	return nil
}
