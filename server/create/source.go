package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cube/util/git"
	"cube/util/oskit"
	"cube/util/pathkit"
)

// gitUrlPrefixes 触发 git clone 的来源前缀。本地路径（含 ~/ 开头）不在此列。
var gitUrlPrefixes = []string{"https://", "http://", "git://", "ssh://", "file://", "git@"}

// IsGitSource 判定模板来源是否为 git url 形态（本地路径——含 ~/ 前缀——返回 false）。
// 供 cmd 层决定来源是否要展开为绝对路径：本地路径展开，url 原样传给 clone。
func IsGitSource(source string) bool {
	if strings.HasSuffix(source, ".git") {
		return true
	}
	for _, p := range gitUrlPrefixes {
		if strings.HasPrefix(source, p) {
			return true
		}
	}
	return false
}

// ResolveTemplateDir 把来源（本地目录或 git url）解析为「来源根目录」。
// 本地来源须为绝对路径（~/ 与相对路径由 cmd 层展开，domain 不感知进程 cwd 与 home）；
// git 来源 clone --depth 1 到系统临时目录，返回 cleanup 供成功后删除临时目录；
// 失败时 cleanup 为 nil（现场保留，路径已在错误信息中给出，便于排查模板问题）。
func ResolveTemplateDir(source string) (dir string, cleanup func(), err error) {
	if !IsGitSource(source) {
		if info, err := os.Stat(source); err != nil || !info.IsDir() {
			return "", nil, fmt.Errorf("模板来源目录不存在或不是目录: %s", pathkit.PrettyPath(source))
		}
		return source, nil, nil
	}

	tempDir, err := os.MkdirTemp("", "cube-create-")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	if err := git.Clone(tempDir, source, 1, ""); err != nil {
		return "", nil, fmt.Errorf("clone 模板仓库失败（临时目录 %s 保留供排查）: %w", tempDir, err)
	}
	return tempDir, func() { _ = os.RemoveAll(tempDir) }, nil
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
