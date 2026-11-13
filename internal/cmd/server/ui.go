package server

import (
	"github.com/spf13/cobra"

	"cube/internal/app"
)

// newUiCmd `cube server ui` 命令。
func newUiCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "打开 Web UI 首页",
		RunE: func(_ *cobra.Command, _ []string) error {
			baseURL := a.Server().ServerURL()
			openInBrowser(baseURL)
			return nil
		},
	}
	return cmd
}
