package template

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/heyuuu/x/xstd/xos"

	"cube/pkg/git"
)

// Template 模板
type Template struct {
	path string
	meta *Meta
}

// LoadTemplate 加载模板目录：校验并解析根目录 template.yaml。
func LoadTemplate(tplPath string) (*Template, error) {
	if !xos.IsDir(tplPath) {
		return nil, fmt.Errorf("模板目录不存在或不是目录: %s", tplPath)
	}

	meteFilePath := filepath.Join(tplPath, MetaFileName)
	if !xos.IsFile(meteFilePath) {
		return nil, fmt.Errorf("模板 %s 文件不存在或非文件: %s", MetaFileName, meteFilePath)
	}

	data, err := os.ReadFile(meteFilePath)
	if err != nil {
		return nil, fmt.Errorf("读取模板 %s 文件失败: tplPath=%s, err=%w", MetaFileName, meteFilePath, err)
	}

	meta, err := InitTemplateMeta(data)
	if err != nil {
		return nil, fmt.Errorf("解析模板 %s 文件失败: tplPath=%s, err=%w", MetaFileName, meteFilePath, err)
	}

	return &Template{
		path: tplPath,
		meta: meta,
	}, nil
}

// Path 返回模板根目录路径。
func (t *Template) Path() string { return t.path }

// Meta 返回模板协议声明。
func (t *Template) Meta() *Meta { return t.meta }

// WalkFile 遍历模板内文件（以相对路径回调，跳过 .git 与 template.yaml，目录不回调）。
// 模板目录在 git 仓库内时按 git 视角取文件（tracked + 未跟踪未忽略，忽略链整条
// 生效），与 clone 发布态的内容口径对齐——node_modules/构建产物等不进生成结果；
// 仓库外目录（如 zip 解压的裸模板）降级普通遍历。
func (t *Template) WalkFile(fn func(rel string) error) error {
	files, err := git.ListFilesUnder(t.path)
	if err != nil {
		slog.Warn("git 列举模板文件失败，降级普通遍历", "path", t.path, "err", err)
	}
	if files != nil {
		for _, abs := range files {
			rel, err := filepath.Rel(t.path, abs)
			if err != nil {
				return err
			}
			// ls-files 不会输出 .git 内部文件，meta 过滤与 walkDir 同口径（任意层级同名文件不算模板内容）
			if filepath.Base(rel) == MetaFileName {
				continue
			}
			if err := fn(rel); err != nil {
				return err
			}
		}
		return nil
	}
	return t.walkDir(fn)
}

// walkDir 降级遍历（目录不在 git 仓库内）：WalkDir 全量，跳过 .git 与 template.yaml。
func (t *Template) walkDir(fn func(rel string) error) error {
	err := filepath.WalkDir(t.path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// 计算相对路径
		rel, err := filepath.Rel(t.path, p)
		if err != nil {
			return err
		}
		// 过滤特殊文件和目录
		base := filepath.Base(p)
		if base == ".git" || base == MetaFileName {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		// 对目录不回调
		if d.IsDir() {
			return nil
		}
		// 文件回调，使用相对路径回调
		return fn(rel)
	})
	return err
}

// ReadFile 按相对路径读取模板内文件。
func (t *Template) ReadFile(relPath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(t.path, relPath))
}

// CheckVars 校验外部传入的变量名（--var）：未在 template.yaml 声明的 key 视为拼写错误。
// 由 cmd 层在交互收集之前调用，让拼写错误先于逐项提问暴露。
func (t *Template) CheckVars(cliVars map[string]string) error {
	varNameSet := make(map[string]struct{}, len(t.meta.Variables))
	for _, v := range t.meta.Variables {
		varNameSet[v.Name] = struct{}{}
	}
	for name := range cliVars {
		if _, exists := varNameSet[name]; !exists {
			return fmt.Errorf("变量 %q 未在 template.yaml 中声明（检查拼写）", name)
		}
	}
	return nil
}
