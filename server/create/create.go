package create

import (
	"fmt"
	"os"
	"os/exec"

	"cube/util/pathkit"
	"cube/util/tui"
)

// Create 生成项目到 targetPath
func Create(source string, tplName string, targetPath string, cliVars map[string]string) error {
	if source == "" {
		return fmt.Errorf("模板来源不能为空")
	}

	// 目标路径预检放在一切交互之前——不能让用户答完来源/模板/变量才被告知路径非法
	if err := validateTarget(targetPath); err != nil {
		return err
	}

	sourcePath, cleanup, err := ResolveTemplateDir(source)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	// 选择模板
	tplPath, err := selectTplPath(sourcePath, tplName)
	if err != nil {
		return err
	}

	// 加载模板
	tpl, err := LoadTemplate(tplPath)
	if err != nil {
		return err
	}

	// 收集模板必备的参数
	vars, err := collectVariables(tpl, cliVars)
	if err != nil {
		return err
	}

	// 写入目标目录
	count, err := Render(tpl, targetPath, vars)
	if err != nil {
		return err
	}

	// 执行初始化
	if err := runInit(tpl, targetPath, vars); err != nil {
		return err
	}

	fmt.Printf("已生成 %d 个文件到 %s\n", count, pathkit.PrettyPath(targetPath))
	return nil
}

// validateTarget 预检目标路径（绝对路径，cmd 层已展开）：已存在时必须是空目录（防覆盖既有内容）。
// 在 Create 一切交互之前调用，让路径错误第一时间暴露。
func validateTarget(targetPath string) error {
	if info, err := os.Stat(targetPath); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("目标路径已存在且不是目录: %s", pathkit.PrettyPath(targetPath))
		}
		entries, err := os.ReadDir(targetPath)
		if err != nil {
			return fmt.Errorf("读取目标目录失败: %w", err)
		}
		if len(entries) > 0 {
			return fmt.Errorf("目标目录非空，拒绝覆盖: %s", pathkit.PrettyPath(targetPath))
		}
	}
	return nil
}

// 交互选择模板, 返回模板路径
func selectTplPath(sourcePath string, tplName string) (string, error) {
	layout, err := InspectSource(sourcePath)
	if err != nil {
		return "", err
	}
	templateDir, err := SelectTemplateDir(layout, tplName)
	if err != nil {
		return "", err
	}
	return templateDir, nil
}

// collectVariables 收集变量：cliVars 优先；未提供的用 prompt 交互提问
// （default 预填，required 拒绝空值）。cliVars 里的未声明 key 视为拼写错误报错。
func collectVariables(tpl *Template, cliVars map[string]string) (map[string]string, error) {
	varDecls := tpl.Meta().Variables

	// 检查预设参数是否有拼写错误
	varNameSet := make(map[string]struct{}, len(varDecls))
	for _, v := range varDecls {
		varNameSet[v.Name] = struct{}{}
	}
	for name := range cliVars {
		if _, exists := varNameSet[name]; exists {
			return nil, fmt.Errorf("变量 %q 未在 template.yaml 中声明（检查拼写）", name)
		}
	}

	// 按定义属性遍历变量并处理
	vars := make(map[string]string, len(varDecls))
	for _, decl := range varDecls {
		name := decl.Name
		if v, ok := cliVars[name]; ok {
			vars[name] = v
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
			title = name
		}

		v, err := tui.Input(title, decl.Default, "", validate)
		if err != nil {
			return nil, err
		}
		if v == "" && decl.Default != "" {
			v = decl.Default
		}
		vars[name] = v
	}
	return vars, nil
}

// runInit 逐条执行 init 命令（sh -c，cwd 为生成后的项目根，stdio 接终端），任一失败中止。
func runInit(tpl *Template, target string, vars map[string]string) error {
	cmds, err := interpolateInit(tpl.Meta().Init, vars)
	if err != nil {
		return err
	}

	for i, cmd := range cmds {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = target
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		fmt.Printf("init[%d]: %s\n", i+1, cmd)
		if err := c.Run(); err != nil {
			return fmt.Errorf("init 第 %d 条命令失败（已中止后续命令）: %q: %w", i+1, cmd, err)
		}
	}
	return nil
}
