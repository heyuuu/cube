package template

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/heyuuu/x/xstd/xmaps"
	"github.com/heyuuu/x/xstd/xos"

	"cube/pkg/git"
)

// DefaultSourceName 缺省模板源名（免配置即可 create 的体验兜底）。
const DefaultSourceName = "default"

// RootTplName 根模板形态的模板名（template.yaml 在源根时整个源即一个模板，名固定，无选择语义）。
const RootTplName = "<root>"

// Source 已解析的可用模板源（本地路径直读或临时 clone 的仓库），含模板清单。
// 用完须调用 Cleanup（临时 clone 目录删除；本地路径形态为空操作，可无条件 defer）。
type Source struct {
	dir       string
	cleanup   func()
	templates map[string]string // 模板名 → 模板目录（loadSourceTemplates 口径）
	rootTpl   bool              // 根模板形态（template.yaml 在源根，整个源即唯一模板）
}

func newSingleSource(dir string) *Source {
	return &Source{
		dir:       dir,
		templates: map[string]string{RootTplName: dir},
		rootTpl:   true,
	}
}

func newLibrarySource(dir string, templates map[string]string) *Source {
	return &Source{
		dir:       dir,
		templates: maps.Clone(templates),
		rootTpl:   false,
	}
}

func (s *Source) setCleanup(cleanup func()) {
	s.cleanup = cleanup
}

// Cleanup 释放源占用的资源；幂等，本地路径形态为空操作。
func (s *Source) Cleanup() {
	if s.cleanup != nil {
		s.cleanup()
		s.cleanup = nil
	}
}

// Names 返回源内全部模板名（字典序，交互选择候选用）。
func (s *Source) Names() []string {
	return xmaps.SortedKeys(s.templates)
}

// RootTemplateName 根模板名；模板库形态返回空串。cmd 层据此在未指名模板时
// 直接选中根模板、跳过交互——自动选中挂在源的结构声明上（template.yaml 在根），
// 不挂在模板数量上：模板库只有一个模板是暂时状态，库增长不应改变交互行为。
func (s *Source) RootTemplateName() string {
	if s.rootTpl {
		return RootTplName
	}
	return ""
}

// Template 按名加载模板；不存在时报错附可用清单。
func (s *Source) Template(name string) (*Template, error) {
	tplPath, ok := s.templates[name]
	if !ok {
		return nil, fmt.Errorf("模板 %q 不存在，可用模板: %v", name, strings.Join(s.Names(), ", "))
	}
	return LoadTemplate(tplPath)
}

// 根据 sourceRef 构建模板源对象
// sourceRef 两种形态：本地绝对路径直读（开发态，不 clone）；注册名查注册表后临时 clone（发布态）。
func openSource(sourceRef string, sources []TplSource) (*Source, error) {
	// 本地绝对路径：直接使用，不产生需要清理的资源
	if filepath.IsAbs(sourceRef) {
		return openDirSource(sourceRef)
	}

	// 注册名：查 tplSources 注册表拿 repoURL
	idx := slices.IndexFunc(sources, func(s TplSource) bool { return s.Name == sourceRef })
	if idx < 0 {
		return nil, fmt.Errorf("未找到指定模板源: %s（可在设置页「模板源」配置）", sourceRef)
	}
	return openRepoSource(sources[idx].RepoURL)
}

// 将指定目录作为 source 加载模板
// 根模板形态：template.yaml 在源根，整个源即唯一模板（名固定 RootTplName，源根即模板目录）；
// 模板库形态：templates/ 下各含 template.yaml 的子目录为模板，目录名为模板名。
func openDirSource(dir string) (*Source, error) {
	if !xos.IsDir(dir) {
		return nil, fmt.Errorf("模板源路径不存在或不为文件夹: %s", dir)
	}

	// 根模板形态
	if xos.IsFile(filepath.Join(dir, MetaFileName)) {
		return newSingleSource(dir), nil
	}

	// 模板库形态 (默认 templates/* 为各模板地址)
	tplRoot := filepath.Join(dir, "templates")
	if !xos.IsDir(tplRoot) {
		return nil, fmt.Errorf("%s 不是合法模板来源：根目录无 template.yaml，也无 templates/ 子目录", dir)
	}

	// 读取子目录
	entries, err := os.ReadDir(tplRoot)
	if err != nil {
		return nil, fmt.Errorf("读取 templates/ 目录失败: %w", err)
	}

	// 遍历并记录符合条件的路径
	templates := make(map[string]string)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// 通过 MetaFileName 是否存在判断是否为 template 目录
		tplPath := filepath.Join(tplRoot, e.Name())
		if !xos.IsFile(filepath.Join(tplPath, MetaFileName)) {
			continue
		}
		// 添加到结果
		templates[e.Name()] = tplPath
	}
	return newLibrarySource(dir, templates), nil
}

// 按指定 repoURL 加载模板
func openRepoSource(repoURL string) (*Source, error) {
	dir, cleanup, err := cloneSource(repoURL)
	if err != nil {
		return nil, err
	}

	src, err := openDirSource(dir)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}

	// 挂载 cleanup
	src.setCleanup(cleanup)
	return src, nil
}

// cloneSource 浅克隆模板仓库到系统临时目录，cleanup 删除整个临时目录（clone 失败时自清理）。
func cloneSource(repoURL string) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "cube-tpl-")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	if err := git.Clone(dir, repoURL, 1, ""); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("拉取模板源失败: url=%s err=%w", repoURL, err)
	}
	return dir, cleanup, nil
}
