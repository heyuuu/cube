package tpl

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"cube/internal/app"
	"cube/internal/cmd/cmdutil"
	"cube/internal/domain/template"
	"cube/pkg/tui"
)

// cmd `cube tpl create`（模板引擎：命名 git 源 / 本地目录，单模板或模板集）
//
// 两种主用法，其余是边缘校验：
//  1. cube create [模板源] <目标路径> —— 交互式逐步创建（模板名/变量缺啥问啥）；
//  2. cube create [模板源] <目标路径> --var k=v ... —— 非交互一步生成。
//
// <模板源> 可省略（单参数形态 = 目标路径，默认源交互选模板），由首字符零歧义区分形态（提案 1045）：
//  1. `@<source>/<tpl>` 命名源的指定模板；`@<source>` 命名源交互选模板（npm scope 式 @ 前缀）
//  2. `<本地路径>`      . ~/ / 开头，直读模板目录（开发态：正在编辑的模板仓库）
//  3. `<tpl>`           裸名，默认源 core 的指定模板
//  4. 省略              默认源 core，交互选模板
//
// 用户交互（选模板/提问变量）收敛在本层：domain 只暴露无交互的分阶段 API
// （CheckTarget → OpenSource → Source.Template → CheckVars → Generate）。
func newCreateCmd(a *app.App) *cobra.Command {
	var cliVars []string
	cmd := &cobra.Command{
		Use:   "create [模板源] <目标路径> [--var key=value ...]",
		Short: "使用模板生成项目",
		Long: `使用模板生成项目

--var key=value：模板变量，可多次。传未声明的变量报错，缺的交互提问。

模板源（可省略）四种写法：
  @core/go-service   命名源 core 的 go-service 模板（源在设置页「模板源」配置）
  go-service         默认源 core 的 go-service 模板
  ~/code/tpl-dev     本地模板目录直读（开发态：不 clone，吃未推送的改动；目录在 git 仓库内时按 .gitignore 过滤，与 clone 口径一致）
  省略               默认源 core，交互选模板

示例：
  cube create ~/code/new-app                     # 默认源交互选模板
  cube create go-service ~/code/new-app --var author=heyu
  cube create @core/go-service ~/code/new-app
  cube create @work ~/code/new-app
  cube create ~/code/tpl-dev ~/code/new-app`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			// 目标路径恒为最后一个参数；模板源是可选的前一个（缺省走默认源交互）
			cliTpl := ""
			target := args[len(args)-1]
			if len(args) == 2 {
				cliTpl = args[0]
			}
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
			target, err = cmdutil.ExtendPath(target)
			if err != nil {
				return fmt.Errorf("解析目标路径失败: %w", err)
			}

			svc := a.TemplateService()

			// 路径预检先行——clone、交互都可能耗时，不能跑完才发现路径非法
			if err := svc.CheckTarget(target); err != nil {
				return err
			}

			// 加载模板源
			src, err := svc.OpenSource(sourceRef)
			if err != nil {
				return err
			}
			defer src.Cleanup() // 临时 clone 目录用后即删（失败路径同样清理）

			// 未指名模板时：根模板形态唯一确定直接选中；模板库形态交互选择
			// （库即使当前只有一个模板也仍交互——自动选中挂在源的结构声明上，不挂在数量上）
			if tplName == "" {
				if name := src.RootTemplateName(); name != "" {
					tplName = name
				} else {
					tplName, err = tui.SelectItem("选择模板", src.Names(), func(n string) string { return n })
					if err != nil {
						return err
					}
				}
			}
			tpl, err := src.Template(tplName)
			if err != nil {
				return err
			}

			// 收集变量：--var 优先，缺的交互提问
			vars, err = promptVars(tpl, vars)
			if err != nil {
				return err
			}

			count, err := svc.Generate(tpl, target, vars)
			if err != nil {
				return err
			}
			fmt.Printf("已生成 %d 个文件到 %s\n", count, target)
			return nil
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
	if absPath, err := cmdutil.AsPathArg(cliTpl); err == nil {
		return absPath, "", nil
	}

	// 完全缺省（单参数形态）→ 默认源交互选模板
	if cliTpl == "" {
		return template.DefaultSource, "", nil
	}
	// 裸名 → 默认源；带 / 的既非路径也非 @ 引用，提前拦截（否则要 clone 完才在模板查找处报错）
	if strings.Contains(cliTpl, "/") {
		return "", "", fmt.Errorf("命名源引用请用 @source/模板名 形式: %q", cliTpl)
	}
	return template.DefaultSource, cliTpl, nil
}

// promptVars 收集模板变量：cliVars 优先；未提供的用 prompt 交互提问
// （default 预填，required 拒绝空值）。cliVars 里的未声明 key 视为拼写错误报错
// （先于逐项提问暴露，校验在 domain 的 CheckVars）。
func promptVars(tpl *template.Template, cliVars map[string]string) (map[string]string, error) {
	if err := tpl.CheckVars(cliVars); err != nil {
		return nil, err
	}

	// 按定义属性遍历变量并处理
	vars := make(map[string]string, len(tpl.Meta().Variables))
	for _, decl := range tpl.Meta().Variables {
		if v, ok := cliVars[decl.Name]; ok {
			vars[decl.Name] = v
			continue
		}

		var validate func(string) error
		if decl.Required && decl.Default == "" {
			validate = func(v string) error {
				if v == "" {
					return fmt.Errorf("%s 为必填项", decl.Prompt)
				}
				return nil
			}
		}

		title := decl.Prompt
		if title == "" {
			title = decl.Name
		}

		v, err := tui.Input(title, decl.Default, "", validate)
		if err != nil {
			return nil, err
		}
		if v == "" && decl.Default != "" {
			v = decl.Default
		}
		vars[decl.Name] = v
	}
	return vars, nil
}
