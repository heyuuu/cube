package server

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
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
			baseURL := a.Server().ServerURL()
			return runServerStatus(baseURL)
		},
	}
	return cmd
}

func runServerStatus(baseURL string) error {
	st := serverStatus(baseURL)

	// print server status
	state := "未运行"
	serverVersion := "-"
	serverInstance := "-"
	url := "-"
	if st.Running {
		state = "运行中"
		serverVersion = st.Version
		serverInstance = st.Instance
		url = baseURL
	}
	tui.PrintTable(
		[]string{"状态", "版本", "实例", "访问地址"},
		[][]string{{state, serverVersion, serverInstance, url}},
	)
	return nil
}
