package template

import (
	"fmt"
	"log/slog"
	"strings"

	"cube/core/config"
	"cube/util/slicekit"
)

// Service 是模板引擎（cube create）的入口。
// 模板源映射存 settings.json 的 tplSources 节（直读不缓存，保存即生效）。
type Service struct {
	settingsFile  string
	tplSourceRoot string // 本地 source 目录（create 现行链路用；tplSources 接管 create 后移除）
}

func NewService(settingsFile string, tplSourceRoot string) *Service {
	return &Service{
		settingsFile:  settingsFile,
		tplSourceRoot: tplSourceRoot,
	}
}

// Create 生成项目到 targetPath。
func (s *Service) Create(sourceName string, tplName string, targetPath string, cliVars map[string]string) error {
	source, err := LoadSource(sourceName, s.tplSourceRoot)
	if err != nil {
		return err
	}

	return Create(source, tplName, targetPath, cliVars)
}

// --- 模板源配置（settings.json tplSources 节，forge 同款形态） ---

// TplSources 读全部模板源（直读不缓存）。坏条目（name/repoUrl 非法）跳过不阻断。
func (s *Service) TplSources() []TplSource {
	var specs []TplSource
	config.LoadSection(s.settingsFile, tplSourcesSection, &specs)

	sources := make([]TplSource, 0, len(specs))
	for _, src := range specs {
		if err := ValidateTplSource(src); err != nil {
			slog.Warn("tplSources 配置条目非法，跳过", "name", src.Name, "err", err)
			continue
		}
		sources = append(sources, src)
	}
	return sources
}

// SaveTplSource 新增或按 name 替换一条模板源（name 是唯一键——编辑 name 等价于删旧存新）。
// 校验收敛在写侧，坏数据中文错误不落文件。
func (s *Service) SaveTplSource(src TplSource) error {
	src.Name = strings.TrimSpace(src.Name)
	src.RepoUrl = strings.TrimSpace(src.RepoUrl)
	if err := ValidateTplSource(src); err != nil {
		return err
	}

	specs := s.TplSources()
	specs = slicekit.UpsertKeyed(specs, func(cur TplSource) string { return cur.Name }, src)
	return config.SaveSection(s.settingsFile, tplSourcesSection, specs)
}

// DeleteTplSource 按 name 删除一条模板源；不存在时返回中文错误。
func (s *Service) DeleteTplSource(name string) error {
	name = strings.TrimSpace(name)
	rest, removed := slicekit.RemoveKeyed(s.TplSources(), func(cur TplSource) string { return cur.Name }, name)
	if !removed {
		return fmt.Errorf("未找到指定模板源: %s", name)
	}
	return config.SaveSection(s.settingsFile, tplSourcesSection, rest)
}

// ReorderTplSources 按 names 顺序重排模板源节（顺序即 create 交互选择序）。
// 未列出的条目保持原相对顺序排在末尾，不丢数据；未知或重复 name 返回中文错误。
func (s *Service) ReorderTplSources(names []string) error {
	ordered, err := slicekit.ReorderKeyed(
		s.TplSources(), func(cur TplSource) string { return cur.Name }, names,
		"模板源", func(k string) string { return k },
	)
	if err != nil {
		return err
	}
	return config.SaveSection(s.settingsFile, tplSourcesSection, ordered)
}
