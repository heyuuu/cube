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

// Create 生成项目到 targetPath。
// sourceRef 为模板源引用：@语法解析后的名字或本地绝对路径（cmd 层已定形态）。
func (s *Service) Create(sourceRef string, tplName string, targetPath string, cliVars map[string]string) error {
	// 目标路径预检先行——clone、交互都可能耗时，不能让它们跑完才发现路径非法
	if err := validateTarget(targetPath); err != nil {
		return err
	}

	sourcePath, cleanup, err := loadSourceDir(sourceRef, s.TplSources())
	if cleanup != nil {
		defer cleanup() // 临时 clone 目录用后即删（失败路径同样清理）
	}
	if err != nil {
		return err
	}

	return Create(sourcePath, tplName, targetPath, cliVars)
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
