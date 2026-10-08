package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cube/cmd/env"
	"cube/template"
)

// cmd `cube create`（模板引擎：命名 git 源 / 本地目录，单模板或模板集）
//
// 两种主用法，其余是边缘校验：
//  1. cube create <模板源> <目标路径> —— 交互式逐步创建（模板名/变量缺啥问啥）；
//  2. cube create <模板源> <目标路径> --var k=v ... —— 非交互一步生成。
//
// <模板源> 由首字符零歧义区分三种形态（提案 1045）：
//  1. `@<source>/<tpl>` 命名源的指定模板；`@<source>` 命名源交互选模板（npm scope 式 @ 前缀）
//  2. `<本地路径>`      . ~/ / 开头，直读模板目录（开发态：正在编辑的模板仓库）
//  3. `<tpl>`           裸名，默认源 core 的指定模板
func newCreateCmd(env *env.Env) *cobra.Command {
	var cliVars []string
	cmd := &cobra.Command{
		Use:   "create <模板源> <目标路径> [--var key=value ...]",
		Short: "使用模板生成项目",
		Long: `使用模板生成项目

--var key=value：模板变量，可多次。传未声明的变量报错，缺的交互提问。

模板源三种写法：
  @core/go-service   命名源 core 的 go-service 模板（源在设置页「模板源」配置）
  go-service         默认源 core 的 go-service 模板
  ~/code/tpl-dev     本地模板目录直读（开发态：不 clone，吃未推送的改动）

示例：
  cube create go-service ~/code/new-app --var author=heyu
  cube create @core/go-service ~/code/new-app
  cube create @work ~/code/new-app          # work 源交互选模板
  cube create ~/code/tpl-dev ~/code/new-app`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliTpl := args[0]
			target := args[1]
			vars, err := parseCliVars(cliVars)
			if err != nil {
				return err
			}

			// 解析 cliTpl <模板源> 的参数
			sourceRef, tplName, err := parseCliTpl(cliTpl)
			if err != nil {
				return fmt.Errorf("解析模板源参数失败: %w", err)
			}

			// 目标路径在本层展开为绝对路径（~/ 相对路径），domain 不感知进程 cwd
			target, err = ExtendPath(target)
			if err != nil {
				return fmt.Errorf("解析目标路径失败: %w", err)
			}

			// 创建模板
			svc := env.App().TemplateService()
			return svc.Create(sourceRef, tplName, target, vars)
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

// parseCliTpl 解析命令中的 <模板源> 参数为 (sourceRef, tplName)。
// sourceRef 形态（命名源 / 本地绝对路径）由本函数判定，domain 侧 loadSourceDir 按形态分发。
func parseCliTpl(cliTpl string) (sourceRef string, tplName string, err error) {
	// `@<source>/<tpl>` 或 `@<source>`：@ 前缀显式标记命名源引用
	if strings.HasPrefix(cliTpl, "@") {
		source, tpl, _ := strings.Cut(cliTpl[1:], "/")
		if source == "" {
			return "", "", fmt.Errorf("@ 后缺 source 名: %q", cliTpl)
		}
		if strings.Contains(tpl, "/") {
			return "", "", fmt.Errorf("模板名不能包含 /: %q", tpl)
		}
		return source, tpl, nil
	}

	// 本地路径：./ ~/ / 开头
	if isPathQuery(cliTpl) {
		absPath, err := ExtendPath(cliTpl)
		if err != nil {
			return "", "", err
		}
		return absPath, "", nil
	}

	// 裸名 → 默认源；带 / 的既非路径也非 @ 引用，提前拦截（否则要 clone 完才在模板查找处报错）
	if cliTpl == "" {
		return "", "", fmt.Errorf("模板源不能为空")
	}
	if strings.Contains(cliTpl, "/") {
		return "", "", fmt.Errorf("命名源引用请用 @source/模板名 形式: %q", cliTpl)
	}
	return template.DefaultSource, cliTpl, nil
}
