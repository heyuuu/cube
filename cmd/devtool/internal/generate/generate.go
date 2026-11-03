// Package generate 按子命令生成业务 domain 脚手架、在已有 domain 追加 api 端点。
// 经 justfile 调用：just gen domain <name>、just gen api <GET|POST> <domain> <base-name>。
package generate

import (
	"fmt"

	"github.com/spf13/cobra"
)

var module string

// NewGenerateCmd generate 命令。新增生成子命令在这里 AddCommand，实现放独立文件、模板放 tpl/。
func NewGenerateCmd() *cobra.Command {
	root := &cobra.Command{
		Use:               "generate",
		Short:             "代码生成：domain 脚手架、api 端点、command 命令",
		SilenceUsage:      true,                                             // 运行期错误只报错，不再附整段 usage
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true}, // 内部工具，不需要补全子命令
		PersistentPreRunE: func(_ *cobra.Command, _ []string) (err error) {
			module, err = resolveModule()
			return err
		},
	}
	root.SetErrPrefix("generate: ")

	root.AddCommand(newDomainCmd())
	root.AddCommand(newAPICmd())
	root.AddCommand(newCommandGenCmd())

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

// newCommandGenCmd command 子命令：1 个参数生成根命令，2 个参数生成命令组子命令。
func newCommandGenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "command <name> | command <group> <name>",
		Short: "生成 cobra 命令脚手架（根命令或命令组子命令）",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 || len(args) > 2 {
				return fmt.Errorf("%s 需要 1~2 个参数，实际收到 %d 个，用法: generate %s", cmd.Name(), len(args), cmd.Use)
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			return genCommand(module, args)
		},
	}
}

// exactArgs 位置参数个数校验，报错带上子命令用法。
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("%s 需要 %d 个参数，实际收到 %d 个，用法: generate %s", cmd.Name(), n, len(args), cmd.Use)
		}
		return nil
	}
}
