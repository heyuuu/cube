package create

import (
	"cube/core/config"
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
	return Create(source, templateName, targetPath, cliVars)
}
