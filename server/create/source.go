package create

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cube/util/git"
	"cube/util/oskit"
	"cube/util/pathkit"
)

const DefaultSource = "core"

type SourceType uint8

const (
	SourceTypeGit   SourceType = iota
	SourceTypeLocal SourceType = iota
)

type Source struct {
	Type SourceType
	Path string
}

func NewSource(typ SourceType, path string) *Source {
	return &Source{Type: typ, Path: path}
}

// LoadSource 获取 sourceName 对应的本地路径，sourceName 可以为名称或本地绝对路径
func LoadSource(sourceName string, tplSourceRoot string) (source *Source, err error) {
	if sourceName == "" {
		return nil, fmt.Errorf("sourceName 不能为空")
	}

	// 若 sourceName 为路径，直接返回
	if strings.Contains(sourceName, string(filepath.Separator)) {
		sourcePath := sourceName
		if !filepath.IsAbs(sourcePath) {
			return nil, errors.New("sourceName 为路径时必须为绝对路径")
		}
		if !oskit.IsDir(sourcePath) {
			return nil, errors.New("sourceName 路径不存在或不为文件夹")
		}
		return NewSource(SourceTypeLocal, sourcePath), nil
	}

	// sourceName 作为名字对应本地路径
	sourcePath := filepath.Join(tplSourceRoot, sourceName)
	if !oskit.IsDir(sourcePath) {
		return nil, fmt.Errorf("sourceName 不存在或不可读: %s, path=%s", sourceName, sourcePath)
	}
	return NewSource(SourceTypeGit, sourcePath), nil
}

// InitGitSource 初始化 git source 到本地目录
func InitGitSource(sourceName string, repoUrl string, tplSourceRoot string) error {
	sourcePath := filepath.Join(tplSourceRoot, sourceName)
	if err := git.Clone(sourcePath, repoUrl, 1, ""); err != nil {
		return err
	}
	return nil
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
