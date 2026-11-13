package server

import (
	"fmt"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
)

// newWorkbenchCmd `cube server workbench` 命令。
func newWorkbenchCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workbench <path>",
		Short: "打开工作台（workbench）页面",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, err := cmdutil.ExtendPath(args[0])
			if err != nil {
				return fmt.Errorf("解析路径失败: %w", err)
			}

			baseURL := a.Server().ServerURL()
			url := buildURL(baseURL, "workbench", Q{"path": path})

			openInBrowser(url)
			return nil
		},
	}
	return cmd
}
