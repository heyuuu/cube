package ui

import (
	"github.com/spf13/cobra"

	"cube/cmd/env"
)

// newMdCmd `cube ui md <path>` —— 以 Web 方式打开 markdown 文件（原一级命令 `cube md` 迁入）。
//
// 页面与渲染归前端工程（/md?path=<abs>），本命令只负责拼 URL 并开浏览器。
func newMdCmd(env *env.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "md <path>",
		Short: "以 Web 方式打开 markdown 文件",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return openPage(env.App(), "md", args[0])
		},
	}
	return cmd
}
