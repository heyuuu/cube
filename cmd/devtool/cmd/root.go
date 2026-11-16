// Package cmd devtool 的 cobra 命令层：tag / version / generate 子命令的组装与执行。
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var workdir string

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devtool",
		Short: "开发工具",
		// 错误不再附带 usage；需要时用 `devtool help <命令>` 查看
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// PersistentPreRunE 之前发生的输入类错误仍附 usage；运行期错误不再附 usage
			cmd.SilenceUsage = true
			return applyWorkdir(workdir)
		},
		// 不创建 completion 命令
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}

	cmd.PersistentFlags().StringVarP(&workdir, "workdir", "W", "", "工作目录（子命令相对路径的基准目录，默认当前目录）")

	cmd.AddCommand(newTagCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newGenerateCmd())
	return cmd
}

// Execute 执行 devtool 根命令，失败时以非零码退出。
func Execute() {
	// cobra 默认只执行最近一层 PersistentPreRunE，generate 子命令有自己的钩子（resolveModule），
	// 会顶掉根命令的钩子导致 workdir 不生效；开启后沿命令链逐层执行（先 root 后 generate）
	cobra.EnableTraverseRunHooks = true
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

// applyWorkdir 把工作目录切到 dir（支持 ~ 与相对路径），空串保持当前目录不变。
// 子命令的全部路径（go.mod、internal/...、rootDir 向上找 .git）都以 cwd 为基准，
// 统一在入口 chdir 一次，避免把 workdir 参数穿透到每个子命令。
func applyWorkdir(dir string) error {
	if dir == "" {
		return nil
	}
	if dir == "~" || strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("获取 home 路径失败: %w", err)
		}
		dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("工作目录不存在或不是目录: %s", dir)
	}
	return os.Chdir(dir)
}
