package create

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"cube/util/oskit"
)

// Template 模板
type Template struct {
	path string
	meta *TemplateMeta
}

func LoadTemplate(tplPath string) (*Template, error) {
	if !oskit.IsDir(tplPath) {
		return nil, fmt.Errorf("模板目录不存在或不是目录: %s", tplPath)
	}

	meteFilePath := filepath.Join(tplPath, MetaFileName)
	if !oskit.IsFile(meteFilePath) {
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

func (t *Template) Path() string        { return t.path }
func (t *Template) Meta() *TemplateMeta { return t.meta }

func (t *Template) WalkFile(fn func(rel string) error) error {
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

func (t *Template) ReadFile(relPath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(t.path, relPath))
}
