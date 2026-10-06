package create

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cube/util/oskit"
	"cube/util/slicekit"
)

// ReplaceRule 替换规则
// 和 PatternDecl 同构，但是 Replace 中的 ${var} 插件已经完全替换过了
type ReplaceRule = PatternDecl

// Render 遍历模板目录，把所有文件（路径与内容统一替换后）生成到目标目录。
func Render(templateDir, targetDir string, tpl *TemplateMeta, vars map[string]string) (int, error) {
	allRules, err := interpolatePatterns(tpl.Patterns, vars)
	if err != nil {
		return 0, err
	}

	count := 0
	err = filepath.WalkDir(templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		base := filepath.Base(rel)
		if base == ".git" || base == "template.yaml" && !strings.ContainsRune(rel, '/') {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}

		// 收集命中该文件的所有分组，路径（含文件名）与内容应用同一组规则
		rules := matchRules(allRules, filepath.ToSlash(rel))

		// 计算目标路径
		outPath := filepath.Join(targetDir, applyRules(rel, rules))

		// 计算目标数据
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取模板文件失败: %w", err)
		}
		if !isBinary(data) { // 二进制只做路径替换，内容原样
			data = []byte(applyRules(string(data), rules))
		}

		if err := oskit.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("写入文件失败: %w", err)
		}

		count++
		return nil
	})
	if err != nil {
		return count, fmt.Errorf("生成模板文件失败: %w", err)
	}
	return count, nil
}

// isBinary 粗判二进制文件：内容含 NUL 字节即视为二进制，内容不做替换原样复制。
func isBinary(data []byte) bool {
	return bytes.IndexByte(data, 0) >= 0
}

// matchRules 返回 glob 命中 rel 的所有分组的规则，按分组的 glob 字典序拼接。
func matchRules(rules []ReplaceRule, rel string) []ReplaceRule {
	return slicekit.Filter(rules, func(r ReplaceRule) bool {
		return r.MatchScope(rel)
	})
}

// applyRules 按声明顺序逐条应用精确字符串替换（非正则）。
func applyRules(s string, rules []PatternDecl) string {
	for _, r := range rules {
		s = strings.ReplaceAll(s, r.Pattern, r.Replace)
	}
	return s
}
