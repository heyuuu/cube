package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cube/util/git"
	"cube/util/oskit"
	"cube/util/pathkit"
)

const DefaultSource = "core"
const DefaultSourceRepoUrl = "git@github.com:heyuuu/cube-templates.git"

// loadSourceDir 解析 sourceRef 为可用的模板源目录，返回目录与 cleanup（非 nil 时用后须调用）。
// sourceRef 两种形态：本地绝对路径直读（开发态，不 clone）；名字查注册表后临时 clone（发布态）。
func loadSourceDir(sourceRef string, sources []TplSource) (dir string, cleanup func(), err error) {
	// 本地绝对路径：直接使用，不产生需要清理的资源
	if filepath.IsAbs(sourceRef) {
		if !oskit.IsDir(sourceRef) {
			return "", nil, fmt.Errorf("模板源路径不存在或不为文件夹: %s", sourceRef)
		}
		return sourceRef, nil, nil
	}

	// 名字：查 tplSources 注册表拿 repoUrl
	src := findTplSource(sources, sourceRef)
	if src == nil {
		return "", nil, missingSourceErr(sourceRef)
	}
	return cloneSource(src.RepoUrl)
}

// findTplSource 在注册表中按 name 查找（name 已在写侧保证唯一）。
func findTplSource(sources []TplSource, name string) *TplSource {
	for i, src := range sources {
		if src.Name == name {
			return &sources[i]
		}
	}
	return nil
}

// missingSourceErr 未注册的源引用错误；缺省源缺配置时附建议地址（来自常量，core 是体验兜底）。
func missingSourceErr(name string) error {
	if name == DefaultSource {
		return fmt.Errorf("未找到指定模板源: %s（可在设置页「模板源」配置，默认源建议: %s）", name, DefaultSourceRepoUrl)
	}
	return fmt.Errorf("未找到指定模板源: %s（可在设置页「模板源」配置）", name)
}

// cloneSource 浅克隆模板仓库到系统临时目录，cleanup 删除整个临时目录（clone 失败时自清理）。
func cloneSource(repoUrl string) (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "cube-tpl-")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	if err := git.Clone(dir, repoUrl, 1, ""); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("拉取模板源失败: url=%s err=%w", repoUrl, err)
	}
	return dir, cleanup, nil
}

// LoadSourceTemplates 加载 source 下的模板，返回 Map<模板名, 模板路径>
func LoadSourceTemplates(sourcePath string) (map[string]string, error) {
	// 单模板 source (根目录即为唯一模板目录)
	if oskit.IsFile(filepath.Join(sourcePath, MetaFileName)) {
		return map[string]string{"default": MetaFileName}, nil
	}
	// 多模板 source (默认 templates/* 为各模板地址)
	tplRoot := filepath.Join(sourcePath, "templates")
	if !oskit.IsDir(tplRoot) {
		return nil, fmt.Errorf("%s 不是合法模板来源：根目录无 template.yaml，也无 templates/ 子目录", pathkit.PrettyPath(sourcePath))
	}

	// 读取子目录
	entries, err := os.ReadDir(tplRoot)
	if err != nil {
		return nil, fmt.Errorf("读取 templates/ 目录失败: %w", err)
	}

	// 遍历并记录符合条件的路径
	result := make(map[string]string)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// 通过 MetaFileName 是否存在判断是否为 template 目录
		tplPath := filepath.Join(tplRoot, e.Name())
		if !oskit.IsFile(filepath.Join(tplPath, MetaFileName)) {
			continue
		}
		// 添加到结果
		result[e.Name()] = tplPath
	}
	return result, nil
}
