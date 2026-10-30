// Package cmd devtool 的 cobra 命令层：tag / version / generate 子命令的组装与执行。
package cmd

import (
	"os"

	"cube/cmd/devtool/internal/generate"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devtool",
		Short: "开发工具",
		// 错误不再附带 usage；需要时用 `devtool help <命令>` 查看
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// PersistentPreRunE 之前发生的输入类错误仍附 usage；运行期错误不再附 usage
			cmd.SilenceUsage = true
			return nil
		},
		// 不创建 completion 命令
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	cmd.AddCommand(newTagCmd(), newVersionCmd(), generate.NewGenerateCmd())
	return cmd
}

// Execute 执行 devtool 根命令，失败时以非零码退出。
func Execute() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
