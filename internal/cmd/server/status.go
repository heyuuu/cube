package server

import (
	"fmt"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/server/client"
	"cube/pkg/tui"
)

// newStatusCmd `cube server status`
// 探活（GET /api/system/status）。
func newStatusCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "查看 server 运行状态",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServerStatus(a)
		},
	}
	return cmd
}

func runServerStatus(a *app.App) error {
	client := client.NewClient(a.Server().ServerURL())
	st := client.Status()

	// print server status
	state := "未运行"
	serverVersion := "-"
	serverInstance := "-"
	url := "-"
	if st.Running {
		state = "运行中"
		serverVersion = st.Version
		serverInstance = st.Instance
		url = a.Server().ServerURL()
	}
	tui.PrintTable(
		[]string{"状态", "端口", "版本", "实例", "访问地址"},
		[][]string{{state, fmt.Sprintf("%d", a.Server().Port()), serverVersion, serverInstance, url}},
	)
	return nil
}
