package server

import (
	"fmt"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
)

// newMdCmd `cube server md` 命令。
func newMdCmd(a *app.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "md <path>",
		Short: "以 Web 方式打开 markdown 文件",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			path, err := cmdutil.ExtendPath(args[0])
			if err != nil {
				return fmt.Errorf("解析路径失败: %w", err)
			}

			baseURL := a.Server().ServerURL()
			url := buildURL(baseURL, "md", Q{"path": path})

			openInBrowser(url)
			return nil
		},
	}
	return cmd
}
