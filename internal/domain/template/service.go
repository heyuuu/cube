package template

import (
	"cube/internal/config"
)

// Service 是模板引擎（cube create）的入口。
// 模板源映射存 settings.json 的 tplSources 节（直读不缓存，保存即生效）。
type Service struct {
	tplSourcesStore TplSourcesStore
}

// NewService 由 Config 装配 Service（DI 注入用）。
func NewService(cfg *config.Config) *Service {
	return &Service{
		tplSourcesStore: newTplSourcesStore(cfg.SettingsFile()),
	}
}

// CheckTarget 预检目标路径（已存在时必须是空目录，防覆盖既有内容）。
// 由 cmd 层在 OpenSource/交互之前调用——clone、交互都可能耗时，不能跑完才发现路径非法。
func (s *Service) CheckTarget(targetPath string) error {
	return validateTarget(targetPath)
}

// OpenSource 解析模板源引用为可用源：本地绝对路径直读（开发态）/ 名字查注册表
// 临时 clone（发布态）。同时加载模板清单，非法源结构在此时报错（先于任何交互）。
func (s *Service) OpenSource(sourceRef string) (*Source, error) {
	dir, cleanup, err := loadSourceDir(sourceRef, s.TplSources())
	if err != nil {
		return nil, err
	}

	// 加载模板列表
	templates, rootTpl, err := LoadSourceTemplates(dir)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}

	return &Source{dir: dir, cleanup: cleanup, templates: templates, rootTpl: rootTpl}, nil
}

// Generate 渲染模板到 targetPath（vars 需已收集齐）并执行 init 命令，返回生成文件数。
// 模板选择与变量收集的交互在 cmd 层编排，domain 不做用户交互。
func (s *Service) Generate(tpl *Template, targetPath string, vars map[string]string) (int, error) {
	// 写入目标目录
	count, err := Render(tpl, targetPath, vars)
	if err != nil {
		return count, err
	}

	// 执行初始化
	if err := runInit(tpl, targetPath, vars); err != nil {
		return count, err
	}

	return count, nil
}

// --- 模板源配置（settings.json tplSources 节，forge 同款形态） ---

// TplSources 读全部模板源（直读不缓存）。坏条目（name/repoUrl 非法）跳过不阻断。
func (s *Service) TplSources() []TplSource {
	return s.tplSourcesStore.All()
}

// SaveTplSource 新增或按 name 替换一条模板源（name 是唯一键——编辑 name 等价于删旧存新）。
// 校验收敛在写侧，坏数据中文错误不落文件。
func (s *Service) SaveTplSource(src TplSource) error {
	return s.tplSourcesStore.Save(src)
}

// SaveAllTplSources 以提交清单整节覆盖模板源节（顺序即 create 交互选择序）。
// 逐项校验 + name 查重，任一失败整体不落盘。
func (s *Service) SaveAllTplSources(sources []TplSource) error {
	return s.tplSourcesStore.SaveAll(sources)
}

// DeleteTplSource 按唯一键（name）删除一条模板源；未命中幂等成功。
func (s *Service) DeleteTplSource(src TplSource) error {
	return s.tplSourcesStore.Delete(src)
}
