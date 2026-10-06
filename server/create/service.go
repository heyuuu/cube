package create

import (
	"fmt"
	"os"
	"os/exec"

	"cube/core/config"
	"cube/util/pathkit"
	"cube/util/tui"
)

// settingsSection settings.json 中 create 域的节名。
const settingsSection = "create"

// Service 是模板引擎（cube create）的入口。
// 默认模板来源存 settings.json 的 create 节（templateSource），直读不缓存。
type Service struct {
	settingsFile string
}

func NewService(settingsFile string) *Service {
	return &Service{settingsFile: settingsFile}
}

// DefaultSource 返回默认模板来源（settings.json create 节的 templateSource），
// 供 cmd 层交互收集来源时预填——直读不缓存，改完即生效。
func (s *Service) DefaultSource() string {
	var section struct {
		TemplateSource string `json:"templateSource"` // 未显式传 --tpl 时的默认模板来源（本地目录或 git url）
	}
	config.LoadSection(s.settingsFile, settingsSection, &section)
	return section.TemplateSource
}

// Create 生成项目到 targetPath。
// source 与 targetPath 须为绝对路径（或 git url），~/ 与相对路径由 cmd 层 ExtendPath 展开——
// domain 不感知进程 cwd 与 home；
// templateName 语义：单模板传名报错、模板集缺省交互选择、指定名不存在报错并列出可用；
// cliVars 传了未声明的 key 报错，缺的交互提问。
func (s *Service) Create(source, templateName, targetPath string, cliVars map[string]string) error {
	if source == "" {
		return fmt.Errorf("模板来源不能为空")
	}

	// 目标路径预检放在一切交互之前——不能让用户答完来源/模板/变量才被告知路径非法
	if err := validateTarget(targetPath); err != nil {
		return err
	}

	sourceDir, cleanup, err := ResolveTemplateDir(source)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	layout, err := InspectSource(sourceDir)
	if err != nil {
		return err
	}
	templateDir, err := SelectTemplateDir(layout, templateName)
	if err != nil {
		return err
	}

	tpl, err := loadTemplate(templateDir)
	if err != nil {
		return err
	}

	vars, err := s.collectVariables(tpl, cliVars)
	if err != nil {
		return err
	}

	if err := interpolateTemplate(tpl, vars); err != nil {
		return err
	}

	absTarget, err := prepareTarget(targetPath)
	if err != nil {
		return err
	}

	count, err := Render(templateDir, absTarget, tpl, vars)
	if err != nil {
		return err
	}

	if err := runInit(tpl.Init, absTarget); err != nil {
		return err
	}

	fmt.Printf("已生成 %d 个文件到 %s\n", count, pathkit.PrettyPath(absTarget))
	if len(tpl.Init) == 0 {
		fmt.Println("模板未声明 init 命令，可自行初始化（如 git init）。")
	}
	return nil
}

// loadTemplate 校验模板目录并解析根下的 template.yaml。
func loadTemplate(templateDir string) (*TemplateMeta, error) {
	info, err := os.Stat(templateDir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("模板目录不存在或不是目录: %s", pathkit.PrettyPath(templateDir))
	}
	data, err := os.ReadFile(templateDir + string(os.PathSeparator) + "template.yaml")
	if err != nil {
		return nil, fmt.Errorf("模板目录缺少 template.yaml（不是合法模板目录）: %s", pathkit.PrettyPath(templateDir))
	}
	return InitTemplateMeta(data)
}

// collectVariables 收集变量：cliVars 优先；未提供的用 prompt 交互提问
// （default 预填，required 拒绝空值）。cliVars 里的未声明 key 视为拼写错误报错。
func (s *Service) collectVariables(tpl *TemplateMeta, cliVars map[string]string) (map[string]string, error) {
	varDecls := tpl.Variables

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

// interpolateTemplate 对 patterns 的 Replace 与 init 命令做 ${var} 插值。
func interpolateTemplate(tpl *TemplateMeta, vars map[string]string) (err error) {
	tpl.Patterns, err = interpolatePatterns(tpl.Patterns, vars)
	if err != nil {
		return err
	}
	tpl.Init, err = interpolateInit(tpl.Init, vars)
	if err != nil {
		return err
	}
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

// prepareTarget 创建目标目录（含多级），返回绝对路径。合法性已由 validateTarget 预检。
func prepareTarget(targetPath string) (string, error) {
	if err := os.MkdirAll(targetPath, 0o755); err != nil {
		return "", fmt.Errorf("创建目标目录失败: %w", err)
	}
	return targetPath, nil
}

// runInit 逐条执行 init 命令（sh -c，cwd 为生成后的项目根，stdio 接终端），任一失败中止。
func runInit(cmds []string, dir string) error {
	for i, cmd := range cmds {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = dir
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		fmt.Printf("init[%d]: %s\n", i+1, cmd)
		if err := c.Run(); err != nil {
			return fmt.Errorf("init 第 %d 条命令失败（已中止后续命令）: %q: %w", i+1, cmd, err)
		}
	}
	return nil
}
