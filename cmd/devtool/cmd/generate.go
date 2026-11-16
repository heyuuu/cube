package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"devtool/generate"
)

var module string

func newGenerateCmd() *cobra.Command {
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
			return generate.GenDomain(module, args[0])
		},
	}
}

// newAPICmd api 子命令。
func newAPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "api <GET|POST> <route>",
		Short: "在已有 domain 的 handler 追加 api 端点",
		Args:  exactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return generate.GenAPI(args[0], args[1])
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
			return generate.GenCommand(module, args)
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

// resolveModule 从当前目录 go.mod 读 module 声明。
func resolveModule() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("读取 go.mod 失败（请在工具目录下执行）: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("go.mod 里找不到 module 声明")
}
