// gen 代码生成工具：按子命令生成业务 domain 脚手架、在已有 domain 追加 api 端点。
// 工具目录下经 justfile 调用：just gen domain <name>、just gen api <GET|POST> <domain> <base-name>。
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

var module string

// newRootCmd 根命令：新增生成子命令在这里 AddCommand，实现放独立文件、模板放 tpl/。
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:               "gen",
		Short:             "代码生成工具：domain 脚手架、api 端点",
		SilenceUsage:      true,                                             // 运行期错误只报错，不再附整段 usage
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true}, // 内部工具，不需要补全子命令
		PersistentPreRunE: func(_ *cobra.Command, _ []string) (err error) {
			module, err = resolveModule()
			return err
		},
	}
	root.SetErrPrefix("gen: ")

	root.AddCommand(newDomainCmd())
	root.AddCommand(newAPICmd())

	return root
}

// newDomainCmd domain 子命令。
func newDomainCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "domain <name>",
		Short: "生成业务 domain 脚手架",
		Args:  exactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return genDomain(module, args[0])
		},
	}
}

// newAPICmd api 子命令。
func newAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api <GET|POST> <domain> <base-name>",
		Short: "在已有 domain 的 handler 追加 api 端点",
		Args:  exactArgs(3),
		RunE: func(_ *cobra.Command, args []string) error {
			return genAPI(args[0], args[1], args[2])
		},
	}
}

// exactArgs 位置参数个数校验，报错带上子命令用法。
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("%s 需要 %d 个参数，实际收到 %d 个，用法: gen %s", cmd.Name(), n, len(args), cmd.Use)
		}
		return nil
	}
}
