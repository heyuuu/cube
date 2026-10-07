package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cube/cmd/env"
)

// cmd `cube create`（模板引擎：本地目录 / git 仓库，单模板或模板集）
//
// 两种主用法，其余是边缘校验：
//  1. cube create <模板源> <目标路径> —— 交互式逐步创建（来源/模板名/变量缺啥问啥）；
//  2. cube create <模板源> <目标路径> --var k=v ... —— 非交互一步生成。
//
// <模板源> 支持以下几种值
//  1. `@<source>/<tpl>` 表示 source 模板源下的 tpl 模板
//  2. `<tpl>`			 表示默认模板源下的 tpl 模板
//  3. `<本地目录>` 	 指向一个本地模板目录
func newCreateCmd(env *env.Env) *cobra.Command {
	var cliVars []string
	cmd := &cobra.Command{
		Use:   "create <模板名> <目标路径> [--var key=value ...]",
		Short: "使用模板生成项目",
		Long: `使用模板生成项目

--var key=value：模板变量，可多次。传未声明的变量报错，缺的交互提问。

示例：
  cube create my-app
  cube create my-app --tpl-name go-service --var author=heyu
  cube create my-app --tpl ~/templates --tpl-name full
  cube create my-app --tpl https://github.com/xxx/templates.git --tpl-name go-service --var author=heyu`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliTpl := args[0]
			target := args[1]
			vars, err := parseCliVars(cliVars)
			if err != nil {
				return err
			}

			// 解析 cliTpl <模板源> 的参数
			source, tplName, err := parseCliTpl(cliTpl)
			if err != nil {
				return fmt.Errorf("解析模板源参数失败: %w", err)
			}

			// 目标路径在本层展开为绝对路径（~/ 相对路径），domain 不感知进程 cwd
			target, err = ExtendPath(target)
			if err != nil {
				return fmt.Errorf("解析目标路径失败: %w", err)
			}

			// 创建模板
			svc := env.App().CreateService()
			return svc.Create(source, tplName, target, vars)
		},
	}

	cmd.Flags().StringSliceVarP(&cliVars, "var", "v", nil, "模板变量 key=value（可多次）")

	return cmd
}

// parseCliVars 解析 --var key=value 为 map，非法格式报错。
func parseCliVars(items []string) (map[string]string, error) {
	vars := make(map[string]string, len(items))
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("--var 参数格式应为 key=value: %q", item)
		}
		if _, dup := vars[key]; dup {
			return nil, fmt.Errorf("--var 重复的变量: %s", key)
		}
		vars[key] = value
	}
	return vars, nil
}

// parseCliTpl 解析命令中的 <模板源> 参数
func parseCliTpl(cliTpl string) (source string, tplName string, err error) {
	// <模板源> 为本地路径
	if isPathQuery(cliTpl) {
		absPath, err := ExtendPath(cliTpl)
		if err != nil {
			return "", "", err
		}
		return absPath, "", nil
	}

}
