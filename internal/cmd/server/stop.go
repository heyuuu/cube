package server

import (
	"fmt"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/server/client"
)

// newStopCmd `cube server stop`
// 触发后台 server 平滑关闭（POST /api/system/shutdown）。
func newStopCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "停止后台 server",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			client := client.NewClient(a.Server().ServerURL())
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
